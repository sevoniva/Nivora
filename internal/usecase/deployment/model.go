package deployment

import (
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	portgitops "github.com/sevoniva/nivora/internal/ports/gitops"
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
	GitOpsPlan  GitOpsChangePlan                  `json:"gitopsPlan,omitempty"`
	GitOpsDiff  GitOpsDiff                        `json:"gitopsDiff,omitempty"`
	ArgoCD      portargocd.ApplicationStatus      `json:"argocd,omitempty"`
	DryRun      KubernetesDryRunResult            `json:"dryRun,omitempty"`
	Apply       KubernetesApplyResult             `json:"apply,omitempty"`
	Rollout     RolloutResult                     `json:"rollout,omitempty"`
	Rollback    *domaindeployment.RollbackRecord  `json:"rollback,omitempty"`
	Logs        []event.LogChunk                  `json:"logs,omitempty"`
	Events      []event.Event                     `json:"events,omitempty"`
	Audits      []audit.AuditLog                  `json:"audits,omitempty"`
	Policy      policy.Result                     `json:"policy"`
}

type DeploymentPlan struct {
	DeploymentRunID string                      `json:"deploymentRunId"`
	TargetType      string                      `json:"targetType"`
	TargetContext   string                      `json:"targetContext,omitempty"`
	Namespace       string                      `json:"namespace,omitempty"`
	ManifestCount   int                         `json:"manifestCount"`
	Resources       []ManifestResourceSummary   `json:"resources"`
	Artifacts       []string                    `json:"artifacts,omitempty"`
	ArtifactDetails []domainartifact.Inspection `json:"artifactDetails,omitempty"`
	ManifestImages  []ManifestImage             `json:"manifestImages,omitempty"`
	DryRun          bool                        `json:"dryRun"`
	Apply           bool                        `json:"apply"`
	Wait            bool                        `json:"wait"`
	TimeoutSeconds  int                         `json:"timeoutSeconds,omitempty"`
	Actions         []string                    `json:"actions"`
	Warnings        []string                    `json:"warnings,omitempty"`
	DiffSummary     string                      `json:"diffSummary"`
}

type ManifestImage struct {
	ResourceKind string `json:"resourceKind"`
	ResourceName string `json:"resourceName"`
	Container    string `json:"container"`
	Image        string `json:"image"`
}

type GitOpsChangePlan struct {
	DeploymentRunID       string                       `json:"deploymentRunId"`
	ApplicationName       string                       `json:"applicationName"`
	RepoURL               string                       `json:"repoURL"`
	Path                  string                       `json:"path"`
	Revision              string                       `json:"revision,omitempty"`
	Files                 []string                     `json:"files"`
	FileChanges           []portgitops.FileChange      `json:"fileChanges,omitempty"`
	ArtifactChanges       []string                     `json:"artifactChanges,omitempty"`
	ManifestValueChanges  []string                     `json:"manifestValueChanges,omitempty"`
	CommitMessageProposal string                       `json:"commitMessageProposal,omitempty"`
	DryRun                bool                         `json:"dryRun"`
	Warnings              []string                     `json:"warnings,omitempty"`
	SyncRequested         bool                         `json:"syncRequested"`
	Status                portargocd.ApplicationStatus `json:"status,omitempty"`
}

type GitOpsDiff struct {
	Summary string                  `json:"summary"`
	Files   []portgitops.FileChange `json:"files"`
}

type ManifestDocument struct {
	SourceFile string                  `json:"sourceFile"`
	Index      int                     `json:"documentIndex"`
	Content    string                  `json:"content"`
	Resource   ManifestResourceSummary `json:"resource"`
}

type ManifestResourceSummary struct {
	APIVersion  string            `json:"apiVersion"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	SourceFile  string            `json:"sourceFile,omitempty"`
	Index       int               `json:"index"`
}

type ManifestRequest struct {
	Plan           DeploymentPlan     `json:"plan"`
	Documents      []ManifestDocument `json:"documents"`
	TimeoutSeconds int                `json:"timeoutSeconds,omitempty"`
}

type KubernetesDryRunResult struct {
	Mode      string                    `json:"mode,omitempty"`
	Message   string                    `json:"message,omitempty"`
	Resources []ManifestResourceSummary `json:"resources,omitempty"`
	Warnings  []string                  `json:"warnings,omitempty"`
	Stdout    string                    `json:"stdout,omitempty"`
	Stderr    string                    `json:"stderr,omitempty"`
}

type KubernetesApplyResult struct {
	Mode      string                    `json:"mode,omitempty"`
	Message   string                    `json:"message,omitempty"`
	Resources []ManifestResourceSummary `json:"resources,omitempty"`
	Warnings  []string                  `json:"warnings,omitempty"`
	Stdout    string                    `json:"stdout,omitempty"`
	Stderr    string                    `json:"stderr,omitempty"`
}

type RolloutResult struct {
	Mode      string                    `json:"mode,omitempty"`
	Message   string                    `json:"message,omitempty"`
	Resources []ManifestResourceSummary `json:"resources,omitempty"`
	Warnings  []string                  `json:"warnings,omitempty"`
	Stdout    string                    `json:"stdout,omitempty"`
	Stderr    string                    `json:"stderr,omitempty"`
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
	AllowApply bool
	AllowSync  bool
	Confirm    bool
}

type CreateRunResult struct {
	Record RunRecord
}
