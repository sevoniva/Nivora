package plugin

import "time"

const (
	ManifestAPIVersion = "nivora.io/plugin/v1alpha1"
	PluginAPIVersion   = "v1alpha1"
)

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

type Protocol string

const (
	ProtocolBuiltIn Protocol = "builtin"
	ProtocolHTTP    Protocol = "http"
	ProtocolGRPC    Protocol = "grpc"
)

type Capability struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Inputs      []string          `json:"inputs,omitempty" yaml:"inputs,omitempty"`
	Outputs     []string          `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type Compatibility struct {
	PluginAPIVersion string   `json:"pluginApiVersion" yaml:"pluginApiVersion"`
	NivoraMinVersion string   `json:"nivoraMinVersion,omitempty" yaml:"nivoraMinVersion,omitempty"`
	NivoraMaxVersion string   `json:"nivoraMaxVersion,omitempty" yaml:"nivoraMaxVersion,omitempty"`
	Protocols        []string `json:"protocols,omitempty" yaml:"protocols,omitempty"`
}

type Lifecycle struct {
	Health         bool `json:"health" yaml:"health"`
	Capabilities   bool `json:"capabilities" yaml:"capabilities"`
	ValidateConfig bool `json:"validateConfig" yaml:"validateConfig"`
	Execute        bool `json:"execute" yaml:"execute"`
}

type Manifest struct {
	APIVersion    string            `json:"apiVersion" yaml:"apiVersion"`
	Name          string            `json:"name" yaml:"name"`
	Type          Type              `json:"type" yaml:"type"`
	Version       string            `json:"version" yaml:"version"`
	Protocol      string            `json:"protocol" yaml:"protocol"`
	Endpoint      string            `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Capabilities  []Capability      `json:"capabilities" yaml:"capabilities"`
	Compatibility Compatibility     `json:"compatibility" yaml:"compatibility"`
	Lifecycle     Lifecycle         `json:"lifecycle" yaml:"lifecycle"`
	Status        Status            `json:"status" yaml:"status"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt     time.Time         `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
}
