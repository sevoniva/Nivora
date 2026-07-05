package catalog

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCatalogCreatesHierarchyAndDisablesResources(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore())

	org, err := service.CreateOrg(ctx, CreateOrgInput{Name: "Platform"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if !org.Enabled || org.Slug != "platform" {
		t.Fatalf("unexpected org: %+v", org)
	}

	project, err := service.CreateProject(ctx, CreateProjectInput{OrgID: org.ID, Name: "Delivery"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if project.OrgID != org.ID || !project.Enabled {
		t.Fatalf("unexpected project: %+v", project)
	}

	app, err := service.CreateApplication(ctx, CreateApplicationInput{ProjectID: project.ID, Name: "Control Plane"})
	if err != nil {
		t.Fatalf("create application: %v", err)
	}
	if app.ProjectID != project.ID || app.Slug != "control-plane" {
		t.Fatalf("unexpected application: %+v", app)
	}

	environment, err := service.CreateEnvironment(ctx, CreateEnvironmentInput{ProjectID: project.ID, Name: "Production"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	if environment.ProjectID != project.ID || environment.Slug != "production" {
		t.Fatalf("unexpected environment: %+v", environment)
	}

	disabled, err := service.DisableEnvironment(ctx, environment.ID)
	if err != nil {
		t.Fatalf("disable environment: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("delete should disable environment, got %+v", disabled)
	}

	repository, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Service Repo", URL: "https://example.com/team/service.git"})
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if repository.ProjectID != project.ID || repository.Provider != "generic" || repository.DefaultBranch != "main" {
		t.Fatalf("unexpected repository: %+v", repository)
	}
	repositoryCheck, err := service.ValidateRepository(ctx, repository.ID)
	if err != nil {
		t.Fatalf("validate repository: %v", err)
	}
	if !repositoryCheck.Valid {
		t.Fatalf("repository should validate: %+v", repositoryCheck)
	}
	disabledRepo, err := service.DisableRepository(ctx, repository.ID)
	if err != nil {
		t.Fatalf("disable repository: %v", err)
	}
	if disabledRepo.Enabled {
		t.Fatalf("delete should disable repository, got %+v", disabledRepo)
	}
	repositoryCheck, err = service.ValidateRepository(ctx, repository.ID)
	if err != nil {
		t.Fatalf("validate disabled repository: %v", err)
	}
	if repositoryCheck.Valid || len(repositoryCheck.Errors) == 0 {
		t.Fatalf("disabled repository should not validate: %+v", repositoryCheck)
	}
}

func TestCatalogRejectsMissingParentsAndDuplicates(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore())

	if _, err := service.CreateProject(ctx, CreateProjectInput{OrgID: "missing", Name: "Delivery"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing org error = %v", err)
	}

	org, err := service.CreateOrg(ctx, CreateOrgInput{Name: "Platform"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	if _, err := service.CreateOrg(ctx, CreateOrgInput{Name: "platform"}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate org error = %v", err)
	}
	project, err := service.CreateProject(ctx, CreateProjectInput{OrgID: org.ID, Name: "Delivery"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if _, err := service.CreateApplication(ctx, CreateApplicationInput{ProjectID: "missing", Name: "App"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing project application error = %v", err)
	}
	if _, err := service.CreateApplication(ctx, CreateApplicationInput{ProjectID: project.ID, Name: ""}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid application error = %v", err)
	}
	if _, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Repo", URL: "example.com/no-scheme"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid repository url error = %v", err)
	}
	if _, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Inline Repo", URL: "https://user:pass@example.com/team/service.git"}); !errors.Is(err, ErrInvalid) || strings.Contains(err.Error(), "user") || strings.Contains(err.Error(), "pass") {
		t.Fatalf("inline repository credential error should reject without leaking values, got %v", err)
	}
	if _, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Repo", URL: "https://example.com/team/service.git", Provider: "unknown"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid repository provider error = %v", err)
	}
	genericGitRepository, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Generic Git Repo", URL: "https://example.com/team/generic.git", Provider: "generic_git"})
	if err != nil {
		t.Fatalf("create generic_git repository alias: %v", err)
	}
	if genericGitRepository.Provider != "generic" {
		t.Fatalf("generic_git provider should normalize to generic, got %q", genericGitRepository.Provider)
	}
	localRepository, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Local Repo", URL: "file:///tmp/nivora-local-repo", Provider: "local", DefaultBranch: "HEAD"})
	if err != nil {
		t.Fatalf("create local file repository metadata: %v", err)
	}
	if localRepository.Provider != "local" || localRepository.URL != "file:///tmp/nivora-local-repo" {
		t.Fatalf("unexpected local repository: %+v", localRepository)
	}
	archiveRepository, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Archive Repo", URL: "file:///tmp/nivora-source.tar.gz", Provider: "archive", DefaultBranch: "snapshot"})
	if err != nil {
		t.Fatalf("create archive file repository metadata: %v", err)
	}
	if archiveRepository.Provider != "archive" {
		t.Fatalf("unexpected archive provider: %+v", archiveRepository)
	}
	if _, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Generic File Repo", URL: "file:///tmp/nivora-generic", Provider: "generic"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("generic file repository should require local/archive provider, got %v", err)
	}
	repository, err := service.CreateRepository(ctx, CreateRepositoryInput{ProjectID: project.ID, Name: "Repo", URL: "https://example.com/team/service.git", Provider: "github", CredentialRef: "cred-ref"})
	if err != nil {
		t.Fatalf("create repository with credential ref: %v", err)
	}
	if _, err := service.UpdateRepository(ctx, repository.ID, UpdateRepositoryInput{URL: stringPtr("https://user:pass@example.com/team/other.git")}); !errors.Is(err, ErrInvalid) || strings.Contains(err.Error(), "user") || strings.Contains(err.Error(), "pass") {
		t.Fatalf("inline repository credential update should reject without leaking values, got %v", err)
	}
	result, err := service.ValidateRepository(ctx, repository.ID)
	if err != nil {
		t.Fatalf("validate repository with credential ref: %v", err)
	}
	if !result.Valid || len(result.Warnings) < 2 {
		t.Fatalf("expected metadata-only warnings for provider and credential ref: %+v", result)
	}
}

func TestReleaseTargetCatalog(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore())
	org, err := service.CreateOrg(ctx, CreateOrgInput{Name: "Platform"})
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	project, err := service.CreateProject(ctx, CreateProjectInput{OrgID: org.ID, Name: "Delivery"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	environment, err := service.CreateEnvironment(ctx, CreateEnvironmentInput{ProjectID: project.ID, Name: "Dev"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}

	target, err := service.CreateReleaseTarget(ctx, CreateReleaseTargetInput{
		EnvironmentID: environment.ID,
		Name:          "local-noop",
		TargetType:    "noop",
		CredentialRef: "cred-placeholder",
		Labels:        map[string]string{"tier": "dev"},
	})
	if err != nil {
		t.Fatalf("create release target: %v", err)
	}
	if target.ProjectID != project.ID || !target.Enabled {
		t.Fatalf("unexpected target: %+v", target)
	}
	if target.AllowApply || target.AllowSync || target.AllowRemoteHostDeploy {
		t.Fatalf("unsafe flags should default false: %+v", target)
	}
	if target.CredentialRef != "cred-placeholder" {
		t.Fatalf("credential ref = %q", target.CredentialRef)
	}

	listed, err := service.ListReleaseTargets(ctx, project.ID, environment.ID)
	if err != nil {
		t.Fatalf("list targets: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != target.ID {
		t.Fatalf("listed targets = %+v", listed)
	}

	result, err := service.ValidateReleaseTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("validate target: %v", err)
	}
	if !result.Valid {
		t.Fatalf("target should validate: %+v", result)
	}

	disabled, err := service.DisableReleaseTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("disable target: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("target should be disabled: %+v", disabled)
	}
	result, err = service.ValidateReleaseTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("validate disabled target: %v", err)
	}
	if result.Valid {
		t.Fatalf("disabled target should not validate: %+v", result)
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestReleaseTargetCatalogRejectsInvalidInputs(t *testing.T) {
	ctx := context.Background()
	service := NewService(NewMemoryStore())
	org, _ := service.CreateOrg(ctx, CreateOrgInput{Name: "Platform"})
	project, _ := service.CreateProject(ctx, CreateProjectInput{OrgID: org.ID, Name: "Delivery"})
	environment, _ := service.CreateEnvironment(ctx, CreateEnvironmentInput{ProjectID: project.ID, Name: "Dev"})

	if _, err := service.CreateReleaseTarget(ctx, CreateReleaseTargetInput{EnvironmentID: "missing", Name: "missing", TargetType: "noop"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing environment error = %v", err)
	}
	if _, err := service.CreateReleaseTarget(ctx, CreateReleaseTargetInput{EnvironmentID: environment.ID, ProjectID: "other", Name: "wrong-project", TargetType: "noop"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("wrong project error = %v", err)
	}
	if _, err := service.CreateReleaseTarget(ctx, CreateReleaseTargetInput{EnvironmentID: environment.ID, Name: "bad", TargetType: "cloud"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid target type error = %v", err)
	}
	if _, err := service.CreateReleaseTarget(ctx, CreateReleaseTargetInput{EnvironmentID: environment.ID, Name: "local-noop", TargetType: "noop"}); err != nil {
		t.Fatalf("create first target: %v", err)
	}
	if _, err := service.CreateReleaseTarget(ctx, CreateReleaseTargetInput{EnvironmentID: environment.ID, Name: "LOCAL-NOOP", TargetType: "noop"}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate target error = %v", err)
	}
	second, err := service.CreateReleaseTarget(ctx, CreateReleaseTargetInput{EnvironmentID: environment.ID, Name: "dev-yaml", TargetType: "kubernetes-yaml"})
	if err != nil {
		t.Fatalf("create second target: %v", err)
	}
	duplicateName := "local-noop"
	if _, err := service.UpdateReleaseTarget(ctx, second.ID, UpdateReleaseTargetInput{Name: &duplicateName}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate target update error = %v", err)
	}
}
