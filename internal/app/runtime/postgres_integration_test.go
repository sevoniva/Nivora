package runtime

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/infra/config"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
)

func TestPostgresIntegrationRuntimeBootstrapUsesPostgresStores(t *testing.T) {
	db := newRuntimePostgresIntegration(t)
	defer db.cleanup()

	ctx := context.Background()
	cfg := config.Default()
	cfg.Database.RuntimeStore = "postgres"
	cfg.Database.URL = db.runtimeURL
	cfg.Runner.Name = "bootstrap-runner"

	service, closeFn, err := NewPipelineServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("bootstrap pipeline service with postgres config: %v", err)
	}
	result, err := service.CreateQueued(ctx, pipelineusecase.CreateRunInput{
		Definition: pipelineusecase.Definition{APIVersion: "nivora.io/v1alpha1", Kind: "Pipeline", Metadata: pipelineusecase.Metadata{Name: "bootstrap-pipeline"}, Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
			Name: "build",
			Jobs: []pipelineusecase.Job{{Name: "echo", Executor: "shell", Steps: []pipelineusecase.Step{{Name: "say", Run: "printf durable"}}}},
		}}}},
		ActorID:       "integration-test",
		CorrelationID: "corr-runtime-bootstrap",
	})
	if err != nil {
		closeFn()
		t.Fatalf("create queued pipeline in postgres runtime: %v", err)
	}
	closeFn()

	service, closeFn, err = NewPipelineServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("restart pipeline service with postgres config: %v", err)
	}
	defer closeFn()
	loaded, err := service.Get(ctx, result.Record.Run.ID)
	if err != nil {
		t.Fatalf("reload queued pipeline from restarted postgres runtime: %v", err)
	}
	if loaded.Run.ID != result.Record.Run.ID || loaded.Run.CorrelationID != "corr-runtime-bootstrap" {
		t.Fatalf("runtime bootstrap did not persist pipeline run: %#v", loaded.Run)
	}

	catalog, closeCatalog, err := NewCatalogServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("bootstrap catalog service with postgres config: %v", err)
	}
	org, err := catalog.CreateOrg(ctx, catalogusecase.CreateOrgInput{Name: "Runtime Bootstrap"})
	if err != nil {
		closeCatalog()
		t.Fatalf("create catalog org in postgres runtime: %v", err)
	}
	project, err := catalog.CreateProject(ctx, catalogusecase.CreateProjectInput{OrgID: org.ID, Name: "Runtime Project"})
	if err != nil {
		closeCatalog()
		t.Fatalf("create catalog project in postgres runtime: %v", err)
	}
	closeCatalog()

	catalog, closeCatalog, err = NewCatalogServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("restart catalog service with postgres config: %v", err)
	}
	reloadedProject, err := catalog.GetProject(ctx, project.ID)
	closeCatalog()
	if err != nil {
		t.Fatalf("reload catalog project from restarted postgres runtime: %v", err)
	}
	if reloadedProject.OrgID != org.ID {
		t.Fatalf("runtime bootstrap did not persist catalog project: %#v", reloadedProject)
	}

	pipelineCatalog, closePipelineCatalog, err := NewPipelineDefinitionCatalogWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("bootstrap pipeline definition catalog with postgres config: %v", err)
	}
	definition, err := pipelineCatalog.Create(ctx, pipelineusecase.DefinitionCreateInput{
		ProjectID: project.ID,
		Definition: pipelineusecase.Definition{APIVersion: "nivora.io/v1alpha1", Kind: "Pipeline", Metadata: pipelineusecase.Metadata{Name: "catalog-bootstrap"}, Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
			Name: "build",
			Jobs: []pipelineusecase.Job{{Name: "echo", Executor: "shell", Steps: []pipelineusecase.Step{{Name: "say", Run: "printf durable"}}}},
		}}}},
	})
	if err != nil {
		closePipelineCatalog()
		t.Fatalf("create pipeline definition in postgres runtime: %v", err)
	}
	closePipelineCatalog()

	pipelineCatalog, closePipelineCatalog, err = NewPipelineDefinitionCatalogWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("restart pipeline definition catalog with postgres config: %v", err)
	}
	reloadedDefinition, err := pipelineCatalog.Get(ctx, definition.Pipeline.ID)
	closePipelineCatalog()
	if err != nil {
		t.Fatalf("reload pipeline definition from restarted postgres runtime: %v", err)
	}
	if reloadedDefinition.Pipeline.ProjectID != project.ID || reloadedDefinition.Definition.Metadata.Name != "catalog-bootstrap" {
		t.Fatalf("runtime bootstrap did not persist pipeline definition: %#v", reloadedDefinition)
	}

	registryService, closeRegistry, err := NewArtifactRegistryServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("bootstrap artifact registry service with postgres config: %v", err)
	}
	registry, err := registryService.Create(ctx, artifactusecase.RegistryCreateInput{ProjectID: project.ID, Name: "runtime-registry", Endpoint: "registry.example.invalid/team", CredentialRef: "cred-ref"})
	if err != nil {
		closeRegistry()
		t.Fatalf("create artifact registry in postgres runtime: %v", err)
	}
	closeRegistry()

	registryService, closeRegistry, err = NewArtifactRegistryServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("restart artifact registry service with postgres config: %v", err)
	}
	reloadedRegistry, err := registryService.Get(ctx, registry.ID)
	closeRegistry()
	if err != nil {
		t.Fatalf("reload artifact registry from restarted postgres runtime: %v", err)
	}
	if reloadedRegistry.CredentialRef != "cred-ref" {
		t.Fatalf("runtime bootstrap did not persist artifact registry: %#v", reloadedRegistry)
	}

	policyCatalog, closePolicyCatalog, err := NewPolicyCatalogServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("bootstrap policy catalog with postgres config: %v", err)
	}
	policy, err := policyCatalog.Create(ctx, policyusecase.CreateInput{ProjectID: project.ID, EnvironmentID: "prod", Name: "runtime-policy", RequireDigest: true})
	if err != nil {
		closePolicyCatalog()
		t.Fatalf("create policy in postgres runtime: %v", err)
	}
	closePolicyCatalog()

	policyCatalog, closePolicyCatalog, err = NewPolicyCatalogServiceWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("restart policy catalog with postgres config: %v", err)
	}
	reloadedPolicy, err := policyCatalog.Get(ctx, policy.ID)
	closePolicyCatalog()
	if err != nil {
		t.Fatalf("reload policy from restarted postgres runtime: %v", err)
	}
	if !reloadedPolicy.RequireDigest || reloadedPolicy.ProjectID != project.ID {
		t.Fatalf("runtime bootstrap did not persist policy: %#v", reloadedPolicy)
	}

	prod := config.Default()
	prod.Env = "production"
	prod.Auth.Enabled = true
	prod.Runtime.AllowLocalShellExecutor = false
	prod.Database.RuntimeStore = "memory"
	if err := prod.Validate(); err == nil {
		t.Fatal("production config accepted memory runtime store")
	}
}

