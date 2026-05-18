package argocd

import "context"

type CredentialRef struct {
	Name string `json:"name,omitempty"`
}

type ApplicationStatus struct {
	ApplicationName string           `json:"applicationName"`
	Project         string           `json:"project,omitempty"`
	Namespace       string           `json:"namespace,omitempty"`
	SyncStatus      string           `json:"syncStatus"`
	HealthStatus    string           `json:"healthStatus"`
	Revision        string           `json:"revision,omitempty"`
	TargetRevision  string           `json:"targetRevision,omitempty"`
	Resources       []ResourceStatus `json:"resources,omitempty"`
	Conditions      []string         `json:"conditions,omitempty"`
	ObservedAt      string           `json:"observedAt,omitempty"`
	RawSummary      string           `json:"rawSummary,omitempty"`
	Message         string           `json:"message,omitempty"`
	Warnings        []string         `json:"warnings,omitempty"`
}

type ResourceStatus struct {
	Group      string `json:"group,omitempty"`
	Version    string `json:"version,omitempty"`
	Kind       string `json:"kind"`
	Namespace  string `json:"namespace,omitempty"`
	Name       string `json:"name"`
	Status     string `json:"status,omitempty"`
	Health     string `json:"health,omitempty"`
	SyncStatus string `json:"syncStatus,omitempty"`
	Message    string `json:"message,omitempty"`
}

type SyncRequest struct {
	ApplicationName string `json:"applicationName"`
	Revision        string `json:"revision,omitempty"`
	Prune           bool   `json:"prune"`
	DryRun          bool   `json:"dryRun,omitempty"`
	Force           bool   `json:"force"`
	Wait            bool   `json:"wait"`
	TimeoutSeconds  int    `json:"timeoutSeconds,omitempty"`
	AllowSync       bool   `json:"allowSync"`
	Confirmed       bool   `json:"confirmed"`
	Confirmation    string `json:"confirmation,omitempty"`
}

type SyncResult struct {
	ApplicationName string            `json:"applicationName"`
	Requested       bool              `json:"requested"`
	Started         bool              `json:"started"`
	Completed       bool              `json:"completed"`
	SyncStatus      string            `json:"syncStatus,omitempty"`
	HealthStatus    string            `json:"healthStatus,omitempty"`
	Revision        string            `json:"revision,omitempty"`
	Message         string            `json:"message"`
	Warnings        []string          `json:"warnings,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type Provider interface {
	ValidateCredential(ctx context.Context, credential CredentialRef) error
	GetApplicationStatus(ctx context.Context, applicationName string) (ApplicationStatus, error)
	GetApplicationResources(ctx context.Context, applicationName string) ([]ResourceStatus, error)
	GetApplicationHistory(ctx context.Context, applicationName string) ([]ApplicationStatus, error)
	SyncApplication(ctx context.Context, request SyncRequest) (SyncResult, error)
	WatchApplicationStatus(ctx context.Context, applicationName string, timeoutSeconds int) ([]ApplicationStatus, error)
}
