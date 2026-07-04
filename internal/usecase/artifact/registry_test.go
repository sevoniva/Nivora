package artifact

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	portartifact "github.com/sevoniva/nivora/internal/ports/artifact"
)

func TestArtifactRegistryCatalogCreateUpdateDisable(t *testing.T) {
	service := NewRegistryService(NewRegistryMemoryStore())
	ctx := context.Background()

	created, err := service.Create(ctx, RegistryCreateInput{
		ID:            "areg-local",
		ProjectID:     "project-a",
		Name:          "local-registry",
		Endpoint:      "http://localhost:30500",
		Insecure:      true,
		CredentialRef: "cred-registry",
	})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}
	if created.Type != "oci" || !created.Insecure || !created.Enabled || created.CredentialRef != "cred-registry" {
		t.Fatalf("unexpected registry: %+v", created)
	}

	if _, err := service.Create(ctx, RegistryCreateInput{ProjectID: "project-a", Name: "local-registry", Endpoint: "localhost:30500"}); err == nil {
		t.Fatal("expected duplicate registry error")
	}

	endpoint := "registry.example.com"
	updated, err := service.Update(ctx, created.ID, RegistryUpdateInput{Endpoint: &endpoint})
	if err != nil {
		t.Fatalf("update registry: %v", err)
	}
	if updated.Endpoint != endpoint {
		t.Fatalf("endpoint not updated: %+v", updated)
	}

	disabled, err := service.Disable(ctx, created.ID)
	if err != nil {
		t.Fatalf("disable registry: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("registry should be disabled: %+v", disabled)
	}
}

func TestArtifactRegistryCatalogValidation(t *testing.T) {
	service := NewRegistryService(NewRegistryMemoryStore())
	ctx := context.Background()
	if _, err := service.Create(ctx, RegistryCreateInput{Name: "bad", Endpoint: "http://localhost:30500"}); err == nil {
		t.Fatal("expected insecure http registry to require insecure=true")
	}
	if _, err := service.Create(ctx, RegistryCreateInput{Name: "bad", Type: "harbor", Endpoint: "registry.example.com"}); err == nil {
		t.Fatal("expected unsupported registry type error")
	}
	if _, err := service.Create(ctx, RegistryCreateInput{Endpoint: "registry.example.com"}); err == nil {
		t.Fatal("expected missing name error")
	}
}

func TestArtifactRegistryRepositoryArtifacts(t *testing.T) {
	service := NewRegistryServiceWithProviderFactory(NewRegistryMemoryStore(), func(registry domainartifact.ArtifactRegistry) portartifact.ArtifactProvider {
		return fakeRegistryArtifactProvider{artifacts: []domainartifact.Artifact{{
			Type:       domainartifact.ArtifactTypeImage,
			Name:       "app",
			Version:    "1.0.0",
			Reference:  registry.Endpoint + "/team/app:1.0.0",
			Registry:   registry.Endpoint,
			Repository: "team/app",
		}}}
	})
	ctx := context.Background()
	registry, err := service.Create(ctx, RegistryCreateInput{Name: "local", Endpoint: "registry.example.invalid", CredentialRef: "cred-registry"})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}
	result, err := service.ListRepositoryArtifacts(ctx, RegistryRepositoryListInput{RegistryID: registry.ID, Repository: "team/app"})
	if err != nil {
		t.Fatalf("list repository artifacts: %v", err)
	}
	if result.RegistryID != registry.ID || len(result.Artifacts) != 1 || result.Artifacts[0].Repository != "team/app" {
		t.Fatalf("result = %#v", result)
	}
	if len(result.Warnings) == 0 || !strings.Contains(result.Warnings[0], "CredentialRef") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	if _, err := service.ListRepositoryArtifacts(ctx, RegistryRepositoryListInput{RegistryID: registry.ID, Repository: "team/app", ProjectID: "project-b"}); err == nil || !errors.Is(err, ErrRegistryNotFound) {
		t.Fatalf("cross-project registry listing error = %v, want ErrRegistryNotFound", err)
	}
}

func TestArtifactRegistryRepositoryArtifactsRejectsUnsafeStates(t *testing.T) {
	service := NewRegistryService(NewRegistryMemoryStore())
	ctx := context.Background()
	registry, err := service.Create(ctx, RegistryCreateInput{Name: "disabled", Endpoint: "registry.example.invalid"})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}
	if _, err := service.Disable(ctx, registry.ID); err != nil {
		t.Fatalf("disable registry: %v", err)
	}
	if _, err := service.ListRepositoryArtifacts(ctx, RegistryRepositoryListInput{RegistryID: registry.ID, Repository: "team/app"}); err == nil {
		t.Fatal("expected disabled registry listing to fail")
	}
	registry, err = service.Create(ctx, RegistryCreateInput{Name: "enabled", Endpoint: "registry2.example.invalid"})
	if err != nil {
		t.Fatalf("create enabled registry: %v", err)
	}
	if _, err := service.ListRepositoryArtifacts(ctx, RegistryRepositoryListInput{RegistryID: registry.ID, Repository: "team/app"}); err == nil || !errors.Is(err, ErrRegistryInvalid) {
		t.Fatalf("provider missing error = %v, want ErrRegistryInvalid", err)
	}
}

type fakeRegistryArtifactProvider struct {
	artifacts []domainartifact.Artifact
}

func (f fakeRegistryArtifactProvider) ValidateCredential(ctx context.Context, credential portartifact.CredentialRef) error {
	return ctx.Err()
}

func (f fakeRegistryArtifactProvider) GetArtifact(ctx context.Context, name string, reference string) (domainartifact.Artifact, error) {
	return domainartifact.Artifact{}, ctx.Err()
}

func (f fakeRegistryArtifactProvider) ListArtifacts(ctx context.Context, repository string) ([]domainartifact.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]domainartifact.Artifact(nil), f.artifacts...), nil
}

func (f fakeRegistryArtifactProvider) ResolveDigest(ctx context.Context, name string, reference string) (domainartifact.Resolution, error) {
	return domainartifact.Resolution{}, ctx.Err()
}

func (f fakeRegistryArtifactProvider) InspectReference(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error) {
	return domainartifact.Inspection{}, ctx.Err()
}

func (f fakeRegistryArtifactProvider) Capabilities() portartifact.Capabilities {
	return portartifact.Capabilities{SupportsListing: true}
}
