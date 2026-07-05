package postgres

import (
	"context"
	"strings"
	"testing"

	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func TestCatalogStoresImplementInterfaces(t *testing.T) {
	var _ catalogusecase.Store = (*CatalogStore)(nil)
	var _ pipelineusecase.DefinitionCatalogStore = (*PipelineDefinitionStore)(nil)
}

func TestCatalogPersistenceMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000010_catalog_persistence.up.sql")
	down := readMigration(t, "000010_catalog_persistence.down.sql")
	versionUp := readMigration(t, "000012_pipeline_definition_versions.up.sql")
	versionDown := readMigration(t, "000012_pipeline_definition_versions.down.sql")

	for _, table := range []string{
		"catalog_orgs",
		"catalog_projects",
		"catalog_applications",
		"catalog_environments",
		"catalog_repositories",
		"catalog_release_targets",
		"pipeline_definitions",
	} {
		if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Fatalf("up migration missing table %s", table)
		}
		if !strings.Contains(down, "DROP TABLE IF EXISTS "+table) {
			t.Fatalf("down migration missing table %s", table)
		}
	}

	for _, index := range []string{
		"idx_catalog_projects_org",
		"idx_catalog_applications_project",
		"idx_catalog_environments_project",
		"idx_catalog_release_targets_environment",
		"idx_pipeline_definitions_project",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
	}

	if !strings.Contains(versionUp, "CREATE TABLE IF NOT EXISTS pipeline_definition_versions") {
		t.Fatalf("version migration missing pipeline_definition_versions table")
	}
	for _, index := range []string{
		"idx_pipeline_definition_versions_pipeline",
		"idx_pipeline_definition_versions_unique",
	} {
		if !strings.Contains(versionUp, index) {
			t.Fatalf("version migration missing index %s", index)
		}
	}
	if !strings.Contains(versionUp, "INSERT INTO pipeline_definition_versions") {
		t.Fatalf("version migration missing backfill from pipeline_definitions")
	}
	if !strings.Contains(versionDown, "DROP TABLE IF EXISTS pipeline_definition_versions") {
		t.Fatalf("version down migration missing table drop")
	}
}

