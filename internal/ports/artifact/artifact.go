package artifact

import (
	"context"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
)

type CredentialRef struct {
	ID        string
	SecretKey string
}

type OCIRegistryConfig struct {
	Name          string
	Type          string
	Endpoint      string
	Insecure      bool
	CredentialRef CredentialRef
	Capabilities  Capabilities
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
	ResolveDigest(ctx context.Context, name string, reference string) (domainartifact.Resolution, error)
	InspectReference(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error)
	Capabilities() Capabilities
}
