package postgres

import (
	"context"
	"strings"
	"testing"

	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
)

func TestWorkflowStoreImplementsInterface(t *testing.T) {
	var _ workflowusecase.Store = (*WorkflowStore)(nil)
}

func TestWorkflowPlanMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000018_workflow_plan_persistence.up.sql")
	down := readMigration(t, "000018_workflow_plan_persistence.down.sql")

	if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS workflow_plan_records") {
		t.Fatal("up migration missing workflow_plan_records table")
	}
	if !strings.Contains(down, "DROP TABLE IF EXISTS workflow_plan_records") {
		t.Fatal("down migration missing workflow_plan_records drop")
	}
	for _, index := range []string{
		"idx_workflow_plan_records_workflow_created",
		"idx_workflow_plan_records_repository_created",
		"idx_workflow_plan_records_content_hash",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
	}
}

func TestWorkflowRunMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000019_workflow_run_persistence.up.sql")
	down := readMigration(t, "000019_workflow_run_persistence.down.sql")

	if !strings.Contains(up, "CREATE TABLE IF NOT EXISTS workflow_run_records") {
		t.Fatal("up migration missing workflow_run_records table")
	}
	if !strings.Contains(down, "DROP TABLE IF EXISTS workflow_run_records") {
		t.Fatal("down migration missing workflow_run_records drop")
	}
	for _, index := range []string{
		"idx_workflow_run_records_workflow_created",
		"idx_workflow_run_records_plan_created",
		"idx_workflow_run_records_repository_created",
		"idx_workflow_run_records_project_status",
		"idx_workflow_run_records_pipeline_run",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("up migration missing index %s", index)
		}
	}
}

func TestWorkflowSourceMetadataMigrationIsReversibleAndIndexed(t *testing.T) {
	up := readMigration(t, "000021_workflow_pipeline_source_metadata.up.sql")
	down := readMigration(t, "000021_workflow_pipeline_source_metadata.down.sql")

	for _, value := range []string{
		"workflow_plan_records",
		"workflow_run_records",
		"runtime_pipeline_runs",
		"runtime_job_runs",
		"repository_snapshot_id",
		"workflow_job_id",
	} {
		if !strings.Contains(up, value) {
			t.Fatalf("source metadata up migration missing %s", value)
		}
		if !strings.Contains(down, value) {
			t.Fatalf("source metadata down migration missing %s", value)
		}
	}
	for _, index := range []string{
		"idx_workflow_plan_records_snapshot_created",
		"idx_workflow_run_records_snapshot_created",
	} {
		if !strings.Contains(up, index) {
			t.Fatalf("source metadata up migration missing index %s", index)
		}
		if !strings.Contains(down, index) {
			t.Fatalf("source metadata down migration missing index %s", index)
		}
	}
}

func TestPostgresIntegrationWorkflowPlanRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()
	ctx := context.Background()
	service := workflowusecase.NewService(NewWorkflowStore(db.pool))

	record, err := service.Plan(ctx, workflowusecase.PlanInput{
		Content: `apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: Durable Workflow
on: manual
env:
  API_TOKEN: secretRef:workflow-token
jobs:
  build:
    runsOn: [linux]
    steps:
      - name: test
        run: go test ./...
`,
		RepositoryID:         "repo-durable",
		RepositorySnapshotID: "snap-durable",
		Path:                 ".nivora/workflows/build.yaml",
		Ref:                  "main",
	})
	if err != nil {
		t.Fatalf("create workflow plan: %v", err)
	}

	restartedPool := db.restart(t)
	store := NewWorkflowStore(restartedPool)
	service = workflowusecase.NewService(store)

	loaded, err := service.GetPlan(ctx, record.ID)
	if err != nil {
		t.Fatalf("reload workflow plan: %v", err)
	}
	if loaded.ID != record.ID || loaded.WorkflowID != record.WorkflowID || loaded.RepositorySnapshotID != "snap-durable" || loaded.Plan.PlanID != record.ID || loaded.Plan.RepositorySnapshotID != "snap-durable" {
		t.Fatalf("loaded workflow plan = %#v", loaded)
	}
	if strings.Contains(loaded.Plan.Steps[0].Env["API_TOKEN"], "workflow-token") {
		t.Fatalf("workflow plan leaked secret ref target: %#v", loaded.Plan.Steps[0].Env)
	}
	latest, err := service.GetLatestPlan(ctx, record.WorkflowID)
	if err != nil {
		t.Fatalf("reload latest workflow plan: %v", err)
	}
	if latest.ID != record.ID {
		t.Fatalf("latest workflow plan = %#v", latest)
	}
	plans, err := service.ListPlans(ctx, workflowusecase.PlanListFilter{RepositoryID: "repo-durable"})
	if err != nil || len(plans) != 1 || plans[0].ID != record.ID {
		t.Fatalf("list workflow plans = %#v err=%v", plans, err)
	}

	run := workflowusecase.RunRecord{
		ID:                   "wrun-durable",
		WorkflowID:           record.WorkflowID,
		WorkflowPlanID:       record.ID,
		RepositoryID:         "repo-durable",
		RepositorySnapshotID: "snap-durable",
		PipelineRunID:        "prun-durable",
		PipelineID:           "pipe-durable",
		ProjectID:            "project-durable",
		EnvironmentID:        "env-dev",
		Ref:                  "main",
		Status:               workflowusecase.RunQueued,
		Warnings:             []string{"queued only"},
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.CreatedAt,
	}
	if err := store.SaveRun(ctx, run); err != nil {
		t.Fatalf("save workflow run: %v", err)
	}

	store = NewWorkflowStore(db.restart(t))
	loadedRun, err := store.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("reload workflow run: %v", err)
	}
	if loadedRun.PipelineRunID != run.PipelineRunID || loadedRun.RepositorySnapshotID != "snap-durable" || loadedRun.Status != workflowusecase.RunQueued {
		t.Fatalf("loaded workflow run = %#v", loadedRun)
	}
	runs, err := store.ListRuns(ctx, workflowusecase.RunListFilter{RepositoryID: "repo-durable", Status: workflowusecase.RunQueued})
	if err != nil || len(runs) != 1 || runs[0].ID != run.ID {
		t.Fatalf("list workflow runs = %#v err=%v", runs, err)
	}
}