type runtimePostgresIntegration struct {
	admin      *pgxpool.Pool
	runtimeURL string
	schema     string
}

func newRuntimePostgresIntegration(t *testing.T) *runtimePostgresIntegration {
	t.Helper()
	if os.Getenv("NIVORA_RUN_POSTGRES_INTEGRATION") != "true" {
		t.Skip("set NIVORA_RUN_POSTGRES_INTEGRATION=true and DATABASE_URL to run PostgreSQL integration tests")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required when NIVORA_RUN_POSTGRES_INTEGRATION=true")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin postgres: %v", err)
	}
	schema := fmt.Sprintf("nivora_runtime_it_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatalf("create schema: %v", err)
	}
	runtimeURL := runtimePostgresURLWithSearchPath(t, databaseURL, schema)
	pool, err := pgxpool.New(ctx, runtimeURL)
	if err != nil {
		_, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close()
		t.Fatalf("connect schema postgres: %v", err)
	}
	runtimeApplyUpMigrations(t, pool)
	pool.Close()
	return &runtimePostgresIntegration{admin: admin, runtimeURL: runtimeURL, schema: schema}
}

func (db *runtimePostgresIntegration) cleanup() {
	if db.admin != nil {
		_, _ = db.admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+db.schema+" CASCADE")
		db.admin.Close()
	}
}

func runtimePostgresURLWithSearchPath(t *testing.T, raw string, schema string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("options", "-c search_path="+schema)
	u.RawQuery = q.Encode()
	return u.String()
}

func runtimeApplyUpMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "infra", "migration", "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(body)); err != nil {
			t.Fatalf("execute migration %s: %v", filepath.Base(path), err)
		}
	}
}
