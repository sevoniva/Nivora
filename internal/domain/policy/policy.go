package policy

import "time"

type Approval struct {
	ID              string
	DeploymentRunID string
	Status          string
	RequestedBy     string
	ApprovedBy      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Policy struct {
	ID        string
	ProjectID string
	Name      string
	Mode      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PolicyResult struct {
	ID        string
	PolicyID  string
	Subject   string
	Passed    bool
	Message   string
	CreatedAt time.Time
	UpdatedAt time.Time
}
