package audit

import "time"

type AuditLog struct {
	ID            string            `json:"id" yaml:"id"`
	OrgID         string            `json:"orgId,omitempty" yaml:"orgId,omitempty"`
	ActorID       string            `json:"actorId,omitempty" yaml:"actorId,omitempty"`
	Action        string            `json:"action" yaml:"action"`
	Subject       string            `json:"subject" yaml:"subject"`
	SubjectType   string            `json:"subjectType,omitempty" yaml:"subjectType,omitempty"`
	SubjectID     string            `json:"subjectId,omitempty" yaml:"subjectId,omitempty"`
	ScopeType     string            `json:"scopeType,omitempty" yaml:"scopeType,omitempty"`
	ScopeID       string            `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Reason        string            `json:"reason,omitempty" yaml:"reason,omitempty"`
	RequestID     string            `json:"requestId,omitempty" yaml:"requestId,omitempty"`
	CorrelationID string            `json:"correlationId,omitempty" yaml:"correlationId,omitempty"`
	PreviousHash  string            `json:"previousHash,omitempty" yaml:"previousHash,omitempty"`
	RecordHash    string            `json:"recordHash,omitempty" yaml:"recordHash,omitempty"`
	Before        map[string]string `json:"before,omitempty" yaml:"before,omitempty"`
	After         map[string]string `json:"after,omitempty" yaml:"after,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt" yaml:"createdAt"`
}
