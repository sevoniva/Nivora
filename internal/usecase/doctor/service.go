package doctor

import (
	"fmt"
	"os"
	"strings"

	"github.com/sevoniva/nivora/internal/infra/config"
	"gopkg.in/yaml.v3"
)

const (
	StatusPass       = "PASS"
	StatusWarn       = "WARN"
	StatusFail       = "FAIL"
	StatusNotChecked = "NOT_CHECKED"
)

type Check struct {
	ID          string `json:"id"`
	Area        string `json:"area"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation,omitempty"`
	Evidence    string `json:"evidence,omitempty"`
}

type Report struct {
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}

func CheckConfigFile(path string) (Report, error) {
	if strings.TrimSpace(path) == "" {
		return Report{}, fmt.Errorf("config path is required")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return Report{}, fmt.Errorf("read config %q: %w", path, err)
	}
	cfg := config.Default()
	if err := yaml.Unmarshal(body, &cfg); err != nil {
		return Report{}, fmt.Errorf("decode config %q: %w", path, err)
	}
	return CheckConfig(cfg), nil
}

func CheckConfig(cfg config.Config) Report {
	report := Report{Status: StatusPass}
	report.add(validationCheck(cfg))
	production := cfg.Env == "production" || cfg.Env == "prod"
	report.add(checkBool("config.environment", "config", statusFor(production, StatusPass, StatusWarn), "production-like checks are strict only when environment is production", "set environment: production for production-like installs", cfg.Env))
	report.add(checkBool("database.runtime_store", "runtime", statusFor(!production || cfg.Database.RuntimeStore == "postgres", StatusPass, StatusFail), "runtime store should be postgres for production-like installs", "set database.runtime_store: postgres", cfg.Database.RuntimeStore))
	report.add(checkBool("auth.enabled", "security", statusFor(!production || cfg.Auth.Enabled, StatusPass, StatusFail), "auth must be enabled in production-like installs", "set auth.enabled: true and use token or OIDC", fmt.Sprintf("%t", cfg.Auth.Enabled)))
	report.add(checkBool("auth.mode", "security", statusFor(!production || (cfg.Auth.Mode != "" && cfg.Auth.Mode != "dev" && cfg.Auth.Mode != "disabled"), StatusPass, StatusFail), "production auth mode must not be dev or disabled", "use token or OIDC mode with secrets from env/config refs", cfg.Auth.Mode))
	report.add(checkBool("auth.static_token_env", "security", statusFor(!production || cfg.Auth.Mode != "token" || cfg.Auth.StaticTokenEnv != "", StatusPass, StatusFail), "token auth requires an environment variable reference in production-like installs", "set auth.static_token_env to an environment variable name, not an inline token", cfg.Auth.StaticTokenEnv))
	report.add(checkBool("runtime.allow_local_shell_executor", "runtime", statusFor(!production || !cfg.Runtime.AllowLocalShellExecutor, StatusPass, StatusFail), "local shell executor is not a sandbox", "disable local shell executor in production or isolate runners externally", fmt.Sprintf("%t", cfg.Runtime.AllowLocalShellExecutor)))
	report.add(checkBool("runtime.allow_privileged_executor", "runtime", statusFor(!production || !cfg.Runtime.AllowPrivilegedExecutor, StatusPass, StatusFail), "privileged executor is unsafe by default", "keep privileged executor disabled", fmt.Sprintf("%t", cfg.Runtime.AllowPrivilegedExecutor)))
	report.add(checkBool("runtime.allow_kubernetes_apply", "runtime", statusFor(!production || !cfg.Runtime.AllowKubernetesApply, StatusPass, StatusFail), "Kubernetes apply must stay explicitly guarded", "keep apply disabled unless a separate guarded deployment target enables it", fmt.Sprintf("%t", cfg.Runtime.AllowKubernetesApply)))
	report.add(checkBool("runtime.allow_argo_sync", "runtime", statusFor(!production || !cfg.Runtime.AllowArgoSync, StatusPass, StatusFail), "Argo CD sync must stay explicitly guarded", "keep global Argo sync disabled", fmt.Sprintf("%t", cfg.Runtime.AllowArgoSync)))
	report.add(checkBool("runtime.allow_remote_host_deploy", "runtime", statusFor(!production || !cfg.Runtime.AllowRemoteHostDeploy, StatusPass, StatusFail), "remote host deploy must stay explicitly guarded", "keep global remote host deploy disabled", fmt.Sprintf("%t", cfg.Runtime.AllowRemoteHostDeploy)))
	report.add(checkBool("runtime.allow_insecure_registry", "runtime", statusFor(!production || !cfg.Runtime.AllowInsecureRegistry, StatusPass, StatusFail), "insecure registries must be explicit per registry", "keep global insecure registry disabled", fmt.Sprintf("%t", cfg.Runtime.AllowInsecureRegistry)))
	report.add(checkBool("mcp.action_tools", "mcp", statusFor(!cfg.MCP.AllowActionTools, StatusPass, StatusFail), "MCP action tools are not allowed in this foundation phase", "keep mcp.allow_action_tools: false", fmt.Sprintf("%t", cfg.MCP.AllowActionTools)))
	report.add(checkBool("event_bus.type", "runtime", statusFor(cfg.EventBus.Type != "", StatusPass, StatusFail), "event bus type must be explicit", "set event_bus.type", cfg.EventBus.Type))
	report.add(checkBool("object_store.type", "runtime", statusFor(cfg.ObjectStore.Type != "", StatusPass, StatusFail), "object store type must be explicit", "set object_store.type", cfg.ObjectStore.Type))
	report.add(Check{ID: "database.connectivity", Area: "database", Status: StatusNotChecked, Reason: "local config doctor does not open database connections", Remediation: "use /readyz, /api/v1/system/diagnostics, or the database runbook script against a running server"})
	report.add(Check{ID: "database.migrations", Area: "database", Status: StatusNotChecked, Reason: "migration status requires a live database", Remediation: "run migrations and migration validation in the release pipeline"})
	report.add(Check{ID: "runner.heartbeat_freshness", Area: "runners", Status: StatusNotChecked, Reason: "runner heartbeat freshness requires live runtime state", Remediation: "use nivora runner list, runtime recovery status, or the offline-runner runbook"})
	report.add(Check{ID: "audit.hash_chain", Area: "audit", Status: StatusNotChecked, Reason: "audit hash-chain verification requires stored audit records", Remediation: "use nivora audit verify against a running server"})
	report.finalize()
	return report
}

