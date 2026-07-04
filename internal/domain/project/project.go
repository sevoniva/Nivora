package project

import "time"

type Project struct {
	ID          string            `json:"id" yaml:"id"`
	OrgID       string            `json:"orgId" yaml:"orgId"`
	Name        string            `json:"name" yaml:"name"`
	Slug        string            `json:"slug,omitempty" yaml:"slug,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	CreatedAt   time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt" yaml:"updatedAt"`
}
