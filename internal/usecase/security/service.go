package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
)

type Service struct {
	store    Store
	scanner  portsecurity.SecurityScanner
	verifier portsecurity.SignatureVerifier
	eventBus eventbus.EventBus
	now      func() time.Time
}

func NewService(store Store, scanner portsecurity.SecurityScanner, verifier portsecurity.SignatureVerifier, bus eventbus.EventBus) *Service {
	return &Service{store: store, scanner: scanner, verifier: verifier, eventBus: bus, now: time.Now}
}

func (s *Service) Scan(ctx context.Context, input ScanInput) (ScanRecord, error) {
	if input.SubjectType == "" {
		return ScanRecord{}, fmt.Errorf("security scan subjectType is required")
	}
	if input.SubjectID == "" {
		input.SubjectID = input.Reference
	}
	if input.SubjectID == "" {
		return ScanRecord{}, fmt.Errorf("security scan subjectId or reference is required")
	}
	if s.scanner == nil {
		return ScanRecord{}, fmt.Errorf("security scanner is not configured")
	}
	now := s.now()
	scan := domainsecurity.SecurityScan{
		ID:          newID("scan"),
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		Status:      domainsecurity.ScanPending,
		CreatedAt:   now,
	}
	record := ScanRecord{Scan: scan}
	if err := s.store.Save(ctx, record); err != nil {
		return ScanRecord{}, err
	}
	record, _ = s.record(ctx, record, EventSecurityScanRequested, "Security scan requested", input.ActorID, "scan requested")
	started := s.now()
	record.Scan.Status = domainsecurity.ScanRunning
	record.Scan.StartedAt = &started
	if err := s.store.Save(ctx, record); err != nil {
		return ScanRecord{}, err
	}
	record, _ = s.record(ctx, record, EventSecurityScanStarted, "Security scan started", input.ActorID, "scan started")
	result, err := s.dispatchScan(ctx, input)
	if err != nil {
		finished := s.now()
		record.Scan.Status = domainsecurity.ScanFailed
		record.Scan.FinishedAt = &finished
		record.Warnings = append(record.Warnings, err.Error())
		_ = s.store.Save(ctx, record)
		record, _ = s.record(ctx, record, EventSecurityScanFailed, "Security scan failed", input.ActorID, err.Error())
		return record, err
	}
	finished := s.now()
	record.Scan.Scanner = result.Scanner
	record.Scan.Status = domainsecurity.ScanSucceeded
	record.Scan.FinishedAt = &finished
	record.Scan.Findings = normalizeFindings(result.Findings)
	record.Scan.Summary = domainsecurity.Summarize(record.Scan.Findings)
	record.Warnings = append(record.Warnings, result.Warnings...)
	record.Policy = s.Evaluate(EvaluateInput{
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		Reference:   input.Reference,
		Findings:    record.Scan.Findings,
		Policy:      input.Policy,
		ActorID:     input.ActorID,
	})
	if input.SubjectType == domainsecurity.SubjectArtifact && s.verifier != nil {
		signature, err := s.verifier.VerifyArtifactSignature(ctx, input.SubjectID)
		if err == nil {
			record.Signature = signature
		}
		record.SBOM = domainsecurity.SBOMRef{
			ArtifactID:  input.SubjectID,
			Format:      "unknown",
			StorageRef:  "memory://" + input.SubjectID + "/sbom",
			GeneratedBy: "noop-sbom-foundation",
			CreatedAt:   s.now(),
		}
	}
	if err := s.store.Save(ctx, record); err != nil {
		return ScanRecord{}, err
	}
	record, _ = s.record(ctx, record, EventSecurityScanCompleted, "Security scan completed", input.ActorID, fmt.Sprintf("scan completed with %d finding(s)", record.Scan.Summary.Total))
	record, _ = s.record(ctx, record, EventPolicyEvaluationStarted, "Policy evaluation started", input.ActorID, "policy evaluation started")
	record, _ = s.record(ctx, record, EventPolicyEvaluationCompleted, "Policy evaluation completed", input.ActorID, string(record.Policy.Decision))
	switch record.Policy.Decision {
	case domainsecurity.GateDeny:
		record, _ = s.record(ctx, record, EventPolicyViolationDetected, "Policy violation detected", input.ActorID, record.Policy.Reason)
		record, _ = s.record(ctx, record, EventPolicyGateDenied, "Policy gate denied", input.ActorID, record.Policy.Reason)
	case domainsecurity.GateWarn:
		record, _ = s.record(ctx, record, EventPolicyGateWarning, "Policy gate warning", input.ActorID, record.Policy.Reason)
	case domainsecurity.GateRequireApproval:
		record, _ = s.record(ctx, record, EventPolicyGateApprovalRequired, "Policy gate requires approval", input.ActorID, record.Policy.Reason)
	default:
		record, _ = s.record(ctx, record, EventPolicyGateAllowed, "Policy gate allowed", input.ActorID, record.Policy.Reason)
	}
	if record.Signature.Subject != "" {
		record, _ = s.record(ctx, record, EventSignatureVerificationCompleted, "Signature verification completed", input.ActorID, string(record.Signature.Result))
	}
	if record.SBOM.StorageRef != "" {
		record, _ = s.record(ctx, record, EventSBOMRecorded, "SBOM reference recorded", input.ActorID, record.SBOM.StorageRef)
	}
	return s.store.Get(ctx, record.Scan.ID)
}

