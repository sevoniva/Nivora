package secret

import (
	"context"

	"github.com/sevoniva/nivora/internal/domain/credential"
)

type Provider interface {
	ValidateProvider(ctx context.Context) (ProviderStatus, error)
	PutSecret(ctx context.Context, request PutRequest) (credential.SecretRef, error)
	GetSecret(ctx context.Context, ref credential.SecretRef) ([]byte, error)
	DeleteSecret(ctx context.Context, ref credential.SecretRef) error
	RotateSecret(ctx context.Context, ref credential.SecretRef, newValue []byte) (credential.SecretRef, error)
	ListSecretRefs(ctx context.Context, scope Scope) ([]credential.SecretRef, error)
	RecordUsage(ctx context.Context, usage credential.SecretUsage) error
}

type PutRequest struct {
	Ref   credential.SecretRef
	Value []byte
}

type Scope struct {
	ScopeType string
	ScopeID   string
}

type ProviderStatus struct {
	Provider     string            `json:"provider"`
	Configured   bool              `json:"configured"`
	Reachable    bool              `json:"reachable"`
	Capabilities []string          `json:"capabilities,omitempty"`
	Message      string            `json:"message,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}
