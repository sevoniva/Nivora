package workflow

import (
	"context"
	"strings"
	"testing"
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
