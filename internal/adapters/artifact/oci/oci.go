package oci

import (
	"context"
	"errors"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
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

func (p *Provider) GetArtifact(ctx context.Context, name string, reference string) (domainartifact.Artifact, error) {
	inspection, err := p.InspectReference(ctx, reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return domainartifact.Artifact{}, err
	}
	return domainartifact.Artifact{
		Type:       inspection.Reference.Type,
		Name:       name,
		Version:    inspection.Reference.Version,
		Reference:  inspection.Reference.Normalized,
		Digest:     inspection.Reference.Digest,
		Registry:   inspection.Reference.Registry,
		Repository: inspection.Reference.Repository,
		CreatedAt:  time.Now(),
	}, nil
}

func (p *Provider) ListArtifacts(ctx context.Context, repository string) ([]domainartifact.Artifact, error) {
	return nil, ErrNotImplemented
}

func (p *Provider) ResolveDigest(ctx context.Context, name string, reference string) (string, error) {
	inspection, err := p.InspectReference(ctx, reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return "", err
	}
	if inspection.Reference.Digest != "" {
		return inspection.Reference.Digest, nil
	}
	return "", ErrNotImplemented
}

func (p *Provider) InspectReference(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error) {
	select {
	case <-ctx.Done():
		return domainartifact.Inspection{}, ctx.Err()
	default:
	}
	return domainartifact.InspectReference(reference, artifactType)
}

func (p *Provider) Capabilities() artifact.Capabilities {
	return artifact.Capabilities{
		SupportsDigestResolution:     false,
		SupportsListing:              false,
		SupportsCredentialValidation: false,
	}
}