func validationCheck(cfg config.Config) Check {
	if err := cfg.Validate(); err != nil {
		return Check{
			ID:          "config.validate",
			Area:        "config",
			Status:      StatusFail,
			Reason:      err.Error(),
			Remediation: "fix the config validation error before using this profile",
		}
	}
	return Check{ID: "config.validate", Area: "config", Status: StatusPass, Reason: "config validates"}
}

func checkBool(id string, area string, status string, reason string, remediation string, evidence string) Check {
	return Check{ID: id, Area: area, Status: status, Reason: reason, Remediation: remediation, Evidence: redactEvidence(evidence)}
}

func statusFor(ok bool, good string, bad string) string {
	if ok {
		return good
	}
	return bad
}

func (r *Report) add(check Check) {
	r.Checks = append(r.Checks, check)
}

func (r *Report) finalize() {
	status := StatusPass
	for _, check := range r.Checks {
		switch check.Status {
		case StatusFail:
			r.Status = StatusFail
			return
		case StatusWarn:
			status = StatusWarn
		}
	}
	r.Status = status
}

func redactEvidence(value string) string {
	lower := strings.ToLower(value)
	for _, marker := range []string{"password", "token", "secret", "private_key", "authorization", "kubeconfig", "access_key", "bearer"} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	if strings.Contains(value, "://") && strings.Contains(value, "@") {
		return "[redacted]"
	}
	return value
}
