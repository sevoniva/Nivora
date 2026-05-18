package deployment

import (
	"testing"
	"time"

	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
)

func TestDeploymentRunTransitions(t *testing.T) {
	run := domaindeployment.DeploymentRun{Status: domaindeployment.DeploymentRunCreated}
	now := time.Now()

	for _, status := range []domaindeployment.DeploymentRunStatus{
		domaindeployment.DeploymentRunPlanning,
		domaindeployment.DeploymentRunPreChecking,
		domaindeployment.DeploymentRunVerifying,
		domaindeployment.DeploymentRunSucceeded,
	} {
		if err := transitionDeploymentRun(&run, status, now, ""); err != nil {
			t.Fatalf("transition to %s: %v", status, err)
		}
	}
	if run.StartedAt == nil {
		t.Fatal("expected started_at")
	}
	if run.FinishedAt == nil {
		t.Fatal("expected finished_at")
	}
	if err := transitionDeploymentRun(&run, domaindeployment.DeploymentRunPlanning, now, ""); err == nil {
		t.Fatal("expected terminal transition error")
	}
}

func TestDeploymentRunCancelTransition(t *testing.T) {
	run := domaindeployment.DeploymentRun{Status: domaindeployment.DeploymentRunCreated}
	if err := transitionDeploymentRun(&run, domaindeployment.DeploymentRunCanceled, time.Now(), "canceled"); err != nil {
		t.Fatalf("cancel transition: %v", err)
	}
	if run.Reason != "canceled" {
		t.Fatalf("reason = %q", run.Reason)
	}
}
