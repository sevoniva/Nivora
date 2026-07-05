package mcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	repositoryusecase "github.com/sevoniva/nivora/internal/usecase/repository"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
)

func TestMCPRepositoryInspectToolIsPlanOnlyAndRedacted(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/mcp\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("PASSWORD=should-not-leak\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	recorder := &MemoryAuditRecorder{}
	server := newTestMCPServerWithRecorder(t, domainauth.Subject{
		ID:       "repository-viewer",
		Username: "repository-viewer",
		Roles:    []string{domainauth.RoleViewer},
		AuthMode: "mcp-local",
	}, recorder)
	result, err := server.CallTool(context.Background(), "nivora_repository_inspect", map[string]any{
		"path": root,
		"name": "mcp-local-repo",
	})
	if err != nil {
		t.Fatalf("repository inspect transport error: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("repository inspect result = %#v", result)
	}
	body := result.Content[0].Text
	for _, want := range []string{`"mutated": false`, "go test ./...", "mcp-local-repo"} {
		if !strings.Contains(body, want) {
			t.Fatalf("repository inspect body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("repository inspect leaked .env content: %s", body)
	}
	assertMCPAuditAction(t, recorder, EventDevOpsMCPRepositoryInspected)
	assertMCPAuditDoesNotContain(t, recorder, root, "should-not-leak")
}

func TestMCPRepositoryDevOpsPlanResourceAndToolArePlanOnly(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/devops\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatalf("write Dockerfile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=should-not-leak\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	projectID, _ := createMCPCatalogAndPipelineFixture(t, server)
	repositories, err := server.services.Catalog.ListRepositories(context.Background(), projectID)
	if err != nil {
		t.Fatalf("list catalog repositories: %v", err)
	}
	if len(repositories) == 0 {
		t.Fatal("expected catalog repository fixture")
	}
	repositoryID := repositories[0].ID

	result, err := server.CallTool(context.Background(), "nivora_repository_snapshot_create", map[string]any{"repositoryId": repositoryID, "ref": "HEAD", "localPath": root})
	if err != nil {
		t.Fatalf("repository snapshot create transport error: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("repository snapshot create result = %#v", result)
	}
	body := result.Content[0].Text
	for _, want := range []string{`"mutated": false`, `"snapshot"`, `"intelligence"`, "go test ./...", "preview"} {
		if !strings.Contains(body, want) {
			t.Fatalf("repository snapshot create body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("repository snapshot create leaked .env content: %s", body)
	}

	catalogRepository, err := server.services.Catalog.GetRepository(context.Background(), repositoryID)
	if err != nil {
		t.Fatalf("get catalog repository: %v", err)
	}
	if _, err := server.services.Repositories.CreateSnapshot(context.Background(), repositoryusecase.SnapshotInput{Repository: repositoryUsecaseFromCatalog(catalogRepository), Ref: "HEAD", LocalPath: root}); err != nil {
		t.Fatalf("create repository snapshot fixture: %v", err)
	}

	result, err = server.CallTool(context.Background(), "nivora_repository_intelligence_analyze", map[string]any{"repositoryId": repositoryID})
	if err != nil {
		t.Fatalf("repository intelligence analyze transport error: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("repository intelligence analyze result = %#v", result)
	}
	body = result.Content[0].Text
	for _, want := range []string{`"mutated": false`, `"intelligence"`, "kind: Workflow", "preview"} {
		if !strings.Contains(body, want) {
			t.Fatalf("repository intelligence analyze body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("repository intelligence analyze leaked .env content: %s", body)
	}

	resource, err := server.ReadResource(context.Background(), "nivora://repositories/"+repositoryID+"/devops-plan")
	if err != nil {
		t.Fatalf("ReadResource devops-plan: %v", err)
	}
	for _, want := range []string{`"mutated": false`, `"releaseCandidate"`, "go test ./...", "plan-only"} {
		if !strings.Contains(resource.Text, want) {
			t.Fatalf("devops plan resource missing %q: %s", want, resource.Text)
		}
	}
	if strings.Contains(resource.Text, "should-not-leak") {
		t.Fatalf("devops plan resource leaked .env content: %s", resource.Text)
	}

	result, err = server.CallTool(context.Background(), "nivora_repository_devops_plan", map[string]any{"repositoryId": repositoryID})
	if err != nil {
		t.Fatalf("repository devops plan transport error: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("repository devops plan result = %#v", result)
	}
	body = result.Content[0].Text
	for _, want := range []string{`"mutated": false`, `"releaseCandidate"`, "go build ./...", "metadata-only"} {
		if !strings.Contains(body, want) {
			t.Fatalf("devops plan tool body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("devops plan tool leaked .env content: %s", body)
	}

	result, err = server.CallTool(context.Background(), "nivora_devops_readiness_review", map[string]any{"repositoryId": repositoryID})
	if err != nil {
		t.Fatalf("devops readiness review transport error: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("devops readiness review result = %#v", result)
	}
	body = result.Content[0].Text
	for _, want := range []string{`"mutated": false`, `"planOnly": true`, `"recommendedNextActions"`, "does not execute"} {
		if !strings.Contains(body, want) {
			t.Fatalf("devops readiness review body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("devops readiness review leaked .env content: %s", body)
	}

	result, err = server.CallTool(context.Background(), "nivora_workflow_draft_generate", map[string]any{"repositoryId": repositoryID})
	if err != nil {
		t.Fatalf("workflow draft generate transport error: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("workflow draft generate result = %#v", result)
	}
	body = result.Content[0].Text
	for _, want := range []string{`"mutated": false`, `"workflowDraft"`, "kind: Workflow"} {
		if !strings.Contains(body, want) {
			t.Fatalf("workflow draft generate body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("workflow draft generate leaked .env content: %s", body)
	}
}

func TestMCPTenantScopeFiltersRepositoryAndWorkflowResources(t *testing.T) {
	ctx := context.Background()
	base := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	projectA := base.WithSubject(domainauth.Subject{
		ID:        "sa-repository-project-a",
		Username:  "sa-repository-project-a",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-a",
	})
	projectB := base.WithSubject(domainauth.Subject{
		ID:        "sa-repository-project-b",
		Username:  "sa-repository-project-b",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-b",
	})

	repoA := createScopedMCPRepositorySnapshot(t, base, "project-a", "repo-a", "mcp-repo-a")
	repoB := createScopedMCPRepositorySnapshot(t, base, "project-b", "repo-b", "mcp-repo-b")
	workflowA := createScopedMCPWorkflowPlan(t, base, repoA.ID, "workflow-a")
	workflowB := createScopedMCPWorkflowPlan(t, base, repoB.ID, "workflow-b")

	repositoryList, err := projectA.ReadResource(ctx, "nivora://repositories")
	if err != nil {
		t.Fatalf("project A repository list: %v", err)
	}
	if !strings.Contains(repositoryList.Text, repoA.ID) || strings.Contains(repositoryList.Text, repoB.ID) || strings.Contains(repositoryList.Text, "project-b") {
		t.Fatalf("project A repository list leaked or missed scoped data: %s", repositoryList.Text)
	}

	for _, uri := range []string{
		"nivora://repositories/" + repoB.ID,
		"nivora://repositories/" + repoB.ID + "/snapshot/latest",
		"nivora://repositories/" + repoB.ID + "/intelligence",
		"nivora://repositories/" + repoB.ID + "/devops-plan",
		"nivora://workflows/" + workflowB.ID + "/plan",
		"nivora://workflows/" + workflowB.WorkflowID,
	} {
		if _, err := projectA.ReadResource(ctx, uri); err == nil {
			t.Fatalf("project A should not read cross-project resource %s", uri)
		} else {
			assertMCPScopeDenied(t, err)
		}
	}

	for name, args := range map[string]map[string]any{
		"nivora_repository_snapshot_create":      {"repositoryId": repoB.ID, "localPath": repoB.LocalPath, "ref": "HEAD"},
		"nivora_repository_intelligence_analyze": {"repositoryId": repoB.ID},
		"nivora_repository_devops_plan":          {"repositoryId": repoB.ID},
		"nivora_devops_readiness_review":         {"repositoryId": repoB.ID},
		"nivora_workflow_draft_generate":         {"repositoryId": repoB.ID},
	} {
		result, err := projectA.CallTool(ctx, name, args)
		if err != nil {
			t.Fatalf("%s transport error: %v", name, err)
		}
		if !result.IsError || !strings.Contains(result.Content[0].Text, "mcp_scope_denied") {
			t.Fatalf("%s should be scope denied for project A, got %#v", name, result)
		}
	}

	snapshot, err := projectB.ReadResource(ctx, "nivora://repositories/"+repoB.ID+"/snapshot/latest")
	if err != nil {
		t.Fatalf("project B read own snapshot: %v", err)
	}
	if !strings.Contains(snapshot.Text, repoB.ID) || !strings.Contains(snapshot.Text, `"mutated": false`) {
		t.Fatalf("project B snapshot missing scoped data or read-only marker: %s", snapshot.Text)
	}

	workflows, err := projectA.ReadResource(ctx, "nivora://workflows")
	if err != nil {
		t.Fatalf("project A workflow list: %v", err)
	}
	if !strings.Contains(workflows.Text, workflowA.WorkflowID) || strings.Contains(workflows.Text, workflowB.WorkflowID) || strings.Contains(workflows.Text, "project-b") {
		t.Fatalf("project A workflow list leaked or missed scoped data: %s", workflows.Text)
	}
}

func TestMCPWorkflowToolsPlanOnly(t *testing.T) {
	workflow := `
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: mcp-ci
on: [manual]
jobs:
  test:
    steps:
      - name: test
        run: go test ./...
`
	recorder := &MemoryAuditRecorder{}
	server := newTestMCPServerWithRecorder(t, domainauth.Subject{
		ID:       "workflow-developer",
		Username: "workflow-developer",
		Roles:    []string{domainauth.RoleDeveloper},
		AuthMode: "mcp-local",
	}, recorder)
	validate, err := server.CallTool(context.Background(), "nivora_workflow_validate", map[string]any{"content": workflow})
	if err != nil {
		t.Fatalf("workflow validate transport error: %v", err)
	}
	if validate.IsError || !strings.Contains(validate.Content[0].Text, `"valid": true`) || !strings.Contains(validate.Content[0].Text, `"mutated": false`) {
		t.Fatalf("workflow validate result = %#v", validate)
	}

	plan, err := server.CallTool(context.Background(), "nivora_workflow_plan", map[string]any{"content": workflow})
	if err != nil {
		t.Fatalf("workflow plan transport error: %v", err)
	}
	if plan.IsError || !strings.Contains(plan.Content[0].Text, `"workflowId"`) || !strings.Contains(plan.Content[0].Text, `"mutated": false`) {
		t.Fatalf("workflow plan result = %#v", plan)
	}
	assertMCPAuditAction(t, recorder, EventDevOpsMCPWorkflowPlanned)
	assertMCPAuditDoesNotContain(t, recorder, "go test ./...")
}

func assertMCPScopeDenied(t *testing.T, err error) {
	t.Helper()
	var op OperationError
	if !errors.As(err, &op) || op.Code != "mcp_scope_denied" {
		t.Fatalf("expected mcp_scope_denied, got %T %v", err, err)
	}
}

type scopedMCPRepositorySnapshot struct {
	ID        string
	LocalPath string
}

func createScopedMCPRepositorySnapshot(t *testing.T, server *Server, projectID string, repositoryID string, name string) scopedMCPRepositorySnapshot {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/"+repositoryID+"\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatalf("write go.mod fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatalf("write Dockerfile fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=should-not-leak\n"), 0o600); err != nil {
		t.Fatalf("write .env fixture: %v", err)
	}
	org, err := server.services.Catalog.CreateOrg(ctx, catalogusecase.CreateOrgInput{
		Name: "Org " + projectID,
		Slug: "org-" + projectID,
	})
	if err != nil {
		t.Fatalf("create scoped org fixture: %v", err)
	}
	if _, err := server.services.Catalog.CreateProject(ctx, catalogusecase.CreateProjectInput{
		ID:    projectID,
		OrgID: org.ID,
		Name:  "Project " + projectID,
		Slug:  projectID,
	}); err != nil {
		t.Fatalf("create scoped project fixture: %v", err)
	}
	repository, err := server.services.Catalog.CreateRepository(ctx, catalogusecase.CreateRepositoryInput{
		ID:            repositoryID,
		ProjectID:     projectID,
		Name:          name,
		URL:           "file://" + root,
		Provider:      "local",
		DefaultBranch: "HEAD",
		CredentialRef: "credential-ref-placeholder",
	})
	if err != nil {
		t.Fatalf("create scoped repository fixture: %v", err)
	}
	if _, err := server.services.Repositories.CreateSnapshot(ctx, repositoryusecase.SnapshotInput{
		Repository: repositoryUsecaseFromCatalog(repository),
		Ref:        "HEAD",
		LocalPath:  root,
	}); err != nil {
		t.Fatalf("create scoped repository snapshot fixture: %v", err)
	}
	return scopedMCPRepositorySnapshot{ID: repository.ID, LocalPath: root}
}

func createScopedMCPWorkflowPlan(t *testing.T, server *Server, repositoryID string, name string) workflowusecase.PlanRecord {
	t.Helper()
	record, err := server.services.Workflows.Plan(context.Background(), workflowusecase.PlanInput{
		RepositoryID: repositoryID,
		Content: `
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: ` + name + `
on: [manual]
jobs:
  test:
    steps:
      - name: test
        run: go test ./...
`,
	})
	if err != nil {
		t.Fatalf("create scoped workflow plan fixture: %v", err)
	}
	return record
}

func assertMCPAuditAction(t *testing.T, recorder *MemoryAuditRecorder, action string) {
	t.Helper()
	for _, entry := range recorder.Entries() {
		if entry.Action == action {
			return
		}
	}
	t.Fatalf("missing MCP audit action %s in %#v", action, recorder.Entries())
}

func assertMCPAuditDoesNotContain(t *testing.T, recorder *MemoryAuditRecorder, values ...string) {
	t.Helper()
	body := mustMarshal(t, recorder.Entries())
	for _, value := range values {
		if value != "" && strings.Contains(body, value) {
			t.Fatalf("MCP audit leaked %q: %s", value, body)
		}
	}
}

func TestMCPStoredWorkflowPlanResourceReadsSavedPlan(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	record, err := server.services.Workflows.Plan(context.Background(), workflowusecase.PlanInput{Content: `
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: stored-mcp
on: [manual]
jobs:
  test:
    steps:
      - run: go test ./...
`})
	if err != nil {
		t.Fatalf("store workflow plan: %v", err)
	}
	list, err := server.ReadResource(context.Background(), "nivora://workflows")
	if err != nil {
		t.Fatalf("ReadResource workflows: %v", err)
	}
	if !strings.Contains(list.Text, `"workflows"`) || !strings.Contains(list.Text, record.WorkflowID) || !strings.Contains(list.Text, `"mutated": false`) {
		t.Fatalf("workflow list resource = %#v", list)
	}
	resource, err := server.ReadResource(context.Background(), "nivora://workflows/"+record.ID+"/plan")
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if !strings.Contains(resource.Text, `"workflowPlan"`) || !strings.Contains(resource.Text, `"mutated": false`) {
		t.Fatalf("workflow plan resource = %#v", resource)
	}
	detail, err := server.ReadResource(context.Background(), "nivora://workflows/"+record.WorkflowID)
	if err != nil {
		t.Fatalf("ReadResource workflow detail: %v", err)
	}
	for _, want := range []string{`"workflow"`, record.WorkflowID, record.ID, `"mutated": false`} {
		if !strings.Contains(detail.Text, want) {
			t.Fatalf("workflow detail missing %q: %s", want, detail.Text)
		}
	}
}

func TestMCPWorkflowRunResourcesReadGuardedRunMetadata(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	result, err := server.services.Workflows.Run(context.Background(), workflowusecase.RunInput{
		Content: `
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: mcp-run
on: [manual]
jobs:
  test:
    steps:
      - run: echo mcp
`,
		RepositoryID:     "repo-mcp",
		ProjectID:        "project-mcp",
		EnvironmentID:    "env-mcp",
		Confirm:          true,
		AllowPipelineRun: true,
	}, server.services.Pipelines)
	if err != nil {
		t.Fatalf("store workflow run: %v", err)
	}
	if _, err := server.services.Pipelines.ProcessQueued(context.Background(), 1); err != nil {
		t.Fatalf("process linked PipelineRun: %v", err)
	}

	list, err := server.ReadResource(context.Background(), "nivora://workflows/runs")
	if err != nil {
		t.Fatalf("ReadResource workflow runs: %v", err)
	}
	for _, want := range []string{`"workflowRuns"`, result.WorkflowRun.ID, `"Succeeded"`, `"mutated": false`} {
		if !strings.Contains(list.Text, want) {
			t.Fatalf("workflow run list missing %q: %s", want, list.Text)
		}
	}

	resource, err := server.ReadResource(context.Background(), "nivora://workflows/runs/"+result.WorkflowRun.ID)
	if err != nil {
		t.Fatalf("ReadResource workflow run: %v", err)
	}
	for _, want := range []string{`"workflowRun"`, result.WorkflowRun.ID, result.PipelineRun.Run.ID, `"Succeeded"`, `"mutated": false`} {
		if !strings.Contains(resource.Text, want) {
			t.Fatalf("workflow run resource missing %q: %s", want, resource.Text)
		}
	}
}

func TestMCPPipelineRunDAGAliasIsReadOnly(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	result, err := server.services.Workflows.Run(context.Background(), workflowusecase.RunInput{
		Content: `
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: mcp-dag
on: [manual]
jobs:
  build:
    steps:
      - name: build
        run: echo build
  test:
    needs: [build]
    steps:
      - name: test
        run: echo test
`,
		ProjectID:        "project-mcp",
		Confirm:          true,
		AllowPipelineRun: true,
	}, server.services.Pipelines)
	if err != nil {
		t.Fatalf("run workflow: %v", err)
	}
	resource, err := server.ReadResource(context.Background(), "nivora://pipeline-runs/"+result.PipelineRun.Run.ID+"/dag")
	if err != nil {
		t.Fatalf("ReadResource pipeline-run dag: %v", err)
	}
	for _, want := range []string{`"nodes"`, `"edges"`, `"pipelineRun"`, `"jobRun"`, `"stepRun"`, `"mutated": false`} {
		if !strings.Contains(resource.Text, want) {
			t.Fatalf("pipeline dag missing %q: %s", want, resource.Text)
		}
	}
}

func TestMCPDeploymentAndReleasePlanResourcesAreReadOnly(t *testing.T) {
	server, deployments := newTestMCPServerAndDeploymentService(t, domainauth.RoleMaintainer, "mcp-local")
	manifest := writeMCPDeploymentManifest(t)
	deploymentResult, err := deployments.CreateAndRun(context.Background(), deploymentusecase.CreateRunInput{
		ProjectID: "project-mcp",
		Definition: deploymentusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Deployment",
			Metadata:   deploymentusecase.Metadata{Name: "mcp-plan"},
			Spec: deploymentusecase.Spec{
				Application: "demo",
				Environment: "staging",
				Target:      deploymentusecase.Target{Type: "kubernetes-yaml", Name: "dry-run", Namespace: "default"},
				Manifests:   []string{manifest},
				Options:     deploymentusecase.Options{DryRun: true, Apply: false},
			},
		},
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	deploymentPlan, err := server.ReadResource(context.Background(), "nivora://deployment-plans/"+deploymentResult.Record.Run.ID)
	if err != nil {
		t.Fatalf("ReadResource deployment plan: %v", err)
	}
	for _, want := range []string{`"deploymentPlan"`, deploymentResult.Record.Run.ID, `"mutated": false`} {
		if !strings.Contains(deploymentPlan.Text, want) {
			t.Fatalf("deployment plan missing %q: %s", want, deploymentPlan.Text)
		}
	}
	deploymentExplain, err := server.CallTool(context.Background(), "nivora_explain_deployment_plan", map[string]any{"id": deploymentResult.Record.Run.ID})
	if err != nil {
		t.Fatalf("explain deployment plan: %v", err)
	}
	for _, want := range []string{`"deploymentPlanId"`, deploymentResult.Record.Run.ID, `"guardrails"`, `"mutated": false`} {
		if deploymentExplain.IsError || !strings.Contains(deploymentExplain.Content[0].Text, want) {
			t.Fatalf("deployment plan explain missing %q: %#v", want, deploymentExplain)
		}
	}

	releaseDefinition := releaseusecase.Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "ReleaseOrchestration",
		Metadata:   releaseusecase.Metadata{Name: "mcp-release-plan"},
		Spec: releaseusecase.Spec{
			Environment: "staging",
			Strategy:    releaseusecase.StrategyPlanOnly,
			Release: artifactusecase.ReleaseDefinition{
				APIVersion: "nivora.io/v1alpha1",
				Kind:       "Release",
				Metadata:   artifactusecase.ReleaseMetadata{Name: "mcp-release"},
				Spec: artifactusecase.ReleaseSpec{
					Version:     "1.0.0",
					Application: "demo",
					Environment: "staging",
					Artifacts: []artifactusecase.ReleaseArtifactSpec{{
						Name:      "demo",
						Type:      "image",
						Required:  true,
						Reference: "registry.example.com/demo/app@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					}},
				},
			},
			Targets: []releaseusecase.TargetSpec{{
				Name: "noop",
				Type: "noop",
				Deployment: deploymentusecase.Definition{
					APIVersion: "nivora.io/v1alpha1",
					Kind:       "Deployment",
					Metadata:   deploymentusecase.Metadata{Name: "mcp-release-deploy"},
					Spec: deploymentusecase.Spec{
						Application: "demo",
						Environment: "staging",
						Target:      deploymentusecase.Target{Type: "kubernetes-yaml", Name: "dry-run", Namespace: "default"},
						Manifests:   []string{manifest},
						Options:     deploymentusecase.Options{DryRun: true, Apply: false},
					},
				},
			}},
		},
	}
	releasePlanRecord, err := server.services.Releases.Plan(context.Background(), releaseusecase.PlanInput{Definition: releaseDefinition, ProjectID: "project-mcp"})
	if err != nil {
		t.Fatalf("plan release: %v", err)
	}
	releasePlan, err := server.ReadResource(context.Background(), "nivora://release-plans/"+releasePlanRecord.Plan.ID)
	if err != nil {
		t.Fatalf("ReadResource release plan: %v", err)
	}
	for _, want := range []string{`"releasePlan"`, releasePlanRecord.Plan.ID, `"deploymentPlans"`, `"mutated": false`} {
		if !strings.Contains(releasePlan.Text, want) {
			t.Fatalf("release plan missing %q: %s", want, releasePlan.Text)
		}
	}
	releaseExplain, err := server.CallTool(context.Background(), "nivora_explain_release_plan", map[string]any{"id": releasePlanRecord.Plan.ID})
	if err != nil {
		t.Fatalf("explain release plan: %v", err)
	}
	for _, want := range []string{`"releasePlanId"`, releasePlanRecord.Plan.ID, `"targetCount"`, `"guardrails"`, `"mutated": false`} {
		if releaseExplain.IsError || !strings.Contains(releaseExplain.Content[0].Text, want) {
			t.Fatalf("release plan explain missing %q: %#v", want, releaseExplain)
		}
	}
}

func writeMCPDeploymentManifest(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "resources.yaml")
	if err := os.WriteFile(path, []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-plan
  namespace: default
data:
  app: demo
`), 0o600); err != nil {
		t.Fatalf("write deployment manifest: %v", err)
	}
	return path
}
