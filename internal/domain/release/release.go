package release

import "time"

type ReleaseStatus string

const (
	ReleaseStatusDraft           ReleaseStatus = "Draft"
	ReleaseStatusReady           ReleaseStatus = "Ready"
	ReleaseStatusPlanning        ReleaseStatus = "Planning"
	ReleaseStatusWaitingApproval ReleaseStatus = "WaitingApproval"
	ReleaseStatusDeploying       ReleaseStatus = "Deploying"
	ReleaseStatusSucceeded       ReleaseStatus = "Succeeded"
	ReleaseStatusFailed          ReleaseStatus = "Failed"
	ReleaseStatusCanceled        ReleaseStatus = "Canceled"
)

func ValidStatus(status ReleaseStatus) bool {
	switch status {
	case ReleaseStatusDraft, ReleaseStatusReady, ReleaseStatusPlanning, ReleaseStatusWaitingApproval,
		ReleaseStatusDeploying, ReleaseStatusSucceeded, ReleaseStatusFailed, ReleaseStatusCanceled:
		return true
	default:
		return false
	}
}

func TerminalStatus(status ReleaseStatus) bool {
	switch status {
	case ReleaseStatusSucceeded, ReleaseStatusFailed, ReleaseStatusCanceled:
		return true
	default:
		return false
	}
}

type Release struct {
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	Version             string            `json:"version"`
	ApplicationID       string            `json:"applicationId,omitempty"`
	EnvironmentID       string            `json:"environmentId,omitempty"`
	SourcePipelineRunID string            `json:"sourcePipelineRunId,omitempty"`
	Commit              string            `json:"commit,omitempty"`
	Status              string            `json:"status,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
	CreatedAt           time.Time         `json:"createdAt"`
	UpdatedAt           time.Time         `json:"updatedAt"`
}

type ReleaseArtifact struct {
	ID              string            `json:"id"`
	ReleaseID       string            `json:"releaseId"`
	ArtifactID      string            `json:"artifactId,omitempty"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	Role            string            `json:"role,omitempty"`
	Required        bool              `json:"required"`
	Reference       string            `json:"reference"`
	Digest          string            `json:"digest,omitempty"`
	DigestReference string            `json:"digestReference,omitempty"`
	MediaType       string            `json:"mediaType,omitempty"`
	SizeBytes       int64             `json:"sizeBytes,omitempty"`
	ManifestSchema  string            `json:"manifestSchema,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}
