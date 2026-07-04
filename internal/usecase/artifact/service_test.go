package artifact

import (
	"context"
	"testing"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
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
	if record.Release.Status != string(release.ReleaseStatusReady) {
		t.Fatalf("release status = %q, want %q", record.Release.Status, release.ReleaseStatusReady)
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
	if record.Bindings[0].MediaType == "" || record.Bindings[0].SizeBytes == 0 || record.Bindings[0].ManifestSchema == "" {
		t.Fatalf("binding metadata = %#v", record.Bindings[0])
	}
	if len(record.Resolutions) != 1 || !record.Resolutions[0].Resolved {
		t.Fatalf("resolutions = %#v", record.Resolutions)
	}
}

func TestListReleasesFiltersByProjectEnvironmentAndStatus(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	releaseA, err := service.CreateRelease(context.Background(), CreateReleaseInput{
		ProjectID: "project-a",
		ActorID:   "tester",
		Definition: ReleaseDefinition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Release",
			Metadata:   ReleaseMetadata{Name: "demo-a"},
			Spec: ReleaseSpec{
				Version:     "1.0.0",
				Application: "app-a",
				Environment: "env-a",
				Artifacts: []ReleaseArtifactSpec{{
					Name:      "demo-a",
					Type:      "image",
					Required:  true,
					Reference: "registry.example.com/team/demo-a@sha256:abcdef",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("create release A: %v", err)
	}
	if _, err := service.CreateRelease(context.Background(), CreateReleaseInput{
		ProjectID: "project-b",
		ActorID:   "tester",
		Definition: ReleaseDefinition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Release",
			Metadata:   ReleaseMetadata{Name: "demo-b"},
			Spec: ReleaseSpec{
				Version:     "1.0.0",
				Application: "app-b",
				Environment: "env-b",
				Artifacts: []ReleaseArtifactSpec{{
					Name:      "demo-b",
					Type:      "image",
					Required:  true,
					Reference: "registry.example.com/team/demo-b@sha256:abcdef",
				}},
			},
		},
	}); err != nil {
		t.Fatalf("create release B: %v", err)
	}
	if _, err := service.CancelRelease(context.Background(), releaseA.Release.ID, "tester"); err != nil {
		t.Fatalf("cancel release A: %v", err)
	}

	filtered, err := service.ListReleases(context.Background(), ListReleasesInput{ProjectID: "project-a", EnvironmentID: "env-a", ApplicationID: "app-a", Status: "canceled"})
	if err != nil {
		t.Fatalf("list releases: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Release.ID != releaseA.Release.ID {
		t.Fatalf("filtered releases = %#v", filtered)
	}
	none, err := service.ListReleases(context.Background(), ListReleasesInput{ProjectID: "project-a", Status: "ready"})
	if err != nil {
		t.Fatalf("list ready releases: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("ready filter should be empty after cancel: %#v", none)
	}
}

func TestArtifactCatalogQueriesReleaseBindings(t *testing.T) {
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
				Required:  true,
				Reference: "registry.example.com/team/demo@sha256:abcdef",
			}},
		},
	}})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}
	artifactID := record.Artifacts[0].ID

	artifacts, err := service.ListArtifacts(context.Background(), ListArtifactsInput{Registry: "registry.example.com", Repository: "team/demo"})
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 1 || artifacts[0].ID != artifactID {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}

	artifact, err := service.GetArtifact(context.Background(), artifactID)
	if err != nil {
		t.Fatalf("get artifact: %v", err)
	}
	if artifact.Reference != "registry.example.com/team/demo@sha256:abcdef" {
		t.Fatalf("artifact = %+v", artifact)
	}

	bindings, err := service.ArtifactReleases(context.Background(), artifactID)
	if err != nil {
		t.Fatalf("artifact releases: %v", err)
	}
	if len(bindings) != 1 || bindings[0].Release.ID != record.Release.ID || bindings[0].Binding.ArtifactID != artifactID {
		t.Fatalf("bindings = %+v", bindings)
	}

	if _, err := service.GetArtifact(context.Background(), "missing"); err == nil {
		t.Fatal("expected missing artifact error")
	}
}

