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
	Username    string
	Email       string
	DisplayName string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
