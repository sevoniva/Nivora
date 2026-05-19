package kms

import (
	"context"
	"errors"

	"github.com/sevoniva/nivora/internal/domain/credential"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

var ErrKMSProviderNotImplemented = errors.New("cloud KMS secret provider is a Phase 7.1 placeholder")

type Provider struct {
	name string
}

func NewAWS() *Provider     { return &Provider{name: "aws-kms"} }
func NewAliyun() *Provider  { return &Provider{name: "aliyun-kms"} }
func NewTencent() *Provider { return &Provider{name: "tencent-kms"} }

func (p *Provider) ValidateProvider(ctx context.Context) (portsecret.ProviderStatus, error) {
	select {
	case <-ctx.Done():
		return portsecret.ProviderStatus{}, ctx.Err()
	default:
	}
	return portsecret.ProviderStatus{
		Provider:     p.name,
		Configured:   false,
		Reachable:    false,
		Capabilities: []string{"metadata_placeholder"},
		Message:      "cloud KMS provider placeholder; real cloud KMS integration is future work",
	}, nil
}

func (p *Provider) PutSecret(ctx context.Context, request portsecret.PutRequest) (credential.SecretRef, error) {
	return credential.SecretRef{}, ErrKMSProviderNotImplemented
}

func (p *Provider) GetSecret(ctx context.Context, ref credential.SecretRef) ([]byte, error) {
	return nil, ErrKMSProviderNotImplemented
}

func (p *Provider) DeleteSecret(ctx context.Context, ref credential.SecretRef) error {
	return ErrKMSProviderNotImplemented
}

func (p *Provider) RotateSecret(ctx context.Context, ref credential.SecretRef, newValue []byte) (credential.SecretRef, error) {
	return credential.SecretRef{}, ErrKMSProviderNotImplemented
}

func (p *Provider) ListSecretRefs(ctx context.Context, scope portsecret.Scope) ([]credential.SecretRef, error) {
	return nil, ErrKMSProviderNotImplemented
}

func (p *Provider) RecordUsage(ctx context.Context, usage credential.SecretUsage) error {
	return nil
}