func (s *Service) Evaluate(input EvaluateInput) domainsecurity.PolicyResult {
	policyConfig := input.Policy
	if policyConfig.CriticalDenyThreshold == 0 && policyConfig.HighWarnThreshold == 0 {
		policyConfig = DefaultPolicyConfig()
	}
	summary := domainsecurity.Summarize(input.Findings)
	decision := domainsecurity.GateAllow
	reason := "policy gate allowed"
	if policyConfig.ApprovalOnCritical && summary.Critical > 0 {
		decision = domainsecurity.GateRequireApproval
		reason = "critical findings require approval"
	} else if policyConfig.CriticalDenyThreshold > 0 && summary.Critical >= policyConfig.CriticalDenyThreshold {
		decision = domainsecurity.GateDeny
		reason = "critical findings exceed deny threshold"
	} else if policyConfig.HighWarnThreshold > 0 && summary.High >= policyConfig.HighWarnThreshold {
		decision = domainsecurity.GateWarn
		reason = "high findings exceed warning threshold"
	}
	if warning := referenceWarning(input.Reference, policyConfig); warning != "" && decision == domainsecurity.GateAllow {
		decision = domainsecurity.GateWarn
		reason = warning
	}
	if policyConfig.RequireDigest && input.Reference != "" && !strings.Contains(input.Reference, "@sha256:") {
		decision = domainsecurity.GateDeny
		reason = "artifact digest is required"
	}
	return domainsecurity.PolicyResult{
		ID:          newID("policy"),
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		Decision:    decision,
		Reason:      reason,
		Findings:    append([]domainsecurity.SecurityFinding(nil), input.Findings...),
		EvaluatedAt: s.now(),
	}
}

func (s *Service) Get(ctx context.Context, id string) (ScanRecord, error) {
	return s.store.Get(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]ScanRecord, error) {
	return s.store.List(ctx)
}

func (s *Service) Findings(ctx context.Context, id string) ([]domainsecurity.SecurityFinding, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]domainsecurity.SecurityFinding(nil), record.Scan.Findings...), nil
}

func (s *Service) EvaluateAndStore(ctx context.Context, input EvaluateInput) (domainsecurity.PolicyResult, error) {
	result := s.Evaluate(input)
	return result, nil
}

