package deployment

import (
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	portgitops "github.com/sevoniva/nivora/internal/ports/gitops"
	"github.com/sevoniva/nivora/internal/ports/policy"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

type RunRecord struct {
	Definition      Definition                        `json:"definition,omitempty"`
	Release         release.Release                   `json:"release"`
	Artifacts       []release.ReleaseArtifact         `json:"artifacts,omitempty"`
	Environment     environment.Environment           `json:"environment"`
	Target          environment.ReleaseTarget         `json:"target"`
	Run             domaindeployment.DeploymentRun    `json:"run"`
	Steps           []domaindeployment.DeploymentStep `json:"steps,omitempty"`
	Plan            DeploymentPlan                    `json:"plan"`
	GitOpsPlan      GitOpsChangePlan                  `json:"gitopsPlan,omitempty"`
	GitOpsDiff      GitOpsDiff                        `json:"gitopsDiff,omitempty"`
	GitOpsCommit    portgitops.CommitResult           `json:"gitopsCommit,omitempty"`
	GitOpsPush      portgitops.CommitResult           `json:"gitopsPush,omitempty"`
	GitOpsRollback  portgitops.CommitResult           `json:"gitopsRollback,omitempty"`
	HostPlan        HostDeploymentPlan                `json:"hostPlan,omitempty"`
	HostDetails     []HostDeploymentRunDetail         `json:"hostDetails,omitempty"`
	ArgoCD          portargocd.ApplicationStatus      `json:"argocd,omitempty"`
	ArgoCDResources []portargocd.ResourceStatus       `json:"argocdResources,omitempty"`
	ArgoCDSync      portargocd.SyncResult             `json:"argocdSync,omitempty"`
	ArgoCDWatch     []portargocd.ApplicationStatus    `json:"argocdWatch,omitempty"`
	Inventory       ResourceInventory                 `json:"inventory,omitempty"`
	Health          HealthEvaluation                  `json:"health,omitempty"`
	Snapshot        ManifestSnapshot                  `json:"manifestSnapshot,omitempty"`
	Diff            DeploymentDiff                    `json:"diff,omitempty"`
	RollbackPlan    RollbackPlan                      `json:"rollbackPlan,omitempty"`
	DryRun          KubernetesDryRunResult            `json:"dryRun,omitempty"`
	Apply           KubernetesApplyResult             `json:"apply,omitempty"`
	Rollout         RolloutResult                     `json:"rollout,omitempty"`
	Rollback        *domaindeployment.RollbackRecord  `json:"rollback,omitempty"`
	Logs            []event.LogChunk                  `json:"logs,omitempty"`
	Events          []event.Event                     `json:"events,omitempty"`
	Audits          []audit.AuditLog                  `json:"audits,omitempty"`
	Policy          policy.Result                     `json:"policy"`
	Security        securityusecase.ScanRecord        `json:"security,omitempty"`
	Approval        domainapproval.ApprovalRequest    `json:"approval,omitempty"`
	ChangeWindow    domainapproval.ChangeWindowResult `json:"changeWindow,omitempty"`
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

type HostTarget struct {
	ID            string            `json:"id,omitempty"`
	Name          string            `json:"name"`
	Address       string            `json:"address,omitempty"`
	EnvironmentID string            `json:"environmentId,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

type HostGroup struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	EnvironmentID string            `json:"environmentId,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty"`
	Hosts         []HostTarget      `json:"hosts"`
	Labels        map[string]string `json:"labels,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type HostDeploymentPlan struct {
	DeploymentRunID string               `json:"deploymentRunId"`
	GroupName       string               `json:"groupName"`
	EnvironmentID   string               `json:"environmentId"`
	Artifact        string               `json:"artifact"`
	DeployPath      string               `json:"deployPath"`
	ServiceName     string               `json:"serviceName,omitempty"`
	HealthCheck     string               `json:"healthCheck,omitempty"`
	HealthChecks    []HostHealthCheck    `json:"healthChecks,omitempty"`
	RestartCommand  string               `json:"restartCommand,omitempty"`
	Strategy        string               `json:"strategy"`
	BatchSize       int                  `json:"batchSize"`
	PauseOnFailure  bool                 `json:"pauseOnFailure"`
	DryRun          bool                 `json:"dryRun"`
	Apply           bool                 `json:"apply"`
	Hosts           []HostDeploymentStep `json:"hosts"`
	Actions         []string             `json:"actions"`
	Warnings        []string             `json:"warnings,omitempty"`
	RollbackPlan    RollbackPlan         `json:"rollbackPlan"`
}

type HostDeploymentStep struct {
	HostID          string   `json:"hostId"`
	HostName        string   `json:"hostName"`
	Address         string   `json:"address,omitempty"`
	BatchIndex      int      `json:"batchIndex"`
	ReleaseDir      string   `json:"releaseDir"`
	CurrentSymlink  string   `json:"currentSymlink"`
	PreviousSymlink string   `json:"previousSymlink"`
	NextSymlink     string   `json:"nextSymlink"`
	TimeoutSeconds  int      `json:"timeoutSeconds,omitempty"`
	Actions         []string `json:"actions"`
}

type HostDeploymentRunDetail struct {
	HostID        string    `json:"hostId"`
	HostName      string    `json:"hostName"`
	Address       string    `json:"address,omitempty"`
	BatchIndex    int       `json:"batchIndex"`
	Status        string    `json:"status"`
	HealthStatus  string    `json:"healthStatus,omitempty"`
	RollbackReady bool      `json:"rollbackReady"`
	Message       string    `json:"message,omitempty"`
	StartedAt     time.Time `json:"startedAt,omitempty"`
	FinishedAt    time.Time `json:"finishedAt,omitempty"`
}

type HostHealthCheck struct {
	Type           string `json:"type" yaml:"type"`
	Target         string `json:"target,omitempty" yaml:"target,omitempty"`
	Command        string `json:"command,omitempty" yaml:"command,omitempty"`
	TimeoutSeconds int    `json:"timeoutSeconds,omitempty" yaml:"timeoutSeconds,omitempty"`
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
	CommitRevision        string                       `json:"commitRevision,omitempty"`
	RollbackRevision      string                       `json:"rollbackRevision,omitempty"`
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
	Group       string            `json:"group,omitempty"`
	Version     string            `json:"version,omitempty"`
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	SourceFile  string            `json:"sourceFile,omitempty"`
	Index       int               `json:"index"`
	DesiredHash string            `json:"desiredHash,omitempty"`
	LiveUID     string            `json:"liveUid,omitempty"`
	LiveVersion string            `json:"liveResourceVersion,omitempty"`
	Status      string            `json:"status,omitempty"`
	Health      ResourceHealth    `json:"health,omitempty"`
	CreatedAt   time.Time         `json:"createdAt,omitempty"`
	UpdatedAt   time.Time         `json:"updatedAt,omitempty"`
}

type ResourceHealth string

const (
	ResourceHealthUnknown     ResourceHealth = "Unknown"
	ResourceHealthProgressing ResourceHealth = "Progressing"
	ResourceHealthHealthy     ResourceHealth = "Healthy"
	ResourceHealthDegraded    ResourceHealth = "Degraded"
	ResourceHealthMissing     ResourceHealth = "Missing"
	ResourceHealthSuspended   ResourceHealth = "Suspended"
	ResourceHealthUnsupported ResourceHealth = "Unsupported"
)

type ResourceInventory struct {
	DeploymentRunID string                    `json:"deploymentRunId"`
	Desired         []ManifestResourceSummary `json:"desired"`
	Applied         []ManifestResourceSummary `json:"applied,omitempty"`
	Warnings        []string                  `json:"warnings,omitempty"`
	CreatedAt       time.Time                 `json:"createdAt"`
	UpdatedAt       time.Time                 `json:"updatedAt"`
}

type ResourceHealthSummary struct {
	Resource       ManifestResourceSummary `json:"resource"`
	DesiredExists  bool                    `json:"desiredExists"`
	LiveExists     bool                    `json:"liveExists"`
	Health         ResourceHealth          `json:"health"`
	DiffSummary    string                  `json:"diffSummary,omitempty"`
	LastObservedAt time.Time               `json:"lastObservedAt"`
	Warnings       []string                `json:"warnings,omitempty"`
}

type HealthEvaluation struct {
	DeploymentRunID  string                  `json:"deploymentRunId"`
	Status           ResourceHealth          `json:"status"`
	Resources        []ResourceHealthSummary `json:"resources"`
	ResourcesChecked int                     `json:"resourcesChecked"`
	Healthy          int                     `json:"healthy"`
	Degraded         int                     `json:"degraded"`
	Warnings         []string                `json:"warnings,omitempty"`
	EvaluatedAt      time.Time               `json:"evaluatedAt"`
}

type ManifestSnapshot struct {
	ID              string    `json:"id"`
	DeploymentRunID string    `json:"deploymentRunId"`
	ContentHash     string    `json:"contentHash"`
	DocumentCount   int       `json:"documentCount"`
	ResourceCount   int       `json:"resourceCount"`
	StorageRef      string    `json:"storageRef,omitempty"`
	InlineContent   string    `json:"inlineContent,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}

type DeploymentDiff struct {
	DeploymentRunID  string   `json:"deploymentRunId"`
	AddedResources   []string `json:"addedResources,omitempty"`
	RemovedResources []string `json:"removedResources,omitempty"`
	ChangedResources []string `json:"changedResources,omitempty"`
	Unchanged        []string `json:"unchangedResources,omitempty"`
	UnknownLiveState []string `json:"unknownLiveState,omitempty"`
	Summary          string   `json:"summary"`
	Warnings         []string `json:"warnings,omitempty"`
}

type RollbackPlan struct {
	DeploymentRunID    string                    `json:"deploymentRunId"`
	PreviousSnapshotID string                    `json:"previousSnapshotId,omitempty"`
	CurrentSnapshotID  string                    `json:"currentSnapshotId"`
	TargetType         string                    `json:"targetType"`
	TargetName         string                    `json:"targetName"`
	Resources          []ManifestResourceSummary `json:"resources"`
	Strategy           string                    `json:"strategy"`
	Executable         bool                      `json:"executable"`
	Warnings           []string                  `json:"warnings,omitempty"`
	CreatedAt          time.Time                 `json:"createdAt"`
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

type KubernetesRollbackResult struct {
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
	Definition    Definition
	ActorID       string
	CorrelationID string
	AllowApply    bool
	AllowSync     bool
	Confirm       bool
}

type RollbackInput struct {
	DeploymentRunID string
	ActorID         string
	Confirm         bool
}

type CreateRunResult struct {
	Record RunRecord
}
