package artifact

import (
	"context"
	"testing"
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
