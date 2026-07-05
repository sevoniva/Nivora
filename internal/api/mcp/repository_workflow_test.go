package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
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
	resource, err := server.ReadResource(context.Background(), "nivora://workflows/"+record.ID+"/plan")
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if !strings.Contains(resource.Text, `"workflowPlan"`) || !strings.Contains(resource.Text, `"mutated": false`) {
		t.Fatalf("workflow plan resource = %#v", resource)
	}
}
