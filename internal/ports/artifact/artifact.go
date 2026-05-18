package artifact

import "context"

type CredentialRef struct {
	ID        string
	SecretKey string
}

type Artifact struct {
	Name   string
	Tag    string
	Digest string
}

type ArtifactProvider interface {
	ValidateCredential(ctx context.Context, credential CredentialRef) error
	GetArtifact(ctx context.Context, name string, reference string) (Artifact, error)
	ListArtifacts(ctx context.Context, repository string) ([]Artifact, error)
	ResolveDigest(ctx context.Context, name string, reference string) (string, error)
}
