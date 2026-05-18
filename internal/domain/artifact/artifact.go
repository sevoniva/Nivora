package artifact

import "time"

type ArtifactRegistry struct {
	ID        string
	ProjectID string
	Name      string
	Type      string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}
