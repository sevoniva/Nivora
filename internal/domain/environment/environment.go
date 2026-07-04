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
	ID                    string            `json:"id" yaml:"id"`
	ProjectID             string            `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID         string            `json:"environmentId" yaml:"environmentId"`
	Name                  string            `json:"name" yaml:"name"`
	TargetType            string            `json:"targetType" yaml:"targetType"`
	Context               string            `json:"context,omitempty" yaml:"context,omitempty"`
	Namespace             string            `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	ConfigRef             string            `json:"configRef,omitempty" yaml:"configRef,omitempty"`
	CredentialRef         string            `json:"credentialRef,omitempty" yaml:"credentialRef,omitempty"`
	Labels                map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	AllowApply            bool              `json:"allowApply" yaml:"allowApply"`
	AllowSync             bool              `json:"allowSync" yaml:"allowSync"`
	AllowRemoteHostDeploy bool              `json:"allowRemoteHostDeploy" yaml:"allowRemoteHostDeploy"`
	Enabled               bool              `json:"enabled" yaml:"enabled"`
	CreatedAt             time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt             time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

type EnvironmentLock struct {
	ID            string    `json:"id"`
	EnvironmentID string    `json:"environmentId"`
	Reason        string    `json:"reason"`
	LockedBy      string    `json:"lockedBy"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