func TestPostgresIntegrationCatalogAndPipelineDefinitionRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()

	catalog := catalogusecase.NewService(NewCatalogStore(db.pool))
	org, err := catalog.CreateOrg(ctx, catalogusecase.CreateOrgInput{Name: "Platform", Labels: map[string]string{"tier": "root"}})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	project, err := catalog.CreateProject(ctx, catalogusecase.CreateProjectInput{OrgID: org.ID, Name: "Delivery"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	app, err := catalog.CreateApplication(ctx, catalogusecase.CreateApplicationInput{ProjectID: project.ID, Name: "Control Plane"})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	environment, err := catalog.CreateEnvironment(ctx, catalogusecase.CreateEnvironmentInput{ProjectID: project.ID, Name: "Production"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	repository, err := catalog.CreateRepository(ctx, catalogusecase.CreateRepositoryInput{ProjectID: project.ID, Name: "Service Repo", URL: "https://example.com/team/service.git", CredentialRef: "cred-ref"})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	validation, err := catalog.ValidateRepository(ctx, repository.ID)
	if err != nil {
		t.Fatalf("validate repository: %v", err)
	}
	if !validation.Valid {
		t.Fatalf("repository should validate: %#v", validation)
	}
	target, err := catalog.CreateReleaseTarget(ctx, catalogusecase.CreateReleaseTargetInput{EnvironmentID: environment.ID, Name: "prod-noop", TargetType: "noop", CredentialRef: "target-cred-ref"})
	if err != nil {
		t.Fatalf("create release target: %v", err)
	}

	pipelineCatalog := pipelineusecase.NewDefinitionCatalog(NewPipelineDefinitionStore(db.pool))
	definition, err := pipelineCatalog.Create(ctx, pipelineusecase.DefinitionCreateInput{ProjectID: project.ID, Definition: testPipelineDefinition("build")})
	if err != nil {
		t.Fatalf("create pipeline definition: %v", err)
	}
	updatedDefinition, err := pipelineCatalog.Update(ctx, definition.Pipeline.ID, pipelineusecase.DefinitionUpdateInput{Definition: ptrPipelineDefinition(testPipelineDefinition("build-v2"))})
	if err != nil {
		t.Fatalf("update pipeline definition: %v", err)
	}
	if updatedDefinition.Version.Version != 2 {
		t.Fatalf("pipeline definition version = %d, want 2", updatedDefinition.Version.Version)
	}

	restartedPool := db.restart(t)
	catalog = catalogusecase.NewService(NewCatalogStore(restartedPool))
	pipelineCatalog = pipelineusecase.NewDefinitionCatalog(NewPipelineDefinitionStore(restartedPool))

	loadedOrg, err := catalog.GetOrg(ctx, org.ID)
	if err != nil || loadedOrg.Labels["tier"] != "root" {
		t.Fatalf("reload org = %#v err=%v", loadedOrg, err)
	}
	if loadedApp, err := catalog.GetApplication(ctx, app.ID); err != nil || loadedApp.ProjectID != project.ID {
		t.Fatalf("reload application = %#v err=%v", loadedApp, err)
	}
	if repos, err := catalog.ListRepositories(ctx, project.ID); err != nil || len(repos) != 1 || repos[0].ID != repository.ID || repos[0].CredentialRef != "cred-ref" {
		t.Fatalf("reload repositories = %#v err=%v", repos, err)
	}
	events, err := catalog.Events(ctx, repository.ID)
	if err != nil {
		t.Fatalf("reload repository validation events: %v", err)
	}
	if len(events) != 1 || events[0].Type != catalogusecase.EventRepositoryValidated {
		t.Fatalf("repository validation events = %#v", events)
	}
	audits, err := catalog.Audits(ctx, repository.ID)
	if err != nil {
		t.Fatalf("reload repository validation audits: %v", err)
	}
	if len(audits) != 1 || audits[0].Action != "repository validated" || audits[0].RecordHash == "" {
		t.Fatalf("repository validation audits = %#v", audits)
	}
	if targets, err := catalog.ListReleaseTargets(ctx, project.ID, environment.ID); err != nil || len(targets) != 1 || targets[0].ID != target.ID || targets[0].CredentialRef != "target-cred-ref" {
		t.Fatalf("reload release targets = %#v err=%v", targets, err)
	}
	loadedDefinition, err := pipelineCatalog.Get(ctx, definition.Pipeline.ID)
	if err != nil {
		t.Fatalf("reload pipeline definition: %v", err)
	}
	if loadedDefinition.Version.Version != 2 || loadedDefinition.Definition.Metadata.Name != "build-v2" {
		t.Fatalf("loaded definition = %#v", loadedDefinition)
	}
	versions, err := pipelineCatalog.Versions(ctx, definition.Pipeline.ID)
	if err != nil {
		t.Fatalf("reload pipeline definition versions: %v", err)
	}
	if len(versions) != 2 || versions[0].Version != 1 || versions[1].Version != 2 {
		t.Fatalf("loaded definition versions = %#v", versions)
	}
	firstVersion, err := pipelineCatalog.Version(ctx, definition.Pipeline.ID, 1)
	if err != nil {
		t.Fatalf("reload first pipeline definition version: %v", err)
	}
	if firstVersion.Version.ID != definition.Version.ID || firstVersion.Definition.Metadata.Name != "build" {
		t.Fatalf("loaded first definition version = %#v", firstVersion)
	}
	secondVersion, err := pipelineCatalog.Version(ctx, definition.Pipeline.ID, 2)
	if err != nil {
		t.Fatalf("reload second pipeline definition version: %v", err)
	}
	if secondVersion.Version.ID != updatedDefinition.Version.ID || secondVersion.Definition.Metadata.Name != "build-v2" {
		t.Fatalf("loaded second definition version = %#v", secondVersion)
	}
}

func ptrPipelineDefinition(def pipelineusecase.Definition) *pipelineusecase.Definition {
	return &def
}

func testPipelineDefinition(name string) pipelineusecase.Definition {
	return pipelineusecase.Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Pipeline",
		Metadata:   pipelineusecase.Metadata{Name: name},
		Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
			Name: "build",
			Jobs: []pipelineusecase.Job{{
				Name:     "test",
				Executor: "shell",
				Steps:    []pipelineusecase.Step{{Name: "echo", Run: "printf ok"}},
			}},
		}}},
	}
}