func TestTrackArtifactCreatesStandaloneCatalogRecord(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	artifact, err := service.TrackArtifact(context.Background(), TrackArtifactInput{
		ID:        "artifact-standalone",
		Name:      "tracked-demo",
		Type:      "image",
		Reference: "registry.example.com/team/demo:1.0.0",
		Metadata:  map[string]string{"source": "manual"},
	})
	if err != nil {
		t.Fatalf("track artifact: %v", err)
	}
	if artifact.ID != "artifact-standalone" || artifact.Name != "tracked-demo" {
		t.Fatalf("artifact identity = %#v", artifact)
	}
	if artifact.Reference != "registry.example.com/team/demo:1.0.0" || artifact.Registry != "registry.example.com" || artifact.Repository != "team/demo" {
		t.Fatalf("artifact reference fields = %#v", artifact)
	}
	if artifact.Metadata["source"] != "manual" {
		t.Fatalf("artifact metadata = %#v", artifact.Metadata)
	}

	listed, err := service.ListArtifacts(context.Background(), ListArtifactsInput{Registry: "registry.example.com", Repository: "team/demo"})
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != artifact.ID {
		t.Fatalf("listed artifacts = %#v", listed)
	}
	got, err := service.GetArtifact(context.Background(), artifact.ID)
	if err != nil {
		t.Fatalf("get artifact: %v", err)
	}
	if got.ID != artifact.ID || got.Metadata["source"] != "manual" {
		t.Fatalf("got artifact = %#v", got)
	}
	bindings, err := service.ArtifactReleases(context.Background(), artifact.ID)
	if err != nil {
		t.Fatalf("artifact releases: %v", err)
	}
	if len(bindings) != 0 {
		t.Fatalf("standalone artifact should have no release bindings: %#v", bindings)
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

func TestCreateReleaseBlocksMutableWhenConfigured(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeArtifactProvider{}, fakeEventBus{})
	blockMutable := true
	_, err := service.CreateRelease(context.Background(), CreateReleaseInput{Definition: ReleaseDefinition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Release",
		Metadata:   ReleaseMetadata{Name: "demo"},
		Spec: ReleaseSpec{
			Version: "1.0.0",
			Artifacts: []ReleaseArtifactSpec{{
				Name:         "demo-app",
				Type:         "image",
				Reference:    "nginx:latest",
				BlockMutable: &blockMutable,
			}},
		},
	}})
	if err == nil {
		t.Fatal("expected blockMutable failure")
	}
}

func TestCancelReleaseRecordsEventAndAudit(t *testing.T) {
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
				Reference: "registry.example.com/team/demo@sha256:abcdef",
			}},
		},
	}, ProjectID: "project-a", ActorID: "creator"})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}

	canceled, err := service.CancelRelease(context.Background(), record.Release.ID, "operator")
	if err != nil {
		t.Fatalf("cancel release: %v", err)
	}
	if canceled.Release.Status != "Canceled" || canceled.Release.Metadata["projectId"] != "project-a" || canceled.Release.Metadata["canceledBy"] != "operator" {
		t.Fatalf("canceled release = %#v", canceled.Release)
	}
	if !hasReleaseEvent(canceled.Events, EventReleaseCanceled) {
		t.Fatalf("cancel event missing: %#v", canceled.Events)
	}
	if !hasReleaseAudit(canceled.Audits, "Release canceled") {
		t.Fatalf("cancel audit missing: %#v", canceled.Audits)
	}

	again, err := service.CancelRelease(context.Background(), record.Release.ID, "operator")
	if err != nil {
		t.Fatalf("second cancel should be idempotent: %v", err)
	}
	if len(again.Events) != len(canceled.Events) || len(again.Audits) != len(canceled.Audits) {
		t.Fatalf("second cancel should not duplicate evidence: events %d/%d audits %d/%d", len(again.Events), len(canceled.Events), len(again.Audits), len(canceled.Audits))
	}
}

func TestUpdateReleaseStatusRecordsEventAndAudit(t *testing.T) {
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
				Reference: "registry.example.com/team/demo@sha256:abcdef",
			}},
		},
	}, ActorID: "creator"})
	if err != nil {
		t.Fatalf("create release: %v", err)
	}

	updated, err := service.UpdateReleaseStatus(context.Background(), record.Release.ID, release.ReleaseStatusPlanning, "planner", "release plan created")
	if err != nil {
		t.Fatalf("update release status: %v", err)
	}
	if updated.Release.Status != string(release.ReleaseStatusPlanning) || updated.Release.Metadata["statusUpdatedBy"] != "planner" {
		t.Fatalf("updated release = %#v", updated.Release)
	}
	if !hasReleaseEvent(updated.Events, EventReleaseStatusUpdated) {
		t.Fatalf("status event missing: %#v", updated.Events)
	}
	if !hasReleaseAudit(updated.Audits, "Release status updated") {
		t.Fatalf("status audit missing: %#v", updated.Audits)
	}

	again, err := service.UpdateReleaseStatus(context.Background(), record.Release.ID, release.ReleaseStatusPlanning, "planner", "release plan created")
	if err != nil {
		t.Fatalf("idempotent status update: %v", err)
	}
	if len(again.Events) != len(updated.Events) || len(again.Audits) != len(updated.Audits) {
		t.Fatalf("idempotent status update duplicated evidence: events %d/%d audits %d/%d", len(again.Events), len(updated.Events), len(again.Audits), len(updated.Audits))
	}
}

func hasReleaseEvent(events []event.Event, eventType string) bool {
	for _, evt := range events {
		if evt.Type == eventType {
			return true
		}
	}
	return false
}

func hasReleaseAudit(audits []audit.AuditLog, action string) bool {
	for _, entry := range audits {
		if entry.Action == action {
			return true
		}
	}
	return false
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
		MediaType:                "application/vnd.oci.image.manifest.v1+json",
		SizeBytes:                123,
		ManifestSchema:           "schemaVersion:2",
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
