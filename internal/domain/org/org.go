package org

import "time"

type Org struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID          string
	OrgID       string
	Email       string
	DisplayName string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
