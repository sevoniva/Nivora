package runner

import "time"

type Runner struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	GroupID         string            `json:"groupId,omitempty"`
	Status          string            `json:"status"`
	Labels          map[string]string `json:"labels,omitempty"`
	Executors       []string          `json:"executors,omitempty"`
	LastHeartbeatAt *time.Time        `json:"lastHeartbeatAt,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

type RunnerGroup struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
