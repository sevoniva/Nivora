package plugin

import "time"

type Type string

const (
	TypeSCM          Type = "scm"
	TypeArtifact     Type = "artifact"
	TypeCloud        Type = "cloud"
	TypeExecutor     Type = "executor"
	TypeSecret       Type = "secret"
	TypeNotification Type = "notification"
	TypePolicy       Type = "policy"
	TypeScanner      Type = "scanner"
	TypeGitOps       Type = "gitops"
)

type Status string

const (
	StatusBuiltIn     Status = "builtin"
	StatusConfigured  Status = "configured"
	StatusUnavailable Status = "unavailable"
	StatusExternal    Status = "external"
)

type Capability struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Inputs      []string          `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Outputs     []string          `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Manifest struct {
	Name         string            `json:"name" yaml:"name"`
	Type         Type              `json:"type" yaml:"type"`
	Version      string            `json:"version" yaml:"version"`
	Protocol     string            `json:"protocol" yaml:"protocol"`
	Endpoint     string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Capabilities []Capability      `json:"capabilities" yaml:"capabilities"`
	Status       Status            `json:"status" yaml:"status"`
	Metadata     map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt    time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt    time.Time         `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}
