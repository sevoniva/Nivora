package environment

import "time"

type Environment struct {
	ID        string
	ProjectID string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ReleaseTarget struct {
	ID            string
	EnvironmentID string
	Name          string
	TargetType    string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type EnvironmentLock struct {
	ID            string
	EnvironmentID string
	Reason        string
	LockedBy      string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
