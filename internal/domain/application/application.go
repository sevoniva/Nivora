package application

import "time"

type Application struct {
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

type Service struct {
	ID            string
	ApplicationID string
	Name          string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Repository struct {
	ID            string            `json:"id" yaml:"id"`
	ProjectID     string            `json:"projectId" yaml:"projectId"`
	Name          string            `json:"name" yaml:"name"`
	URL           string            `json:"url" yaml:"url"`
	Provider      string            `json:"provider" yaml:"provider"`
	DefaultBranch string            `json:"defaultBranch,omitempty" yaml:"defaultBranch,omitempty"`
	CredentialRef string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Labels        map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled       bool              `json:"enabled" yaml:"enabled"`
	CreatedAt     time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt" yaml:"updatedAt"`
}
