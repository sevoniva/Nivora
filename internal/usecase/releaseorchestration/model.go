package releaseorchestration

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

type ExecutionStatus string

const (
	ExecutionCreated            ExecutionStatus = "Created"
	ExecutionPlanning           ExecutionStatus = "Planning"
	ExecutionWaitingApproval    ExecutionStatus = "WaitingApproval"
	ExecutionRunning            ExecutionStatus = "Running"
	ExecutionPartiallySucceeded ExecutionStatus = "PartiallySucceeded"
	ExecutionSucceeded          ExecutionStatus = "Succeeded"
	ExecutionFailed             ExecutionStatus = "Failed"
	ExecutionCanceling          ExecutionStatus = "Canceling"
	ExecutionCanceled           ExecutionStatus = "Canceled"
	ExecutionRollingBack        ExecutionStatus = "RollingBack"
	ExecutionRolledBack         ExecutionStatus = "RolledBack"
)

type ExecutionStrategy string

const (
	StrategyPlanOnly   ExecutionStrategy = "plan-only"
	StrategySequential ExecutionStrategy = "sequential"
	StrategyParallel   ExecutionStrategy = "parallel"
)

type Definition struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Metadata   Metadata `json:"metadata" yaml:"metadata"`
	Spec       Spec     `json:"spec" yaml:"spec"`
}

type Metadata struct {
	Name string `json:"name" yaml:"name"`
}

type Spec struct {
	Release           artifactusecase.ReleaseDefinition `json:"release,omitempty" yaml:"release,omitempty"`
	ReleaseID         string                            `json:"releaseId,omitempty" yaml:"releaseId,omitempty"`
	Environment       string                            `json:"environment" yaml:"environment"`
	Strategy          ExecutionStrategy                 `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	Concurrency       int                               `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	ContinueOnFailure bool                              `json:"continueOnFailure,omitempty" yaml:"continueOnFailure,omitempty"`
	ApprovalRequired  bool                              `json:"approvalRequired,omitempty" yaml:"approvalRequired,omitempty"`
	Targets           []TargetSpec                      `json:"targets" yaml:"targets"`
}

type TargetSpec struct {
	Name         string                       `json:"name" yaml:"name"`
	Type         string                       `json:"type" yaml:"type"`
	Order        int                          `json:"order,omitempty" yaml:"order,omitempty"`
	Dependencies []string                     `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
	Enabled      *bool                        `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	Labels       map[string]string            `json:"labels,omitempty" yaml:"labels,omitempty"`
	Deployment   deploymentusecase.Definition `json:"deployment" yaml:"deployment"`
}

type ReleasePlan struct {
	ID              string                             `json:"id"`
	ReleaseID       string                             `json:"releaseId"`
	EnvironmentID   string                             `json:"environmentId"`
	EnvironmentName string                             `json:"environmentName"`
	Targets         []environment.ReleaseTarget        `json:"targets"`
	ArtifactSummary []string                           `json:"artifactSummary,omitempty"`
	PolicyResults   []PolicyResult                     `json:"policyResults,omitempty"`
	DeploymentPlans []deploymentusecase.DeploymentPlan `json:"deploymentPlans"`
	Ordering        []string                           `json:"ordering,omitempty"`
	Concurrency     int                                `json:"concurrency"`
	Strategy        ExecutionStrategy                  `json:"strategy"`
	Warnings        []string                           `json:"warnings,omitempty"`
	CreatedAt       time.Time                          `json:"createdAt"`
}

type PolicyResult struct {
	Target  string   `json:"target,omitempty"`
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

type ReleaseExecution struct {
	ID               string            `json:"id"`
	ReleaseID        string            `json:"releaseId"`
	EnvironmentID    string            `json:"environmentId"`
	EnvironmentName  string            `json:"environmentName"`
	Status           ExecutionStatus   `json:"status"`
	DeploymentRunIDs []string          `json:"deploymentRunIds,omitempty"`
	Targets          []TargetExecution `json:"targets,omitempty"`
	StartedAt        *time.Time        `json:"startedAt,omitempty"`
	FinishedAt       *time.Time        `json:"finishedAt,omitempty"`
	Reason           string            `json:"reason,omitempty"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

type TargetExecution struct {
	TargetID        string          `json:"targetId"`
	TargetName      string          `json:"targetName"`
	TargetType      string          `json:"targetType"`
	DeploymentRunID string          `json:"deploymentRunId,omitempty"`
	Status          ExecutionStatus `json:"status"`
	Order           int             `json:"order"`
	Dependencies    []string        `json:"dependencies,omitempty"`
	Warnings        []string        `json:"warnings,omitempty"`
}

type PlanRecord struct {
	Definition Definition       `json:"definition,omitempty"`
	Release    release.Release  `json:"release"`
	Plan       ReleasePlan      `json:"plan"`
	Events     []event.Event    `json:"events,omitempty"`
	Audits     []audit.AuditLog `json:"audits,omitempty"`
}

type ExecutionRecord struct {
	Definition  Definition                    `json:"definition,omitempty"`
	Release     release.Release               `json:"release"`
	Plan        ReleasePlan                   `json:"plan"`
	Execution   ReleaseExecution              `json:"execution"`
	Deployments []deploymentusecase.RunRecord `json:"deployments,omitempty"`
	Events      []event.Event                 `json:"events,omitempty"`
	Audits      []audit.AuditLog              `json:"audits,omitempty"`
}

type TimelineEntry struct {
	Type    string    `json:"type"`
	Time    time.Time `json:"time"`
	Subject string    `json:"subject"`
	Status  string    `json:"status,omitempty"`
	Message string    `json:"message,omitempty"`
}

type PlanInput struct {
	Definition Definition
	ActorID    string
}

type DeployInput struct {
	Definition Definition
	ActorID    string
}
