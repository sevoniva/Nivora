package artifact

import (
	"context"
	"testing"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/event"
	portartifact "github.com/sevoniva/nivora/internal/ports/artifact"
)

func TestCreateReleaseBindsArtifacts(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	record, err := service.CreateRelease(context.Background(), CreateReleaseInput{Definition: ReleaseDefinition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Release",
		Metadata:   ReleaseMetadata{Name: "demo"},
		Spec: ReleaseSpec{
			Version:     "1.0.0",
			Application: "demo-app",
			Artifacts: []ReleaseArtifactSpec{{
				Name:      "demo-app",
				Type:      "image",
				Role:      "primary",
				Required:  true,
				Reference: "registry.example.com/team/demo@sha256:abcdef",
			}},
		},
	}})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if record.Release.ID == "" || len(record.Bindings) != 1 || len(record.Artifacts) != 1 {
		t.Fatalf("record = %#v", record)
	}
	if record.Bindings[0].Digest != "sha256:abcdef" {
		t.Fatalf("digest = %q", record.Bindings[0].Digest)
	}
	if len(record.Events) == 0 || len(record.Audits) == 0 {
		t.Fatalf("events=%d audits=%d", len(record.Events), len(record.Audits))
	}
}

func TestCreateReleaseRecordsMutableWarnings(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	record, err := service.CreateRelease(context.Background(), CreateReleaseInput{Definition: ReleaseDefinition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Release",
		Metadata:   ReleaseMetadata{Name: "demo"},
		Spec: ReleaseSpec{
			Version: "1.0.0",
			Artifacts: []ReleaseArtifactSpec{{
				Name:      "demo-app",
				Type:      "image",
				Reference: "nginx:latest",
			}},
		},
	}})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if len(record.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
	if len(record.Audits) < 3 {
		t.Fatalf("expected warning audit record, got %d audit entries", len(record.Audits))
	}
}

func TestCreateReleaseResolvesDigestWhenRequested(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	record, err := service.CreateRelease(context.Background(), CreateReleaseInput{Definition: ReleaseDefinition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Release",
		Metadata:   ReleaseMetadata{Name: "demo"},
		Spec: ReleaseSpec{
			Version:       "1.0.0",
			ResolveDigest: true,
			RequireDigest: true,
			Artifacts: []ReleaseArtifactSpec{{
				Name:      "demo-app",
				Type:      "image",
				Reference: "registry.example.com/team/demo:1.0.0",
			}},
		},
	}})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	if record.Bindings[0].Digest != "sha256:resolved" || record.Bindings[0].DigestReference == "" {
		t.Fatalf("binding = %#v", record.Bindings[0])
	}
	if len(record.Resolutions) != 1 || !record.Resolutions[0].Resolved {
		t.Fatalf("resolutions = %#v", record.Resolutions)
	}
}

func TestCreateReleaseRequireDigestFailsWhenResolutionUnavailable(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	_, err := service.CreateRelease(context.Background(), CreateReleaseInput{Definition: ReleaseDefinition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Release",
		Metadata:   ReleaseMetadata{Name: "demo"},
		Spec: ReleaseSpec{
			Version:       "1.0.0",
			RequireDigest: true,
			Artifacts: []ReleaseArtifactSpec{{
				Name:      "demo-app",
				Type:      "image",
				Reference: "nginx:1.0.0",
			}},
		},
	}})
	if err == nil {
		t.Fatal("expected requireDigest failure")
	}
}

type fakeArtifactProvider struct{}

func (fakeArtifactProvider) ValidateCredential(ctx context.Context, credential portartifact.CredentialRef) error {
	return ctx.Err()
}

func (fakeArtifactProvider) GetArtifact(ctx context.Context, name string, reference string) (domainartifact.Artifact, error) {
	inspection, err := domainartifact.InspectReference(reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return domainartifact.Artifact{}, err
	}
	return domainartifact.Artifact{Name: name, Reference: inspection.Reference.Normalized, Digest: inspection.Reference.Digest}, nil
}

func (fakeArtifactProvider) ListArtifacts(ctx context.Context, repository string) ([]domainartifact.Artifact, error) {
	return nil, nil
}

func (fakeArtifactProvider) ResolveDigest(ctx context.Context, name string, reference string) (domainartifact.Resolution, error) {
	inspection, err := domainartifact.InspectReference(reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return domainartifact.Resolution{}, err
	}
	digest := inspection.Reference.Digest
	if digest == "" && inspection.Reference.Registry != "" {
		digest = "sha256:resolved"
	}
	return domainartifact.Resolution{
		Reference:                inspection.Reference,
		Digest:                   digest,
		DigestQualifiedReference: domainartifact.DigestQualifiedReference(inspection.Reference, digest),
		Resolved:                 digest != "",
		Warnings:                 inspection.Warnings,
	}, nil
}

func (fakeArtifactProvider) InspectReference(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error) {
	return domainartifact.InspectReference(reference, artifactType)
}

func (fakeArtifactProvider) Capabilities() portartifact.Capabilities {
	return portartifact.Capabilities{}
}

type fakeEventBus struct{}

func (fakeEventBus) Publish(ctx context.Context, evt event.Event) error { return ctx.Err() }

func (fakeEventBus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event)
	close(ch)
	return ch, nil
}
