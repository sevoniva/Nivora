package kubernetes

import (
	"context"
	"errors"

	"github.com/sevoniva/nivora/internal/domain/credential"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

var ErrKubernetesClientNotConfigured = errors.New("kubernetes secret provider requires an external Kubernetes client")

type Config struct {
	Context   string
	Namespace string
}

type Provider struct {
	config Config
}

func New(config Config) *Provider {
	return &Provider{config: config}
}

func (p *Provider) ValidateProvider(ctx context.Context) (portsecret.ProviderStatus, error) {
	select {
	case <-ctx.Done():
		return portsecret.ProviderStatus{}, ctx.Err()
	default:
	}
	configured := p.config.Namespace != ""
	return portsecret.ProviderStatus{
		Provider:     "kubernetes",
		Configured:   configured,
		Reachable:    false,
		Capabilities: []string{"get", "put", "rotate", "delete", "usage_audit"},
		Message:      "Kubernetes Secret adapter foundation is configured but no external client is wired in Phase 7.1",
		Metadata:     map[string]string{"namespace": p.config.Namespace},
	}, nil
}

func (p *Provider) PutSecret(ctx context.Context, request portsecret.PutRequest) (credential.SecretRef, error) {
	return credential.SecretRef{}, ErrKubernetesClientNotConfigured
}

func (p *Provider) GetSecret(ctx context.Context, ref credential.SecretRef) ([]byte, error) {
	return nil, ErrKubernetesClientNotConfigured
}

func (p *Provider) DeleteSecret(ctx context.Context, ref credential.SecretRef) error {
	return ErrKubernetesClientNotConfigured
}

func (p *Provider) RotateSecret(ctx context.Context, ref credential.SecretRef, newValue []byte) (credential.SecretRef, error) {
	return credential.SecretRef{}, ErrKubernetesClientNotConfigured
}

func (p *Provider) ListSecretRefs(ctx context.Context, scope portsecret.Scope) ([]credential.SecretRef, error) {
	return nil, ErrKubernetesClientNotConfigured
}

func (p *Provider) RecordUsage(ctx context.Context, usage credential.SecretUsage) error {
	return nil
}
