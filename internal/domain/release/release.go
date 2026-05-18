package release

import "time"

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
	ID         string            `json:"id"`
	ReleaseID  string            `json:"releaseId"`
	ArtifactID string            `json:"artifactId,omitempty"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Role       string            `json:"role,omitempty"`
	Required   bool              `json:"required"`
	Reference  string            `json:"reference"`
	Digest     string            `json:"digest,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}
