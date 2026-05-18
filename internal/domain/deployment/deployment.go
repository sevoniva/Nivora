package deployment

import "time"

type DeploymentRunStatus string

const (
	DeploymentRunCreated         DeploymentRunStatus = "Created"
	DeploymentRunPlanning        DeploymentRunStatus = "Planning"
	DeploymentRunWaitingApproval DeploymentRunStatus = "WaitingApproval"
	DeploymentRunPreChecking     DeploymentRunStatus = "PreChecking"
	DeploymentRunDeploying       DeploymentRunStatus = "Deploying"
	DeploymentRunVerifying       DeploymentRunStatus = "Verifying"
	DeploymentRunSucceeded       DeploymentRunStatus = "Succeeded"
	DeploymentRunFailed          DeploymentRunStatus = "Failed"
	DeploymentRunRollingBack     DeploymentRunStatus = "RollingBack"
	DeploymentRunRolledBack      DeploymentRunStatus = "RolledBack"
	DeploymentRunCanceled        DeploymentRunStatus = "Canceled"
)

func (s DeploymentRunStatus) Valid() bool {
	switch s {
	case DeploymentRunCreated, DeploymentRunPlanning, DeploymentRunWaitingApproval,
		DeploymentRunPreChecking, DeploymentRunDeploying, DeploymentRunVerifying,
		DeploymentRunSucceeded, DeploymentRunFailed, DeploymentRunRollingBack,
		DeploymentRunRolledBack, DeploymentRunCanceled:
		return true
	default:
		return false
	}
}

type DeploymentRun struct {
	ID              string
	ReleaseID       string
	EnvironmentID   string
	ReleaseTargetID string
	Status          DeploymentRunStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
