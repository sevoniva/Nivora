package project

import "time"

type Project struct {
	ID        string
	OrgID     string
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
