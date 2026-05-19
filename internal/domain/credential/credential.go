package credential

import "time"

const (
	ScopeOrg         = "org"
	ScopeProject     = "project"
	ScopeEnvironment = "environment"
	ScopeRunner      = "runner"
	ScopeGlobal      = "global"

	TypeToken            = "token"
	TypeUsernamePassword = "username_password"
	TypeSSHKey           = "ssh_key"
	TypeKubeconfig       = "kubeconfig"
	TypeRegistry         = "registry"
	TypeCloud            = "cloud"
	TypeArgoCD           = "argocd"
	TypeWebhook          = "webhook"
	TypeGeneric          = "generic"

	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusUnknown  = "unknown"
)

type Credential struct {
	ID        string            `json:"id" yaml:"id"`
	Name      string            `json:"name" yaml:"name"`
	Type      string            `json:"type" yaml:"type"`
	ScopeType string            `json:"scopeType" yaml:"scopeType"`
	ScopeID   string            `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	SecretRef SecretRef         `json:"secretRef" yaml:"secretRef"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Status    string            `json:"status" yaml:"status"`
	CreatedAt time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

type SecretRef struct {
	ID        string            `json:"id" yaml:"id"`
	Name      string            `json:"name" yaml:"name"`
	ScopeType string            `json:"scopeType" yaml:"scopeType"`
	ScopeID   string            `json:"scopeId,omitempty" yaml:"scopeId,omitempty"`
	Provider  string            `json:"provider" yaml:"provider"`
	Key       string            `json:"key" yaml:"key"`
	Version   string            `json:"version,omitempty" yaml:"version,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt time.Time         `json:"createdAt" yaml:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt" yaml:"updatedAt"`
}

type SecretUsage struct {
	ID          string    `json:"id" yaml:"id"`
	SecretRef   SecretRef `json:"secretRef" yaml:"secretRef"`
	UsedBy      string    `json:"usedBy" yaml:"usedBy"`
	Purpose     string    `json:"purpose" yaml:"purpose"`
	SubjectType string    `json:"subjectType" yaml:"subjectType"`
	SubjectID   string    `json:"subjectId,omitempty" yaml:"subjectId,omitempty"`
	CreatedAt   time.Time `json:"createdAt" yaml:"createdAt"`
}
