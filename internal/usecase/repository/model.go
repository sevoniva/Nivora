package repository

import "time"

type Provider string

const (
	ProviderGenericGit       Provider = "generic_git"
	ProviderGitHub           Provider = "github"
	ProviderGitLab           Provider = "gitlab"
	ProviderGitea            Provider = "gitea"
	ProviderLocal            Provider = "local"
	ProviderArchive          Provider = "archive"
	RepositoryStatusActive            = "active"
	RepositoryStatusDisabled          = "disabled"
)

type Repository struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Provider      Provider          `json:"provider"`
	URL           string            `json:"url"`
	WebURL        string            `json:"webUrl,omitempty"`
	DefaultBranch string            `json:"defaultBranch,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty"`
	ProjectID     string            `json:"projectId,omitempty"`
	EnvironmentID string            `json:"environmentId,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Status        string            `json:"status"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
}

type RepositoryFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Hash string `json:"hash,omitempty"`
}

type RepositorySnapshot struct {
	ID                      string            `json:"id"`
	RepositoryID            string            `json:"repositoryId"`
	Ref                     string            `json:"ref"`
	CommitSHA               string            `json:"commitSha,omitempty"`
	Branch                  string            `json:"branch,omitempty"`
	Tag                     string            `json:"tag,omitempty"`
	TreeHash                string            `json:"treeHash"`
	Files                   []RepositoryFile  `json:"files,omitempty"`
	DetectedLanguages       []string          `json:"detectedLanguages,omitempty"`
	DetectedFrameworks      []string          `json:"detectedFrameworks,omitempty"`
	DetectedBuildTools      []string          `json:"detectedBuildTools,omitempty"`
	DetectedPackageManagers []string          `json:"detectedPackageManagers,omitempty"`
	DetectedDeploymentFiles []string          `json:"detectedDeploymentFiles,omitempty"`
	DetectedWorkflowFiles   []string          `json:"detectedWorkflowFiles,omitempty"`
	DetectedSecurityFiles   []string          `json:"detectedSecurityFiles,omitempty"`
	Warnings                []string          `json:"warnings,omitempty"`
	Metadata                map[string]string `json:"metadata,omitempty"`
	CreatedAt               time.Time         `json:"createdAt"`
}

type CommandCandidate struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Source  string `json:"source"`
}

type RepositoryIntelligence struct {
	RepositoryID                   string             `json:"repositoryId"`
	SnapshotID                     string             `json:"snapshotId,omitempty"`
	LanguageSummary                []string           `json:"languageSummary,omitempty"`
	FrameworkSummary               []string           `json:"frameworkSummary,omitempty"`
	BuildCommandCandidates         []CommandCandidate `json:"buildCommandCandidates,omitempty"`
	TestCommandCandidates          []CommandCandidate `json:"testCommandCandidates,omitempty"`
	PackageCommandCandidates       []CommandCandidate `json:"packageCommandCandidates,omitempty"`
	DeploymentTargetCandidates     []string           `json:"deploymentTargetCandidates,omitempty"`
	SecurityScanCandidates         []string           `json:"securityScanCandidates,omitempty"`
	RecommendedNivoraWorkflowDraft string             `json:"recommendedNivoraWorkflowDraft,omitempty"`
	Warnings                       []string           `json:"warnings,omitempty"`
	CreatedAt                      time.Time          `json:"createdAt"`
}

type BuildPlan struct {
	RepositoryID string             `json:"repositoryId"`
	SnapshotID   string             `json:"snapshotId,omitempty"`
	Commands     []CommandCandidate `json:"commands,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
	CreatedAt    time.Time          `json:"createdAt"`
}

type TestPlan struct {
	RepositoryID string             `json:"repositoryId"`
	SnapshotID   string             `json:"snapshotId,omitempty"`
	Commands     []CommandCandidate `json:"commands,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
	CreatedAt    time.Time          `json:"createdAt"`
}

type PackagePlan struct {
	RepositoryID string             `json:"repositoryId"`
	SnapshotID   string             `json:"snapshotId,omitempty"`
	Commands     []CommandCandidate `json:"commands,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
	CreatedAt    time.Time          `json:"createdAt"`
}

type SecurityScanPlan struct {
	RepositoryID string    `json:"repositoryId"`
	SnapshotID   string    `json:"snapshotId,omitempty"`
	Candidates   []string  `json:"candidates,omitempty"`
	Warnings     []string  `json:"warnings,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ReleaseCandidatePlan struct {
	RepositoryID       string    `json:"repositoryId"`
	SnapshotID         string    `json:"snapshotId,omitempty"`
	Eligible           bool      `json:"eligible"`
	ArtifactCandidates []string  `json:"artifactCandidates,omitempty"`
	RequiredChecks     []string  `json:"requiredChecks,omitempty"`
	Warnings           []string  `json:"warnings,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
}

type DevOpsPlan struct {
	RepositoryID      string               `json:"repositoryId"`
	SnapshotID        string               `json:"snapshotId,omitempty"`
	Build             BuildPlan            `json:"build"`
	Test              TestPlan             `json:"test"`
	Package           PackagePlan          `json:"package"`
	Security          SecurityScanPlan     `json:"security"`
	ReleaseCandidate  ReleaseCandidatePlan `json:"releaseCandidate"`
	SecurityScans     []string             `json:"securityScans,omitempty"`
	DeploymentTargets []string             `json:"deploymentTargets,omitempty"`
	ReleaseReady      bool                 `json:"releaseReady"`
	Warnings          []string             `json:"warnings,omitempty"`
	Metadata          map[string]string    `json:"metadata,omitempty"`
	CreatedAt         time.Time            `json:"createdAt"`
}

type SnapshotInput struct {
	Repository Repository
	Ref        string
	LocalPath  string
}

type AnalyzeInput struct {
	Snapshot RepositorySnapshot
}
