package deployment

import (
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	"github.com/sevoniva/nivora/internal/ports/policy"
)

type RunRecord struct {
	Definition  Definition                        `json:"definition,omitempty"`
	Release     release.Release                   `json:"release"`
	Artifacts   []release.ReleaseArtifact         `json:"artifacts,omitempty"`
	Environment environment.Environment           `json:"environment"`
	Target      environment.ReleaseTarget         `json:"target"`
	Run         domaindeployment.DeploymentRun    `json:"run"`
	Steps       []domaindeployment.DeploymentStep `json:"steps,omitempty"`
	Plan        DeploymentPlan                    `json:"plan"`
	Logs        []event.LogChunk                  `json:"logs,omitempty"`
	Events      []event.Event                     `json:"events,omitempty"`
	Audits      []audit.AuditLog                  `json:"audits,omitempty"`
	Policy      policy.Result                     `json:"policy"`
}

type DeploymentPlan struct {
	DeploymentRunID string                    `json:"deploymentRunId"`
	TargetType      string                    `json:"targetType"`
	Namespace       string                    `json:"namespace,omitempty"`
	ManifestCount   int                       `json:"manifestCount"`
	Resources       []ManifestResourceSummary `json:"resources"`
	Artifacts       []string                  `json:"artifacts,omitempty"`
	DryRun          bool                      `json:"dryRun"`
	Apply           bool                      `json:"apply"`
	Actions         []string                  `json:"actions"`
	Warnings        []string                  `json:"warnings,omitempty"`
	DiffSummary     string                    `json:"diffSummary"`
}

type ManifestDocument struct {
	SourceFile string                  `json:"sourceFile"`
	Index      int                     `json:"documentIndex"`
	Content    string                  `json:"content"`
	Resource   ManifestResourceSummary `json:"resource"`
}

type ManifestResourceSummary struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	SourceFile string `json:"sourceFile,omitempty"`
	Index      int    `json:"index"`
}

type TimelineEntry struct {
	Type    string    `json:"type"`
	Time    time.Time `json:"time"`
	Subject string    `json:"subject"`
	Status  string    `json:"status,omitempty"`
	Message string    `json:"message,omitempty"`
}

type CreateRunInput struct {
	Definition Definition
	ActorID    string
}

type CreateRunResult struct {
	Record RunRecord
}
