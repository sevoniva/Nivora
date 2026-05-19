package auth

import "time"

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"

	RoleOwner      = "owner"
	RoleAdmin      = "admin"
	RoleMaintainer = "maintainer"
	RoleDeveloper  = "developer"
	RoleViewer     = "viewer"
	RoleAuditor    = "auditor"

	PermissionProjectRead       = "project.read"
	PermissionProjectWrite      = "project.write"
	PermissionApplicationRead   = "application.read"
	PermissionApplicationWrite  = "application.write"
	PermissionEnvironmentRead   = "environment.read"
	PermissionEnvironmentWrite  = "environment.write"
	PermissionPipelineRun       = "pipeline.run"
	PermissionDeploymentCreate  = "deployment.create"
	PermissionDeploymentApprove = "deployment.approve"
	PermissionDeploymentCancel  = "deployment.cancel"
	PermissionReleaseCreate     = "release.create"
	PermissionCredentialManage  = "credential.manage"
	PermissionRunnerManage      = "runner.manage"
	PermissionPolicyManage      = "policy.manage"
	PermissionAuditRead         = "audit.read"
)

type User struct {
	ID          string    `json:"id" yaml:"id"`
	Username    string    `json:"username" yaml:"username"`
	Email       string    `json:"email,omitempty" yaml:"email,omitempty"`
	DisplayName string    `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	Status      string    `json:"status" yaml:"status"`
	CreatedAt   time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" yaml:"updatedAt"`
}

type Role struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	Permissions []Permission `json:"permissions" yaml:"permissions"`
}

type Permission struct {
	Action      string `json:"action" yaml:"action"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type Membership struct {
	ID        string    `json:"id" yaml:"id"`
	ScopeType string    `json:"scopeType" yaml:"scopeType"`
	ScopeID   string    `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	UserID    string    `json:"userId" yaml:"userId"`
	Role      string    `json:"role" yaml:"role"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" yaml:"updatedAt"`
}

type ServiceAccount struct {
	ID        string    `json:"id" yaml:"id"`
	Name      string    `json:"name" yaml:"name"`
	ScopeType string    `json:"scopeType" yaml:"scopeType"`
	ScopeID   string    `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Role      string    `json:"role" yaml:"role"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" yaml:"updatedAt"`
}

type TokenMetadata struct {
	ID        string    `json:"id" yaml:"id"`
	SubjectID string    `json:"subjectId" yaml:"subjectId"`
	Name      string    `json:"name,omitempty" yaml:"name,omitempty"`
	IssuedAt  time.Time `json:"issuedAt" yaml:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt,omitempty" yaml:"expiresAt,omitempty"`
}

type Subject struct {
	ID          string   `json:"id" yaml:"id"`
	Username    string   `json:"username" yaml:"username"`
	DisplayName string   `json:"displayName,omitempty" yaml:"displayName,omitempty"`
	Roles       []string `json:"roles" yaml:"roles"`
	AuthMode    string   `json:"authMode" yaml:"authMode"`
}

type Resource struct {
	Type      string `json:"type" yaml:"type"`
	ID        string `json:"id,omitempty" yaml:"id,omitempty"`
	ScopeType string `json:"scopeType,omitempty" yaml:"scopeType,omitempty"`
	ScopeID   string `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
}

type Decision struct {
	Allowed bool     `json:"allowed" yaml:"allowed"`
	Reason  string   `json:"reason" yaml:"reason"`
	Action  string   `json:"action" yaml:"action"`
	Roles   []string `json:"roles,omitempty" yaml:"roles,omitempty"`
}
