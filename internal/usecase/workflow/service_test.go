package workflow

import (
	"context"
	"io"
	"strings"
	"testing"

	domainevent "github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	"github.com/sevoniva/nivora/internal/ports/executor"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func TestServicePlanStoresRedactedPlanMetadataOnly(t *testing.T) {
	service := NewService(NewMemoryStore())
	record, err := service.Plan(context.Background(), PlanInput{
		Content: `apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: Stored Workflow
on: manual
env:
  API_TOKEN: secretRef:workflow-token
jobs:
  build:
    runsOn: [linux]
    steps:
      - name: build
        run: go test ./...
`,
		RepositoryID: "repo-a",
		Path:         ".nivora/workflows/build.yaml",
		Ref:          "main",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if record.ID == "" || record.WorkflowID != "workflow-stored-workflow" || record.ContentHash == "" {
		t.Fatalf("record metadata not populated: %#v", record)
	}
	if record.Plan.PlanID != record.ID || record.Plan.RepositoryID != "repo-a" || record.Plan.SourcePath != ".nivora/workflows/build.yaml" {
		t.Fatalf("plan metadata not populated: %#v", record.Plan)
	}
	if strings.Contains(record.Plan.Steps[0].Env["API_TOKEN"], "workflow-token") {
		t.Fatalf("plan env leaked secret ref target: %#v", record.Plan.Steps[0].Env)
	}
	loaded, err := service.GetPlan(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("GetPlan: %v", err)
	}
	if loaded.ID != record.ID || loaded.Plan.ContentHash != record.ContentHash {
		t.Fatalf("loaded record = %#v", loaded)
	}
	latest, err := service.GetLatestPlan(context.Background(), record.WorkflowID)
	if err != nil {
		t.Fatalf("GetLatestPlan: %v", err)
	}
	if latest.ID != record.ID {
		t.Fatalf("latest = %#v", latest)
	}
	list, err := service.ListPlans(context.Background(), PlanListFilter{RepositoryID: "repo-a"})
	if err != nil || len(list) != 1 || list[0].ID != record.ID {
		t.Fatalf("ListPlans = %#v err=%v", list, err)
	}
}

func TestServicePlanRejectsInlineSecretLikeEnv(t *testing.T) {
	service := NewService(NewMemoryStore())
	_, err := service.Plan(context.Background(), PlanInput{Content: `apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: Unsafe
on: manual
env:
  API_TOKEN: inline-token-value
jobs:
  build:
    steps:
      - run: go test ./...
`})
	if err == nil || !strings.Contains(err.Error(), "must use secretRef") {
		t.Fatalf("expected secret-like env rejection, got %v", err)
	}
}

func TestServiceRunRequiresExplicitPipelineRunAllow(t *testing.T) {
	service := NewService(NewMemoryStore())
	_, err := service.Run(context.Background(), RunInput{Content: executableWorkflow(t)}, newWorkflowPipelineService())
	if err == nil || !strings.Contains(err.Error(), "confirm=true") {
		t.Fatalf("expected confirmation error, got %v", err)
	}
}

func TestServiceRunCreatesQueuedPipelineRunAndWorkflowRun(t *testing.T) {
	service := NewService(NewMemoryStore())
	pipelines := newWorkflowPipelineService()
	result, err := service.Run(context.Background(), RunInput{
		Content:          executableWorkflow(t),
		RepositoryID:     "repo-a",
		Ref:              "main",
		ProjectID:        "project-a",
		EnvironmentID:    "env-dev",
		ActorID:          "user-a",
		CorrelationID:    "corr-workflow",
		Confirm:          true,
		AllowPipelineRun: true,
	}, pipelines)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.WorkflowRun.ID == "" || result.WorkflowRun.WorkflowPlanID == "" || result.WorkflowRun.PipelineRunID == "" {
		t.Fatalf("workflow run metadata not populated: %#v", result.WorkflowRun)
	}
	if result.WorkflowRun.Status != RunQueued {
		t.Fatalf("workflow run status = %s", result.WorkflowRun.Status)
	}
	if result.PipelineRun.Run.Status != domainpipeline.PipelineRunQueued {
		t.Fatalf("pipeline run status = %s", result.PipelineRun.Run.Status)
	}
	if result.PipelineRun.Pipeline.ProjectID != "project-a" || result.PipelineRun.Pipeline.Metadata["environmentId"] != "env-dev" {
		t.Fatalf("pipeline ownership metadata missing: %#v", result.PipelineRun.Pipeline)
	}
	loaded, err := service.GetRun(context.Background(), result.WorkflowRun.ID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if loaded.PipelineRunID != result.PipelineRun.Run.ID {
		t.Fatalf("loaded workflow run = %#v", loaded)
	}
	runs, err := service.ListRuns(context.Background(), RunListFilter{RepositoryID: "repo-a", Status: RunQueued})
	if err != nil || len(runs) != 1 || runs[0].ID != result.WorkflowRun.ID {
		t.Fatalf("ListRuns = %#v err=%v", runs, err)
	}
}

func executableWorkflow(t *testing.T) string {
	t.Helper()
	return `apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: Executable Workflow
on: manual
jobs:
  test:
    steps:
      - name: test
        run: echo ok
`
}

type noopBus struct{}

func (noopBus) Publish(context.Context, domainevent.Event) error { return nil }
func (noopBus) Subscribe(context.Context, string) (<-chan domainevent.Event, error) {
	ch := make(chan domainevent.Event)
	close(ch)
	return ch, nil
}

type noopExecutor struct{}

func (noopExecutor) Prepare(context.Context, executor.JobContext) error { return nil }
func (noopExecutor) Run(context.Context, executor.Command) (executor.Result, error) {
	return executor.Result{ExitCode: 0}, nil
}
func (noopExecutor) Cancel(context.Context, string) error { return nil }
func (noopExecutor) Logs(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func newWorkflowPipelineService() *pipelineusecase.Service {
	return pipelineusecase.NewService(
		pipelineusecase.NewMemoryStore(),
		pipelineusecase.NewLocalRunner("test-runner", noopExecutor{}),
		noopBus{},
	)
}
