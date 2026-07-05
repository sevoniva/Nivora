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
		RepositoryID: "repo-durable",
		Path:         ".nivora/workflows/build.yaml",
		Ref:          "main",
	})
	if err != nil {
		t.Fatalf("create workflow plan: %v", err)
	}

	restartedPool := db.restart(t)
	service = workflowusecase.NewService(NewWorkflowStore(restartedPool))

	loaded, err := service.GetPlan(ctx, record.ID)
	if err != nil {
		t.Fatalf("reload workflow plan: %v", err)
	}
	if loaded.ID != record.ID || loaded.WorkflowID != record.WorkflowID || loaded.Plan.PlanID != record.ID {
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
}
