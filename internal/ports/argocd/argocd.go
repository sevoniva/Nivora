package argocd

import "context"

type CredentialRef struct {
	Name string `json:"name,omitempty"`
}

type ApplicationStatus struct {
	ApplicationName string   `json:"applicationName"`
	Project         string   `json:"project,omitempty"`
	SyncStatus      string   `json:"syncStatus"`
	HealthStatus    string   `json:"healthStatus"`
	Revision        string   `json:"revision,omitempty"`
	Message         string   `json:"message,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
}

type SyncRequest struct {
	ApplicationName string `json:"applicationName"`
	Revision        string `json:"revision,omitempty"`
	AllowSync       bool   `json:"allowSync"`
	Confirmed       bool   `json:"confirmed"`
}

type SyncResult struct {
	ApplicationName string `json:"applicationName"`
	Requested       bool   `json:"requested"`
	Message         string `json:"message"`
}

type Provider interface {
	ValidateCredential(ctx context.Context, credential CredentialRef) error
	GetApplicationStatus(ctx context.Context, applicationName string) (ApplicationStatus, error)
	SyncApplication(ctx context.Context, request SyncRequest) (SyncResult, error)
}
