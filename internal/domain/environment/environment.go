package environment

import "time"

type Environment struct {
	ID          string            `json:"id" yaml:"id"`
	ProjectID   string            `json:"projectId" yaml:"projectId"`
	Name        string            `json:"name" yaml:"name"`
	Slug        string            `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	CreatedAt   time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

type ReleaseTarget struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	Name          string    `json:"name"`
	TargetType    string    `json:"targetType"`
	Context       string    `json:"context,omitempty"`
	Namespace     string    `json:"namespace,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type EnvironmentLock struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	Reason        string    `json:"reason"`
	LockedBy      string    `json:"lockedBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
