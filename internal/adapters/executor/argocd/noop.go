package argocd

import (
	"context"
	"fmt"

	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
)

type NoopProvider struct {
	AllowSync bool
}

func (p NoopProvider) ValidateCredential(ctx context.Context, credential portargocd.CredentialRef) error {
	return ctx.Err()
}

func (p NoopProvider) GetApplicationStatus(ctx context.Context, applicationName string) (portargocd.ApplicationStatus, error) {
	if err := ctx.Err(); err != nil {
		return portargocd.ApplicationStatus{}, err
	}
	if applicationName == "" {
		return portargocd.ApplicationStatus{}, fmt.Errorf("argocd application name is required")
	}
	return portargocd.ApplicationStatus{
		ApplicationName: applicationName,
		SyncStatus:      "Unknown",
		HealthStatus:    "Unknown",
		Message:         "noop Argo CD provider: status is modeled but no remote API was called",
		Warnings:        []string{"Argo CD status is local/noop in Phase 2.3"},
	}, nil
}

func (p NoopProvider) SyncApplication(ctx context.Context, request portargocd.SyncRequest) (portargocd.SyncResult, error) {
	if err := ctx.Err(); err != nil {
		return portargocd.SyncResult{}, err
	}
	if request.ApplicationName == "" {
		return portargocd.SyncResult{}, fmt.Errorf("argocd application name is required")
	}
	if !p.AllowSync || !request.AllowSync || !request.Confirmed {
		return portargocd.SyncResult{
			ApplicationName: request.ApplicationName,
			Requested:       false,
			Message:         "Argo CD sync skipped; sync is disabled by default and requires explicit allowSync and confirmation",
		}, nil
	}
	return portargocd.SyncResult{
		ApplicationName: request.ApplicationName,
		Requested:       true,
		Message:         "noop Argo CD sync requested; no remote API was called",
	}, nil
}
