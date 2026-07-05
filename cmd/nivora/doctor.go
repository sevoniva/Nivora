package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sevoniva/nivora/internal/usecase/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run production-like posture checks without mutating runtime state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd, configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "file", "configs/production.example.yaml", "config file to inspect")
	cmd.AddCommand(newDoctorAreaCommand("config", "Check config safety", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("security", "Check security posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("runtime", "Check runtime posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("database", "Check database/runtime-store posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("runners", "Check runner posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("audit", "Check audit posture", &configPath))
	cmd.AddCommand(newDoctorLiveCommand())
	return cmd
}

func newDoctorLiveCommand() *cobra.Command {
	var serverURL string
	var tokenEnv string
	var auditScopeType string
	var auditScopeID string
	cmd := &cobra.Command{
		Use:   "live",
		Short: "Run read-only live diagnostics against a running Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			report := runDoctorLive(cmd.Context(), serverURL, os.Getenv(tokenEnv), auditScopeType, auditScopeID)
			printJSON(cmd.OutOrStdout(), report)
			if report.Status == doctor.StatusFail {
				return fmt.Errorf("doctor live checks failed")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&tokenEnv, "token-env", "NIVORA_AUTH_TOKEN", "environment variable containing the bearer token")
	cmd.Flags().StringVar(&auditScopeType, "audit-scope-type", "pipeline", "audit scope type for hash-chain verification")
	cmd.Flags().StringVar(&auditScopeID, "audit-scope-id", "", "optional audit scope id for hash-chain verification")
	return cmd
}

func runDoctorLive(ctx context.Context, serverURL string, token string, auditScopeType string, auditScopeID string) doctor.Report {
	diagnostics, diagnosticsErr := doJSONWithToken(ctx, http.MethodGet, serverURL, "/api/v1/system/diagnostics", nil, token)
	recovery, recoveryErr := doJSONWithToken(ctx, http.MethodGet, serverURL, "/api/v1/system/runtime/recovery", nil, token)
	auditPath := "/api/v1/audit/verify?" + doctorLiveAuditQuery(auditScopeType, auditScopeID)
	audit, auditErr := doJSONWithToken(ctx, http.MethodGet, serverURL, auditPath, nil, token)

	report := doctor.Report{Status: doctor.StatusPass}
	report.Checks = append(report.Checks, liveEndpointCheck("live.diagnostics", "database", "system diagnostics endpoint responded", diagnosticsErr))
	report.Checks = append(report.Checks, liveRuntimeModeCheck(diagnostics))
	report.Checks = append(report.Checks, liveDependencyChecks(diagnostics))
	report.Checks = append(report.Checks, liveEndpointCheck("live.runtime_recovery", "runtime", "runtime recovery endpoint responded", recoveryErr))
	report.Checks = append(report.Checks, liveRecoveryStatusCheck(recovery))
	report.Checks = append(report.Checks, liveOutboxCheck(recovery))
	report.Checks = append(report.Checks, liveEndpointCheck("live.audit_verify", "audit", "audit hash-chain verify endpoint responded", auditErr))
	report.Checks = append(report.Checks, liveAuditVerifyCheck(audit))
	report.Status = recomputeDoctorStatus(report.Checks)
	return report
}

func doctorLiveAuditQuery(scopeType string, scopeID string) string {
	values := url.Values{}
	if strings.TrimSpace(scopeType) != "" {
		values.Set("scopeType", strings.TrimSpace(scopeType))
	}
	if strings.TrimSpace(scopeID) != "" {
		values.Set("scopeId", strings.TrimSpace(scopeID))
	}
	return values.Encode()
}

func liveEndpointCheck(id string, area string, okReason string, err error) doctor.Check {
	if err != nil {
		return doctor.Check{ID: id, Area: area, Status: doctor.StatusFail, Reason: err.Error(), Remediation: "verify the server URL, auth token, route availability, and runtime health"}
	}
	return doctor.Check{ID: id, Area: area, Status: doctor.StatusPass, Reason: okReason}
}

func liveRuntimeModeCheck(diagnostics any) doctor.Check {
	runtime := mapField(mapPayload(diagnostics), "runtime")
	mode := strings.TrimSpace(stringField(runtime, "runtime_mode"))
	if mode == "" {
		mode = strings.TrimSpace(stringField(runtime, "runtimeMode"))
	}
	check := doctor.Check{ID: "live.runtime_mode", Area: "runtime", Evidence: mode}
	switch mode {
	case "postgres":
		check.Status = doctor.StatusPass
		check.Reason = "running server reports postgres runtime mode"
	case "in_memory", "memory":
		check.Status = doctor.StatusWarn
		check.Reason = "running server reports memory runtime mode; state is not restart-durable"
		check.Remediation = "use database.runtime_store=postgres for production-like runtime recovery"
	default:
		check.Status = doctor.StatusWarn
		check.Reason = "running server did not report a recognized runtime mode"
		check.Remediation = "check /api/v1/system/runtime and server configuration"
	}
	return check
}

func liveDependencyChecks(diagnostics any) doctor.Check {
	checks, _ := mapPayload(diagnostics)["checks"].([]any)
	if len(checks) == 0 {
		return doctor.Check{ID: "live.dependency_checks", Area: "runtime", Status: doctor.StatusWarn, Reason: "system diagnostics returned no dependency checks", Remediation: "check /api/v1/system/diagnostics response shape"}
	}
	warnings := 0
	degradedCritical := 0
	degraded := 0
	for _, item := range checks {
		record := mapPayload(item)
		status := strings.ToLower(strings.TrimSpace(stringField(record, "status")))
		critical := boolField(record, "critical")
		switch status {
		case "degraded", "failed", "error":
			degraded++
			if critical {
				degradedCritical++
			}
		case "warning", "warn":
			warnings++
		}
	}
	out := doctor.Check{ID: "live.dependency_checks", Area: "runtime", Evidence: fmt.Sprintf("checks=%d warnings=%d degraded=%d critical_degraded=%d", len(checks), warnings, degraded, degradedCritical)}
	if degradedCritical > 0 {
		out.Status = doctor.StatusFail
		out.Reason = "live diagnostics report degraded critical dependencies"
		out.Remediation = "inspect /api/v1/system/diagnostics and recover critical dependencies before proceeding"
		return out
	}
	if warnings > 0 || degraded > 0 {
		out.Status = doctor.StatusWarn
		out.Reason = "live diagnostics report warnings or non-critical degradation"
		out.Remediation = "inspect diagnostics and run the relevant operations runbook"
		return out
	}
	out.Status = doctor.StatusPass
	out.Reason = "live diagnostics dependency checks are healthy"
	return out
}

func liveRecoveryStatusCheck(recovery any) doctor.Check {
	payload := mapPayload(recovery)
	status := strings.ToLower(strings.TrimSpace(stringField(payload, "status")))
	out := doctor.Check{ID: "live.runtime_recovery_status", Area: "runtime", Evidence: status}
	switch status {
	case "healthy", "pass", "ok":
		out.Status = doctor.StatusPass
		out.Reason = "runtime recovery status is healthy"
	case "warning", "warn":
		out.Status = doctor.StatusWarn
		out.Reason = "runtime recovery reports recoverable work or stale state"
		out.Remediation = "inspect safeNextActions and consider runtime reconcile where appropriate"
	case "degraded", "fail", "failed":
		out.Status = doctor.StatusFail
		out.Reason = "runtime recovery reports degraded state"
		out.Remediation = "inspect runtime recovery output before running workers"
	default:
		out.Status = doctor.StatusWarn
		out.Reason = "runtime recovery endpoint did not report a recognized status"
		out.Remediation = "check /api/v1/system/runtime/recovery response shape"
	}
	return out
}

func liveOutboxCheck(recovery any) doctor.Check {
	payload := mapPayload(recovery)
	pending := intField(payload, "pendingOutboxEvents")
	failed := intField(payload, "failedOutboxEvents")
	published := intField(payload, "publishedOutboxEvents")
	out := doctor.Check{ID: "live.event_outbox", Area: "runtime", Evidence: fmt.Sprintf("pending=%d failed=%d published=%d", pending, failed, published)}
	if failed > 0 {
		out.Status = doctor.StatusFail
		out.Reason = "event outbox has failed records"
		out.Remediation = "preserve failed outbox records, recover the event transport, then retry publication"
		return out
	}
	if pending > 0 {
		out.Status = doctor.StatusWarn
		out.Reason = "event outbox has pending records"
		out.Remediation = "confirm workers/event publishing are running before declaring the system drained"
		return out
	}
	out.Status = doctor.StatusPass
	out.Reason = "event outbox has no failed or pending records"
	return out
}

func liveAuditVerifyCheck(audit any) doctor.Check {
	payload := mapPayload(audit)
	valid := boolField(payload, "valid")
	message := stringField(payload, "message")
	if valid {
		return doctor.Check{ID: "live.audit_hash_chain", Area: "audit", Status: doctor.StatusPass, Reason: "audit hash-chain verification passed", Evidence: message}
	}
	firstBroken := stringField(payload, "firstBrokenId")
	if firstBroken == "" {
		firstBroken = stringField(payload, "first_broken_id")
	}
	return doctor.Check{
		ID:          "live.audit_hash_chain",
		Area:        "audit",
		Status:      doctor.StatusFail,
		Reason:      doctorFirstNonEmpty(message, "audit hash-chain verification did not report valid=true"),
		Remediation: "investigate audit integrity before relying on evidence bundles",
		Evidence:    firstBroken,
	}
}

func doctorFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mapPayload(payload any) map[string]any {
	record, _ := payload.(map[string]any)
	if record == nil {
		return map[string]any{}
	}
	return record
}

func mapField(record map[string]any, key string) map[string]any {
	return mapPayload(record[key])
}

func stringField(record map[string]any, key string) string {
	value, _ := record[key].(string)
	return value
}

func boolField(record map[string]any, key string) bool {
	value, _ := record[key].(bool)
	return value
}

func intField(record map[string]any, key string) int {
	switch value := record[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func newDoctorAreaCommand(name string, short string, parentPath *string) *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := configPath
			if path == "" && parentPath != nil {
				path = *parentPath
			}
			report, err := doctor.CheckConfigFile(path)
			if err != nil {
				return err
			}
			report.Checks = filterDoctorChecks(report.Checks, doctorArea(name))
			report.Status = recomputeDoctorStatus(report.Checks)
			printJSON(cmd.OutOrStdout(), report)
			if report.Status == doctor.StatusFail {
				return fmt.Errorf("doctor %s checks failed", name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "file", "", "config file to inspect")
	return cmd
}

func runDoctor(cmd *cobra.Command, path string) error {
	report, err := doctor.CheckConfigFile(path)
	if err != nil {
		return err
	}
	printJSON(cmd.OutOrStdout(), report)
	if report.Status == doctor.StatusFail {
		return fmt.Errorf("doctor checks failed")
	}
	return nil
}

func doctorArea(command string) string {
	return command
}

func filterDoctorChecks(checks []doctor.Check, area string) []doctor.Check {
	out := make([]doctor.Check, 0, len(checks))
	for _, check := range checks {
		if area == "" || check.Area == area || check.ID == "config.validate" {
			out = append(out, check)
		}
	}
	return out
}

func recomputeDoctorStatus(checks []doctor.Check) string {
	status := doctor.StatusPass
	for _, check := range checks {
		if check.Status == doctor.StatusFail {
			return doctor.StatusFail
		}
		if check.Status == doctor.StatusWarn {
			status = doctor.StatusWarn
		}
	}
	return status
}
