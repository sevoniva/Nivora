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
	ID                 string            `json:"id" yaml:"id"`
	ProjectID          string            `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	EnvironmentID      string            `json:"environmentId,omitempty" yaml:"environmentId,omitempty"`
	Name               string            `json:"name" yaml:"name"`
	Description        string            `json:"description,omitempty" yaml:"description,omitempty"`
	Type               string            `json:"type" yaml:"type"`
	Mode               string            `json:"mode" yaml:"mode"`
	CriticalDeny       int               `json:"criticalDenyThreshold,omitempty" yaml:"criticalDenyThreshold,omitempty"`
	HighWarn           int               `json:"highWarnThreshold,omitempty" yaml:"highWarnThreshold,omitempty"`
	RequireDigest      bool              `json:"requireDigest,omitempty" yaml:"requireDigest,omitempty"`
	ApprovalOnCritical bool              `json:"approvalOnCritical,omitempty" yaml:"approvalOnCritical,omitempty"`
	Labels             map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled            bool              `json:"enabled" yaml:"enabled"`
	CreatedAt          time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

type PolicyAttachment struct {
	ID        string            `json:"id" yaml:"id"`
	PolicyID  string            `json:"policyId" yaml:"policyId"`
	ScopeType string            `json:"scopeType" yaml:"scopeType"`
	ScopeID   string            `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Enabled   bool              `json:"enabled" yaml:"enabled"`
	CreatedAt time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt" yaml:"updatedAt"`
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
