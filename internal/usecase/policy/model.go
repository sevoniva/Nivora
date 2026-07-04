package policy

import "errors"

var (
	ErrInvalid       = errors.New("policy input is invalid")
	ErrNotFound      = errors.New("policy not found")
	ErrAlreadyExists = errors.New("policy already exists")
	ErrDisabled      = errors.New("policy is disabled")
)

type CreateInput struct {
	ID                 string            `json:"id,omitempty" yaml:"id,omitempty"`
	ProjectID          string            `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID      string            `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	Name               string            `json:"name" yaml:"name"`
	Description        string            `json:"description,omitempty" yaml:"description,omitempty"`
	Type               string            `json:"type,omitempty" yaml:"type,omitempty"`
	Mode               string            `json:"mode,omitempty" yaml:"mode,omitempty"`
	CriticalDeny       int               `json:"criticalDenyThreshold,omitempty" yaml:"criticalDenyThreshold,omitempty"`
	HighWarn           int               `json:"highWarnThreshold,omitempty" yaml:"highWarnThreshold,omitempty"`
	RequireDigest      bool              `json:"requireDigest,omitempty" yaml:"requireDigest,omitempty"`
	ApprovalOnCritical bool              `json:"approvalOnCritical,omitempty" yaml:"approvalOnCritical,omitempty"`
	Labels             map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled            *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type UpdateInput struct {
	ProjectID          *string           `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID      *string           `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	Name               *string           `json:"name,omitempty" yaml:"name,omitempty"`
	Description        *string           `json:"description,omitempty" yaml:"description,omitempty"`
	Type               *string           `json:"type,omitempty" yaml:"type,omitempty"`
	Mode               *string           `json:"mode,omitempty" yaml:"mode,omitempty"`
	CriticalDeny       *int              `json:"criticalDenyThreshold,omitempty" yaml:"criticalDenyThreshold,omitempty"`
	HighWarn           *int              `json:"highWarnThreshold,omitempty" yaml:"highWarnThreshold,omitempty"`
	RequireDigest      *bool             `json:"requireDigest,omitempty" yaml:"requireDigest,omitempty"`
	ApprovalOnCritical *bool             `json:"approvalOnCritical,omitempty" yaml:"approvalOnCritical,omitempty"`
	Labels             map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled            *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type AttachInput struct {
	ID        string            `json:"id,omitempty" yaml:"id,omitempty"`
	ScopeType string            `json:"scopeType" yaml:"scopeType"`
	ScopeID   string            `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled   *bool             `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type AttachmentListInput struct {
	PolicyID  string
	ScopeType string
	ScopeID   string
	Enabled   *bool
}

type ResolveInput struct {
	ProjectID     string
	EnvironmentID string
}
