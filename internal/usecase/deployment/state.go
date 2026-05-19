package deployment

import (
	"fmt"
	"time"

	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
)

func transitionDeploymentRun(run *domaindeployment.DeploymentRun, next domaindeployment.DeploymentRunStatus, now time.Time, reason string) error {
	if !canTransitionDeploymentRun(run.Status, next) {
		return fmt.Errorf("invalid DeploymentRun transition from %s to %s", run.Status, next)
	}
	run.Status = next
	run.UpdatedAt = now
	if next == domaindeployment.DeploymentRunPlanning && run.StartedAt == nil {
		run.StartedAt = &now
	}
	if isTerminalDeploymentStatus(next) {
		run.FinishedAt = &now
	}
	if reason != "" {
		run.Reason = reason
	}
	return nil
}

func canTransitionDeploymentRun(from domaindeployment.DeploymentRunStatus, to domaindeployment.DeploymentRunStatus) bool {
	switch from {
	case domaindeployment.DeploymentRunCreated:
		return to == domaindeployment.DeploymentRunPlanning || to == domaindeployment.DeploymentRunCanceled
	case domaindeployment.DeploymentRunPlanning:
		return to == domaindeployment.DeploymentRunPreChecking || to == domaindeployment.DeploymentRunFailed || to == domaindeployment.DeploymentRunCanceled
	case domaindeployment.DeploymentRunPreChecking:
		return to == domaindeployment.DeploymentRunVerifying || to == domaindeployment.DeploymentRunDeploying ||
			to == domaindeployment.DeploymentRunWaitingApproval ||
			to == domaindeployment.DeploymentRunFailed || to == domaindeployment.DeploymentRunCanceled
	case domaindeployment.DeploymentRunWaitingApproval:
		return to == domaindeployment.DeploymentRunVerifying || to == domaindeployment.DeploymentRunDeploying ||
			to == domaindeployment.DeploymentRunFailed || to == domaindeployment.DeploymentRunCanceled
	case domaindeployment.DeploymentRunDeploying:
		return to == domaindeployment.DeploymentRunVerifying || to == domaindeployment.DeploymentRunSucceeded ||
			to == domaindeployment.DeploymentRunFailed || to == domaindeployment.DeploymentRunRollingBack ||
			to == domaindeployment.DeploymentRunCanceled
	case domaindeployment.DeploymentRunVerifying:
		return to == domaindeployment.DeploymentRunDeploying || to == domaindeployment.DeploymentRunSucceeded ||
			to == domaindeployment.DeploymentRunFailed || to == domaindeployment.DeploymentRunCanceled
	case domaindeployment.DeploymentRunRollingBack:
		return to == domaindeployment.DeploymentRunRolledBack || to == domaindeployment.DeploymentRunFailed
	case domaindeployment.DeploymentRunSucceeded:
		return to == domaindeployment.DeploymentRunRollingBack
	default:
		return false
	}
}

func isTerminalDeploymentStatus(status domaindeployment.DeploymentRunStatus) bool {
	switch status {
	case domaindeployment.DeploymentRunSucceeded, domaindeployment.DeploymentRunFailed,
		domaindeployment.DeploymentRunRolledBack, domaindeployment.DeploymentRunCanceled:
		return true
	default:
		return false
	}
}
