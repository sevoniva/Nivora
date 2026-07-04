package security

import (
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
)

const (
	EventSecurityScanRequested          = "devops.security.scan.requested"
	EventSecurityScanStarted            = "devops.security.scan.started"
	EventSecurityScanCompleted          = "devops.security.scan.completed"
	EventSecurityScanFailed             = "devops.security.scan.failed"
	EventPolicyEvaluationStarted        = "devops.policy.evaluation.started"
	EventPolicyEvaluationCompleted      = "devops.policy.evaluation.completed"
	EventPolicyViolationDetected        = "devops.policy.violation.detected"
	EventPolicyGateAllowed              = "devops.policy.gate.allowed"
	EventPolicyGateDenied               = "devops.policy.gate.denied"
	EventPolicyGateWarning              = "devops.policy.gate.warning"
	EventPolicyGateApprovalRequired     = "devops.policy.gate.approval_required"
	EventSignatureVerificationCompleted = "devops.signature.verification.completed"
	EventSBOMRecorded                   = "devops.sbom.recorded"
)

type PolicyConfig struct {
	CriticalDenyThreshold int  `json:"criticalDenyThreshold" yaml:"criticalDenyThreshold"`
	HighWarnThreshold     int  `json:"highWarnThreshold" yaml:"highWarnThreshold"`
	RequireDigest         bool `json:"requireDigest" yaml:"requireDigest"`
	ApprovalOnCritical    bool `json:"approvalOnCritical" yaml:"approvalOnCritical"`
}

func DefaultPolicyConfig() PolicyConfig {
	return PolicyConfig{CriticalDenyThreshold: 1, HighWarnThreshold: 1}
}

type ScanInput struct {
	SubjectType   domainsecurity.SubjectType `json:"subjectType" yaml:"subjectType"`
	SubjectID     string                     `json:"subjectId" yaml:"subjectId"`
	ProjectID     string                     `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID string                     `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	Reference     string                     `json:"reference,omitempty" yaml:"reference,omitempty"`
	Content       string                     `json:"content,omitempty" yaml:"content,omitempty"`
	PolicyID      string                     `json:"policyId,omitempty" yaml:"policyId,omitempty"`
	PolicyMode    string                     `json:"-" yaml:"-"`
	Policy        PolicyConfig               `json:"policy,omitempty" yaml:"policy,omitempty"`
	ActorID       string                     `json:"actorId,omitempty" yaml:"actorId,omitempty"`
}

type ScanRecord struct {
	Scan      domainsecurity.SecurityScan   `json:"scan"`
	Policy    domainsecurity.PolicyResult   `json:"policy"`
	Signature domainsecurity.SignatureCheck `json:"signature,omitempty"`
	SBOM      domainsecurity.SBOMRef        `json:"sbom,omitempty"`
	Events    []event.Event                 `json:"events,omitempty"`
	Audits    []audit.AuditLog              `json:"audits,omitempty"`
	Warnings  []string                      `json:"warnings,omitempty"`
}

type ListScansInput struct {
	SubjectType   domainsecurity.SubjectType `json:"subjectType,omitempty" yaml:"subjectType,omitempty"`
	SubjectID     string                     `json:"subjectId,omitempty" yaml:"subjectId,omitempty"`
	ProjectID     string                     `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID string                     `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	Status        domainsecurity.ScanStatus  `json:"status,omitempty" yaml:"status,omitempty"`
}

type ListFindingsInput struct {
	ScanID        string                         `json:"scanId,omitempty" yaml:"scanId,omitempty"`
	SubjectType   domainsecurity.SubjectType     `json:"subjectType,omitempty" yaml:"subjectType,omitempty"`
	SubjectID     string                         `json:"subjectId,omitempty" yaml:"subjectId,omitempty"`
	ProjectID     string                         `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID string                         `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	Severity      domainsecurity.Severity        `json:"severity,omitempty" yaml:"severity,omitempty"`
	Category      domainsecurity.FindingCategory `json:"category,omitempty" yaml:"category,omitempty"`
}

type EvaluateInput struct {
	SubjectType domainsecurity.SubjectType       `json:"subjectType" yaml:"subjectType"`
	SubjectID   string                           `json:"subjectId" yaml:"subjectId"`
	Reference   string                           `json:"reference,omitempty" yaml:"reference,omitempty"`
	Findings    []domainsecurity.SecurityFinding `json:"findings,omitempty" yaml:"findings,omitempty"`
	PolicyID    string                           `json:"-" yaml:"-"`
	PolicyMode  string                           `json:"-" yaml:"-"`
	Policy      PolicyConfig                     `json:"policy,omitempty" yaml:"policy,omitempty"`
	ActorID     string                           `json:"actorId,omitempty" yaml:"actorId,omitempty"`
}
