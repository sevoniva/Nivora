package runtimecenter

import (
	"context"
	"testing"
	"time"

	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
)

func TestStatusSummarizesRuntimeRecoveryAcrossServices(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	service := NewService(
		fakePipelineRecovery{status: pipelineusecase.RuntimeRecoverySummary{
			QueuedPipelineRuns:       2,
			StaleRunningPipelineRuns: 1,
			PendingOutboxEvents:      1,
			CheckedAt:                now,
		}},
		fakeDeploymentRecovery{status: deploymentusecase.RuntimeRecoverySummary{
			NonTerminalDeploymentRuns: 3,
			StaleDeploymentRuns:       1,
			Actions: []deploymentusecase.RuntimeRecoveryAction{{
				SubjectType:    "deploymentRun",
				SubjectID:      "dep-1",
				Status:         "Deploying",
				SafeNextAction: "inspect",
				ObservedAt:     now,
			}},
		}},
		fakeReleaseRecovery{status: releaseusecase.RuntimeRecoverySummary{
			NonTerminalReleaseExecutions: 1,
			StaleReleaseExecutions:       1,
		}},
	)
	service.now = func() time.Time { return now }

	summary, err := service.Status(context.Background(), Options{})
	if err != nil {
		t.Fatalf("runtime status: %v", err)
	}
	if summary.Status != StatusWarning {
		t.Fatalf("status = %s, want %s", summary.Status, StatusWarning)
	}
	if summary.QueuedPipelineRuns != 2 || summary.StaleDeploymentRuns != 1 || summary.StaleReleaseExecutions != 1 {
		t.Fatalf("summary did not aggregate all services: %#v", summary)
	}
	if len(summary.SafeNextActions) != 1 || summary.SafeNextActions[0].SubjectID != "dep-1" {
		t.Fatalf("safe next actions = %#v", summary.SafeNextActions)
	}
	if !summary.PlanOnly || summary.Reconciled {
		t.Fatalf("status should be plan-only read: %#v", summary)
	}
}

func TestReconcileKeepsDeploymentAndReleasePlanOnly(t *testing.T) {
	service := NewService(
		fakePipelineRecovery{reconcile: pipelineusecase.RuntimeRecoverySummary{RecoveredPipelineRuns: 1}},
		fakeDeploymentRecovery{status: deploymentusecase.RuntimeRecoverySummary{StaleDeploymentRuns: 1}},
		fakeReleaseRecovery{status: releaseusecase.RuntimeRecoverySummary{StaleReleaseExecutions: 1}},
	)

	summary, err := service.Reconcile(context.Background(), Options{})
	if err != nil {
		t.Fatalf("runtime reconcile: %v", err)
	}
	if !summary.Reconciled || summary.PlanOnly {
		t.Fatalf("reconcile flags = %#v", summary)
	}
	if summary.RecoveredPipelineRuns != 1 {
		t.Fatalf("pipeline recovery not preserved: %#v", summary)
	}
	if summary.StaleDeploymentRuns != 1 || summary.StaleReleaseExecutions != 1 {
		t.Fatalf("deployment/release status missing after reconcile: %#v", summary)
	}
}

type fakePipelineRecovery struct {
	status    pipelineusecase.RuntimeRecoverySummary
	reconcile pipelineusecase.RuntimeRecoverySummary
}

func (f fakePipelineRecovery) RuntimeStatus(context.Context) (pipelineusecase.RuntimeRecoverySummary, error) {
	return f.status, nil
}

func (f fakePipelineRecovery) ReconcileRuntime(context.Context, pipelineusecase.RuntimeRecoveryOptions) (pipelineusecase.RuntimeRecoverySummary, error) {
	return f.reconcile, nil
}

type fakeDeploymentRecovery struct {
	status deploymentusecase.RuntimeRecoverySummary
}

func (f fakeDeploymentRecovery) RuntimeStatus(context.Context, time.Duration, int) (deploymentusecase.RuntimeRecoverySummary, error) {
	return f.status, nil
}

type fakeReleaseRecovery struct {
	status releaseusecase.RuntimeRecoverySummary
}

func (f fakeReleaseRecovery) RuntimeStatus(context.Context, time.Duration, int) (releaseusecase.RuntimeRecoverySummary, error) {
	return f.status, nil
}
