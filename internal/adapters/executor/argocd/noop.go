package argocd

import (
	"context"
	"fmt"
	"time"

	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
)

type NoopProvider struct {
	AllowSync    bool
	SyncStatus   string
	HealthStatus string
	SyncFails    bool
	WatchTimeout bool
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
	syncStatus := p.SyncStatus
	if syncStatus == "" {
		syncStatus = "Unknown"
	}
	healthStatus := p.HealthStatus
	if healthStatus == "" {
		healthStatus = "Unknown"
	}
	return portargocd.ApplicationStatus{
		ApplicationName: applicationName,
		Project:         "default",
		Namespace:       "argocd",
		SyncStatus:      syncStatus,
		HealthStatus:    healthStatus,
		Revision:        "noop-revision",
		TargetRevision:  "HEAD",
		Resources:       noopResources(applicationName, syncStatus, healthStatus),
		ObservedAt:      time.Now().UTC().Format(time.RFC3339),
		RawSummary:      "noop provider status",
		Message:         "noop Argo CD provider: status is modeled but no remote API was called",
		Warnings:        []string{"Argo CD status is local/noop in Phase 2.6"},
	}, nil
}

func (p NoopProvider) GetApplicationResources(ctx context.Context, applicationName string) ([]portargocd.ResourceStatus, error) {
	status, err := p.GetApplicationStatus(ctx, applicationName)
	if err != nil {
		return nil, err
	}
	return status.Resources, nil
}

func (p NoopProvider) GetApplicationHistory(ctx context.Context, applicationName string) ([]portargocd.ApplicationStatus, error) {
	status, err := p.GetApplicationStatus(ctx, applicationName)
	if err != nil {
		return nil, err
	}
	return []portargocd.ApplicationStatus{status}, nil
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
	if request.Force {
		return portargocd.SyncResult{}, fmt.Errorf("argocd force sync is not supported in Phase 2.6")
	}
	if p.SyncFails {
		return portargocd.SyncResult{ApplicationName: request.ApplicationName, Requested: true, Started: true, Message: "noop Argo CD sync failed"}, fmt.Errorf("noop Argo CD sync failed")
	}
	return portargocd.SyncResult{
		ApplicationName: request.ApplicationName,
		Requested:       true,
		Started:         true,
		Completed:       true,
		SyncStatus:      "Synced",
		HealthStatus:    "Healthy",
		Revision:        request.Revision,
		Message:         "noop Argo CD sync requested; no remote API was called",
	}, nil
}

func (p NoopProvider) WatchApplicationStatus(ctx context.Context, applicationName string, timeoutSeconds int) ([]portargocd.ApplicationStatus, error) {
	if p.WatchTimeout {
		return nil, fmt.Errorf("noop Argo CD watch timed out")
	}
	status, err := p.GetApplicationStatus(ctx, applicationName)
	if err != nil {
		return nil, err
	}
	final := status
	final.SyncStatus = "Synced"
	final.HealthStatus = "Healthy"
	final.Message = "noop Argo CD watch completed"
	return []portargocd.ApplicationStatus{status, final}, nil
}

func noopResources(applicationName string, syncStatus string, healthStatus string) []portargocd.ResourceStatus {
	return []portargocd.ResourceStatus{{
		Group:      "apps",
		Version:    "v1",
		Kind:       "Deployment",
		Namespace:  "default",
		Name:       applicationName,
		Status:     "Unknown",
		Health:     healthStatus,
		SyncStatus: syncStatus,
		Message:    "noop resource status",
	}}
}
