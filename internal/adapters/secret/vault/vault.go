package vault

import (
	"context"
	"errors"

	"github.com/sevoniva/nivora/internal/domain/credential"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

var ErrVaultClientNotConfigured = errors.New("vault secret provider requires an external Vault client")

type Config struct {
	Address string
	Mount   string
}

type Provider struct {
	config Config
}

func New(config Config) *Provider {
	if config.Mount == "" {
		config.Mount = "secret"
	}
	return &Provider{config: config}
}

func (p *Provider) ValidateProvider(ctx context.Context) (portsecret.ProviderStatus, error) {
	select {
	case <-ctx.Done():
		return portsecret.ProviderStatus{}, ctx.Err()
	default:
	}
	configured := p.config.Address != ""
	return portsecret.ProviderStatus{
		Provider:     "vault",
		Configured:   configured,
		Reachable:    false,
		Capabilities: []string{"get", "put", "rotate", "delete", "usage_audit"},
		Message:      "Vault adapter foundation is configured but no external client is wired in Phase 7.1",
		Metadata:     map[string]string{"mount": p.config.Mount},
	}, nil
}

func (p *Provider) PutSecret(ctx context.Context, request portsecret.PutRequest) (credential.SecretRef, error) {
	return credential.SecretRef{}, ErrVaultClientNotConfigured
}

func (p *Provider) GetSecret(ctx context.Context, ref credential.SecretRef) ([]byte, error) {
	return nil, ErrVaultClientNotConfigured
}

func (p *Provider) DeleteSecret(ctx context.Context, ref credential.SecretRef) error {
	return ErrVaultClientNotConfigured
}

func (p *Provider) RotateSecret(ctx context.Context, ref credential.SecretRef, newValue []byte) (credential.SecretRef, error) {
	return credential.SecretRef{}, ErrVaultClientNotConfigured
}

func (p *Provider) ListSecretRefs(ctx context.Context, scope portsecret.Scope) ([]credential.SecretRef, error) {
	return nil, ErrVaultClientNotConfigured
}

func (p *Provider) RecordUsage(ctx context.Context, usage credential.SecretUsage) error {
	return nil
}
