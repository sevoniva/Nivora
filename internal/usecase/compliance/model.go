package compliance

type AuditSearchInput struct {
	Subject       string `json:"subject,omitempty"`
	ActorID       string `json:"actorId,omitempty"`
	Action        string `json:"action,omitempty"`
	ScopeType     string `json:"scopeType,omitempty"`
	ScopeID       string `json:"scopeId,omitempty"`
	CorrelationID string `json:"correlationId,omitempty"`
}

type EvidenceInput struct {
	SubjectType string
	SubjectID   string
}

type RetentionInput struct {
	ScopeType      string `json:"scopeType,omitempty"`
	ScopeID        string `json:"scopeId,omitempty"`
	LogDays        int    `json:"logDays,omitempty"`
	AuditDays      int    `json:"auditDays,omitempty"`
	EventDays      int    `json:"eventDays,omitempty"`
	EvidenceDays   int    `json:"evidenceDays,omitempty"`
	ImmutableAudit bool   `json:"immutableAudit"`
}
