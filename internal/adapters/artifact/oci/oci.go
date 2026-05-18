package oci

import (
	"context"
	"errors"

	"github.com/sevoniva/nivora/internal/ports/artifact"
)

var ErrNotImplemented = errors.New("oci artifact adapter is not implemented")

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) ValidateCredential(ctx context.Context, credential artifact.CredentialRef) error {
	return ctx.Err()
}

func (p *Provider) GetArtifact(ctx context.Context, name string, reference string) (artifact.Artifact, error) {
	return artifact.Artifact{}, ErrNotImplemented
}

func (p *Provider) ListArtifacts(ctx context.Context, repository string) ([]artifact.Artifact, error) {
	return nil, ErrNotImplemented
}

func (p *Provider) ResolveDigest(ctx context.Context, name string, reference string) (string, error) {
	return "", ErrNotImplemented
}
