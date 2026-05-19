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
	ID                  string              `json:"id"`
	ReleaseID           string              `json:"releaseId,omitempty"`
	ApplicationID       string              `json:"applicationId,omitempty"`
	CorrelationID       string              `json:"correlationId,omitempty"`
	EnvironmentID       string              `json:"environmentId"`
	ReleaseTargetID     string              `json:"releaseTargetId"`
	TargetType          string              `json:"targetType"`
	Status              DeploymentRunStatus `json:"status"`
	Reason              string              `json:"reason,omitempty"`
	OwnerID             string              `json:"ownerId,omitempty"`
	LeaseExpiresAt      *time.Time          `json:"leaseExpiresAt,omitempty"`
	Attempt             int                 `json:"attempt,omitempty"`
	HeartbeatAt         *time.Time          `json:"heartbeatAt,omitempty"`
	ManifestSnapshotRef string              `json:"manifestSnapshotRef,omitempty"`
	ArtifactReferences  []string            `json:"artifactReferences,omitempty"`
	StartedAt           *time.Time          `json:"startedAt,omitempty"`
	FinishedAt          *time.Time          `json:"finishedAt,omitempty"`
	CreatedAt           time.Time           `json:"createdAt"`
	UpdatedAt           time.Time           `json:"updatedAt"`
}

type DeploymentStep struct {
	ID              string              `json:"id"`
	DeploymentRunID string              `json:"deploymentRunId"`
	Name            string              `json:"name"`
	Status          DeploymentRunStatus `json:"status"`
	Reason          string              `json:"reason,omitempty"`
	StartedAt       *time.Time          `json:"startedAt,omitempty"`
	FinishedAt      *time.Time          `json:"finishedAt,omitempty"`
	CreatedAt       time.Time           `json:"createdAt"`
	UpdatedAt       time.Time           `json:"updatedAt"`
}

type RollbackRecord struct {
	ID                  string    `json:"id"`
	DeploymentRunID     string    `json:"deploymentRunId"`
	Strategy            string    `json:"strategy"`
	Status              string    `json:"status"`
	TargetType          string    `json:"targetType,omitempty"`
	TargetName          string    `json:"targetName,omitempty"`
	ManifestSnapshotRef string    `json:"manifestSnapshotRef,omitempty"`
	ResourceRefs        []string  `json:"resourceRefs,omitempty"`
	Reason              string    `json:"reason,omitempty"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}
