package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
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

	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
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

	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	projectID, _ := createMCPCatalogAndPipelineFixture(t, server)
	repositories, err := server.services.Catalog.ListRepositories(context.Background(), projectID)
	if err != nil {
		t.Fatalf("list catalog repositories: %v", err)
	}
	if len(repositories) == 0 {
		t.Fatal("expected catalog repository fixture")
	}
	repositoryID := repositories[0].ID
	saved, err := server.services.Repositories.SaveRepository(context.Background(), repositoryusecase.Repository{
		ID:            repositoryID,
		Name:          repositories[0].Name,
		Provider:      repositoryusecase.ProviderLocal,
		URL:           root,
		DefaultBranch: "HEAD",
		ProjectID:     projectID,
		Status:        repositoryusecase.RepositoryStatusActive,
	})
	if err != nil {
		t.Fatalf("save repository usecase record: %v", err)
	}
	if _, err := server.services.Repositories.CreateSnapshot(context.Background(), repositoryusecase.SnapshotInput{Repository: saved, Ref: "HEAD", LocalPath: root}); err != nil {
		t.Fatalf("create repository snapshot: %v", err)
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

	result, err := server.CallTool(context.Background(), "nivora_repository_devops_plan", map[string]any{"repositoryId": repositoryID})
	if err != nil {
		t.Fatalf("repository devops plan transport error: %v", err)
	}
	if result.IsError || len(result.Content) == 0 {
		t.Fatalf("repository devops plan result = %#v", result)
	}
	body := result.Content[0].Text
	for _, want := range []string{`"mutated": false`, `"releaseCandidate"`, "go build ./...", "metadata-only"} {
		if !strings.Contains(body, want) {
			t.Fatalf("devops plan tool body missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("devops plan tool leaked .env content: %s", body)
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
	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
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