func (s *Service) dispatchScan(ctx context.Context, input ScanInput) (portsecurity.ScanResult, error) {
	request := portsecurity.ScanRequest{
		SubjectType: input.SubjectType,
		SubjectID:   input.SubjectID,
		Reference:   input.Reference,
		Content:     input.Content,
	}
	switch input.SubjectType {
	case domainsecurity.SubjectArtifact:
		return s.scanner.ScanArtifact(ctx, request)
	case domainsecurity.SubjectManifest:
		result, err := s.scanner.ScanManifest(ctx, request)
		if err != nil {
			return result, err
		}
		result.Findings = append(result.Findings, manifestFindings(input.Content)...)
		return result, nil
	case domainsecurity.SubjectDeploymentPlan, domainsecurity.SubjectRelease:
		return s.scanner.ScanDeploymentPlan(ctx, request)
	default:
		return s.scanner.ScanDeploymentPlan(ctx, request)
	}
}

func (s *Service) record(ctx context.Context, record ScanRecord, eventType string, action string, actorID string, message string) (ScanRecord, error) {
	evt := event.Event{
		ID:              newID("evt"),
		SpecVersion:     "1.0",
		Type:            eventType,
		Source:          "nivora.security",
		Subject:         record.Scan.ID,
		Time:            s.now(),
		DataContentType: "application/json",
		Data: map[string]any{
			"status":  string(record.Scan.Status),
			"message": message,
		},
	}
	record.Events = append(record.Events, evt)
	record.Audits = append(record.Audits, audit.AuditLog{ID: newID("audit"), ActorID: actorID, Action: action, Subject: record.Scan.ID, CreatedAt: s.now()})
	if err := s.store.Save(ctx, record); err != nil {
		return ScanRecord{}, err
	}
	if s.eventBus != nil {
		_ = s.eventBus.Publish(ctx, evt)
	}
	return record, nil
}

func normalizeFindings(findings []domainsecurity.SecurityFinding) []domainsecurity.SecurityFinding {
	out := append([]domainsecurity.SecurityFinding(nil), findings...)
	for i := range out {
		if out[i].ID == "" {
			out[i].ID = newID("finding")
		}
		if out[i].Severity == "" {
			out[i].Severity = domainsecurity.SeverityUnknown
		}
		if out[i].Category == "" {
			out[i].Category = domainsecurity.CategoryPolicy
		}
	}
	return out
}

func referenceWarning(reference string, policyConfig PolicyConfig) string {
	if reference == "" {
		return ""
	}
	inspection, err := domainartifact.InspectReference(reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return ""
	}
	for _, warning := range inspection.Warnings {
		if warning.Code == "mutable_latest_tag" || warning.Code == "tag_without_digest" {
			return warning.Message
		}
	}
	return ""
}

func manifestFindings(content string) []domainsecurity.SecurityFinding {
	var findings []domainsecurity.SecurityFinding
	lower := strings.ToLower(content)
	if strings.Contains(lower, "privileged: true") {
		findings = append(findings, domainsecurity.SecurityFinding{
			Severity:    domainsecurity.SeverityHigh,
			Category:    domainsecurity.CategoryMisconfiguration,
			Target:      "manifest",
			Title:       "Privileged container requested",
			Description: "manifest contains privileged: true",
		})
	}
	if strings.Contains(lower, "hostpath:") {
		findings = append(findings, domainsecurity.SecurityFinding{
			Severity:    domainsecurity.SeverityMedium,
			Category:    domainsecurity.CategoryMisconfiguration,
			Target:      "manifest",
			Title:       "hostPath volume requested",
			Description: "manifest contains hostPath",
		})
	}
	if strings.Contains(lower, "imagepullpolicy: always") && strings.Contains(lower, ":latest") {
		findings = append(findings, domainsecurity.SecurityFinding{
			Severity:    domainsecurity.SeverityMedium,
			Category:    domainsecurity.CategoryPolicy,
			Target:      "manifest",
			Title:       "latest image with Always pull policy",
			Description: "manifest uses imagePullPolicy Always with a latest tag",
		})
	}
	return findings
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
