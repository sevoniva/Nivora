package credential

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	"github.com/sevoniva/nivora/internal/domain/event"
)

const (
	EventSecretCreated       = "devops.secret.created"
	EventSecretDeleted       = "devops.secret.deleted"
	EventCredentialCreated   = "devops.credential.created"
	EventCredentialValidated = "devops.credential.validated"
	EventSecretUsed          = "devops.secret.used"
)

type SecretCreateInput struct {
	Name      string            `json:"name" yaml:"name"`
	ScopeType string            `json:"scopeType" yaml:"scopeType"`
	ScopeID   string            `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Provider  string            `json:"provider,omitempty" yaml:"provider,omitempty"`
	Key       string            `json:"key,omitempty" yaml:"key,omitempty"`
	Value     string            `json:"value,omitempty" yaml:"value,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	ActorID   string            `json:"actorId,omitempty" yaml:"actorId,omitempty"`
}

type CredentialCreateInput struct {
	Name      string                     `json:"name" yaml:"name"`
	Type      string                     `json:"type" yaml:"type"`
	ScopeType string                     `json:"scopeType" yaml:"scopeType"`
	ScopeID   string                     `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	SecretRef domaincredential.SecretRef `json:"secretRef" yaml:"secretRef"`
	Metadata  map[string]string          `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	ActorID   string                     `json:"actorId,omitempty" yaml:"actorId,omitempty"`
}

type CredentialValidationResult struct {
	CredentialID string    `json:"credentialId"`
	Valid        bool      `json:"valid"`
	Message      string    `json:"message,omitempty"`
	ValidatedAt  time.Time `json:"validatedAt"`
}

type Record struct {
	Credentials []domaincredential.Credential  `json:"credentials,omitempty"`
	SecretRefs  []domaincredential.SecretRef   `json:"secretRefs,omitempty"`
	Usages      []domaincredential.SecretUsage `json:"usages,omitempty"`
	Events      []event.Event                  `json:"events,omitempty"`
	Audits      []audit.AuditLog               `json:"auditLogs,omitempty"`
}
