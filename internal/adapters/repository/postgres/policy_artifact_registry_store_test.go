package postgres

import (
	"context"
	"strings"
	"testing"

	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
)

func TestPolicyAndArtifactRegistryStoresImplementInterfaces(t *testing.T) {
	var _ artifactusecase.RegistryStore = (*ArtifactRegistryStore)(nil)
	var _ policyusecase.Store = (*PolicyStore)(nil)
}

func TestPolicyArtifactRegistryMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000011_policy_artifact_registry_catalog.up.sql")
	down := readMigration(t, "000011_policy_artifact_registry_catalog.down.sql")

	for _, table := range []string{
		"catalog_artifact_registries",
		"catalog_policies",
		"catalog_policy_attachments",
	} {
		if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("up migration missing table %s", table)
		}
		if !strings.Contains(down, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("down migration missing table %s", table)
		}
	}

	for _, index := range []string{
		"idx_catalog_artifact_registries_project",
		"idx_catalog_policies_project",
		"idx_catalog_policy_attachments_policy",
		"idx_catalog_policy_attachments_unique_scope",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
	}
}

func TestPostgresIntegrationPolicyAndArtifactRegistryRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()

	registries := artifactusecase.NewRegistryService(NewArtifactRegistryStore(db.pool))
	registry, err := registries.Create(ctx, artifactusecase.RegistryCreateInput{
		ProjectID:     "project-a",
		Name:          "local-oci",
		Type:          "oci",
		Endpoint:      "registry.example.invalid/team",
		CredentialRef: "cred-registry",
		Capabilities:  []string{"inspect_manifest", "resolve_digest"},
		Labels:        map[string]string{"scope": "test"},
	})
	if err != nil {
		t.Fatalf("create artifact registry: %v", err)
	}

	policies := policyusecase.NewService(NewPolicyStore(db.pool))
	policy, err := policies.Create(ctx, policyusecase.CreateInput{
		ProjectID:     "project-a",
		EnvironmentID: "prod",
		Name:          "digest-required",
		RequireDigest: true,
		Labels:        map[string]string{"gate": "release"},
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	attachment, err := policies.Attach(ctx, policy.ID, policyusecase.AttachInput{ScopeType: "environment", ScopeID: "prod", Metadata: map[string]string{"source": "test"}})
	if err != nil {
		t.Fatalf("attach policy: %v", err)
	}

	restartedPool := db.restart(t)
	registries = artifactusecase.NewRegistryService(NewArtifactRegistryStore(restartedPool))
	policies = policyusecase.NewService(NewPolicyStore(restartedPool))

	loadedRegistry, err := registries.Get(ctx, registry.ID)
	if err != nil {
		t.Fatalf("reload registry: %v", err)
	}
	if loadedRegistry.CredentialRef != "cred-registry" || loadedRegistry.Labels["scope"] != "test" || len(loadedRegistry.Capabilities) != 2 {
		t.Fatalf("loaded registry = %#v", loadedRegistry)
	}
	loadedPolicies, err := policies.List(ctx, "project-a", "prod")
	if err != nil || len(loadedPolicies) != 1 || loadedPolicies[0].ID != policy.ID || !loadedPolicies[0].RequireDigest {
		t.Fatalf("loaded policies = %#v err=%v", loadedPolicies, err)
	}
	loadedAttachments, err := policies.ListAttachments(ctx, policyusecase.AttachmentListInput{PolicyID: policy.ID, ScopeType: "environment"})
	if err != nil || len(loadedAttachments) != 1 || loadedAttachments[0].ID != attachment.ID || loadedAttachments[0].Metadata["source"] != "test" {
		t.Fatalf("loaded attachments = %#v err=%v", loadedAttachments, err)
	}
}
