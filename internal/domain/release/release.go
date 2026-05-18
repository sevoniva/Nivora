package release

import "time"

type Release struct {
	ID            string    `json:"id"`
	ApplicationID string    `json:"applicationId,omitempty"`
	Version       string    `json:"version"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type ReleaseArtifact struct {
	ID        string    `json:"id"`
	ReleaseID string    `json:"releaseId"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Reference string    `json:"reference"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
