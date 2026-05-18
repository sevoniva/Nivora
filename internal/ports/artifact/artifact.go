package artifact

import (
	"context"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
)

type CredentialRef struct {
	ID        string
	SecretKey string
}

type Capabilities struct {
	SupportsDigestResolution     bool `json:"supportsDigestResolution"`
	SupportsListing              bool `json:"supportsListing"`
	SupportsCredentialValidation bool `json:"supportsCredentialValidation"`
}

type ArtifactProvider interface {
	ValidateCredential(ctx context.Context, credential CredentialRef) error
	GetArtifact(ctx context.Context, name string, reference string) (domainartifact.Artifact, error)
	ListArtifacts(ctx context.Context, repository string) ([]domainartifact.Artifact, error)
	ResolveDigest(ctx context.Context, name string, reference string) (string, error)
	InspectReference(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error)
	Capabilities() Capabilities
}
