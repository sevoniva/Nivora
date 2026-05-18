package runner

import "time"

type Runner struct {
	ID              string
	Name            string
	GroupID         string
	Status          string
	Labels          map[string]string
	Executors       []string
	LastHeartbeatAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type RunnerGroup struct {
	ID        string
	ProjectID string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
