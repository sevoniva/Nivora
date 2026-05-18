package release

import "time"

type Release struct {
	ID            string
	ApplicationID string
	Version       string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
