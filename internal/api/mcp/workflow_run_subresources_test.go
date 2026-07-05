package mcp

import (
	"context"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
)

func TestMCPWorkflowRunSubResourcesAreReadOnlyAndLinkedToPipelineRun(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	result, err := server.services.Workflows.Run(context.Background(), workflowusecase.RunInput{
		Content: `
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: mcp-run-sub-single
on: [manual]
jobs:
  test:
    steps:
      - run: echo mcp
`,
		ProjectID:        "project-mcp-sub",
		EnvironmentID:    "env-mcp-sub",
		Confirm:          true,
		AllowPipelineRun: true,
	}, server.services.Pipelines)
	if err != nil {
		t.Fatalf("store workflow run: %v", err)
	}
	if _, err := server.services.Pipelines.ProcessQueued(context.Background(), 1); err != nil {
		t.Fatalf("process linked PipelineRun: %v", err)
	}
	pipelineID := result.PipelineRun.Run.ID
	if _, err := server.services.Pipelines.RecordArtifact(context.Background(), pipelineusecase.PipelineArtifact{
		PipelineRunID: pipelineID,
		Name:          "build-output",
		Type:          "binary",
		SizeBytes:     128,
		ContentHash:   "sha256:placeholder-hash",
		StorageRef:    "local://artifacts/build-output",
	}); err != nil {
		t.Fatalf("record artifact: %v", err)
	}
	if _, err := server.services.Pipelines.RecordAnnotation(context.Background(), pipelineusecase.StepAnnotation{
		PipelineRunID: pipelineID,
		Level:         "warning",
		Title:         "deprecation",
		Message:       "step uses deprecated syntax",
	}); err != nil {
		t.Fatalf("record annotation: %v", err)
	}
	workflowID := result.WorkflowRun.ID

	cases := []struct {
		name        string
		uri         string
		wantSubstrs []string
	}{
		{
			name:        "timeline alias workflow-runs",
			uri:         "nivora://workflow-runs/" + workflowID + "/timeline",
			wantSubstrs: []string{`"type"`, `"subject": "` + pipelineID + `"`},
		},
		{
			name:        "timeline workflows/runs",
			uri:         "nivora://workflows/runs/" + workflowID + "/timeline",
			wantSubstrs: []string{`"type"`, `"subject": "` + pipelineID + `"`},
		},
		{
			name:        "logs alias workflow-runs",
			uri:         "nivora://workflow-runs/" + workflowID + "/logs",
			wantSubstrs: []string{`"pipelineRunId": "` + pipelineID + `"`},
		},
		{
			name: "run metadata alias workflow-runs",
			uri:  "nivora://workflow-runs/" + workflowID,
			wantSubstrs: []string{
				`"workflowRun"`,
				workflowID,
				`"mutated": false`,
			},
		},
		{
			name: "artifacts alias workflow-runs",
			uri:  "nivora://workflow-runs/" + workflowID + "/artifacts",
			wantSubstrs: []string{
				`"workflowRunId": "` + workflowID + `"`,
				`"pipelineRunId": "` + pipelineID + `"`,
				`"build-output"`,
				`"mutated": false`,
			},
		},
		{
			name: "annotations alias workflow-runs",
			uri:  "nivora://workflow-runs/" + workflowID + "/annotations",
			wantSubstrs: []string{
				`"workflowRunId": "` + workflowID + `"`,
				`"pipelineRunId": "` + pipelineID + `"`,
				`"deprecation"`,
				`"mutated": false`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := server.ReadResource(context.Background(), tc.uri)
			if err != nil {
				t.Fatalf("ReadResource %s: %v", tc.uri, err)
			}
			for _, want := range tc.wantSubstrs {
				if !strings.Contains(out.Text, want) {
					t.Fatalf("%s missing %q: %s", tc.uri, want, out.Text)
				}
			}
		})
	}
}

func TestMCPWorkflowRunSubResourcesRejectUnknownRun(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	for _, kind := range []string{"timeline", "logs", "artifacts", "annotations"} {
		uri := "nivora://workflow-runs/wf-missing/" + kind
		_, err := server.ReadResource(context.Background(), uri)
		if err == nil {
			t.Fatalf("ReadResource %s expected error for unknown run, got nil", uri)
		}
	}
}

func TestMCPRepositoryWorkflowsResourceListsByRepository(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleDeveloper, "mcp-local")
	projectID, _ := createMCPCatalogAndPipelineFixture(t, server)
	repositories, err := server.services.Catalog.ListRepositories(context.Background(), projectID)
	if err != nil {
		t.Fatalf("list repositories: %v", err)
	}
	if len(repositories) == 0 {
		t.Fatal("expected catalog repository fixture")
	}
	repositoryID := repositories[0].ID

	// Plan a workflow against the repository so ListWorkflows returns it.
	plan, err := server.services.Workflows.Plan(context.Background(), workflowusecase.PlanInput{
		Content: `
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: repo-workflow
on: [manual]
jobs:
  build:
    steps:
      - run: echo build
`,
		RepositoryID: repositoryID,
	})
	if err != nil {
		t.Fatalf("plan workflow: %v", err)
	}
	_ = plan

	out, err := server.ReadResource(context.Background(), "nivora://repositories/"+repositoryID+"/workflows")
	if err != nil {
		t.Fatalf("ReadResource repository workflows: %v", err)
	}
	for _, want := range []string{
		`"repositoryId": "` + repositoryID + `"`,
		`"workflows"`,
		`"mutated": false`,
		`"repo-workflow"`,
	} {
		if !strings.Contains(out.Text, want) {
			t.Fatalf("repository workflows missing %q: %s", want, out.Text)
		}
	}
}
