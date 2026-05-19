package compliance

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
)

const (
	RetentionTargetLogs     = "logs"
	RetentionTargetAudit    = "audit"
	RetentionTargetEvents   = "events"
	RetentionTargetEvidence = "evidence"
)

type EvidenceBundle struct {
	ID               string           `json:"id" yaml:"id"`
	SubjectType      string           `json:"subjectType" yaml:"subjectType"`
	SubjectID        string           `json:"subjectId" yaml:"subjectId"`
	ScopeType        string           `json:"scopeType,omitempty" yaml:"scopeType,omitempty"`
	ScopeID          string           `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Summary          string           `json:"summary" yaml:"summary"`
	Release          any              `json:"release,omitempty" yaml:"release,omitempty"`
	Artifacts        []any            `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Approvals        []any            `json:"approvals,omitempty" yaml:"approvals,omitempty"`
	PolicyResults    []any            `json:"policyResults,omitempty" yaml:"policyResults,omitempty"`
	SecurityFindings []any            `json:"securityFindings,omitempty" yaml:"securityFindings,omitempty"`
	DeploymentPlans  []any            `json:"deploymentPlans,omitempty" yaml:"deploymentPlans,omitempty"`
	LogReferences    []LogReference   `json:"logReferences,omitempty" yaml:"logReferences,omitempty"`
	Events           []event.Event    `json:"events,omitempty" yaml:"events,omitempty"`
	Audits           []audit.AuditLog `json:"audits,omitempty" yaml:"audits,omitempty"`
	GeneratedAt      time.Time        `json:"generatedAt" yaml:"generatedAt"`
}

type LogReference struct {
	ID        string    `json:"id" yaml:"id"`
	SubjectID string    `json:"subjectId" yaml:"subjectId"`
	Stream    string    `json:"stream" yaml:"stream"`
	Sequence  int64     `json:"sequence" yaml:"sequence"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
}

type RetentionPolicy struct {
	ID             string    `json:"id" yaml:"id"`
	ScopeType      string    `json:"scopeType,omitempty" yaml:"scopeType,omitempty"`
	ScopeID        string    `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	LogDays        int       `json:"logDays" yaml:"logDays"`
	AuditDays      int       `json:"auditDays" yaml:"auditDays"`
	EventDays      int       `json:"eventDays" yaml:"eventDays"`
	EvidenceDays   int       `json:"evidenceDays" yaml:"evidenceDays"`
	ImmutableAudit bool      `json:"immutableAudit" yaml:"immutableAudit"`
	UpdatedAt      time.Time `json:"updatedAt" yaml:"updatedAt"`
}

type AuditSearchResult struct {
	Items []audit.AuditLog `json:"items" yaml:"items"`
	Count int              `json:"count" yaml:"count"`
}
