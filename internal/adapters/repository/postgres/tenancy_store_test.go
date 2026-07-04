package postgres

import (
	"context"
	"testing"

	domaintenant "github.com/sevoniva/nivora/internal/domain/tenant"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
)

func TestTenancyStoreImplementsRuntimeInterface(t *testing.T) {
	var _ tenancyusecase.Store = (*TenancyStore)(nil)
}

func TestPostgresIntegrationTenancyQuotaAndUsageRecovery(t *testing.T) {
	db := newPostgresIntegration(t, true)
	defer db.cleanup()

	ctx := context.Background()
	now := fixedIntegrationTime()
	scope := domaintenant.Scope{Type: domaintenant.ScopeProject, ID: "project-tenancy-recover"}
	store := NewTenancyStore(db.pool)

	quota := domaintenant.Quota{
		Scope:                       scope,
		MaxConcurrentPipelineRuns:   7,
		MaxConcurrentDeploymentRuns: 4,
		MaxRunners:                  11,
		MaxArtifactsTracked:         22,
		MaxLogStorageBytes:          33,
		APITokenRequestsPerMinute:   44,
		RunnerHeartbeatPerMinute:    55,
		JobClaimRequestsPerMinute:   66,
		DeploymentConcurrency:       3,
		PipelineConcurrency:         5,
		UpdatedAt:                   now,
	}
	if err := store.SaveQuota(ctx, quota); err != nil {
		t.Fatalf("save quota: %v", err)
	}
	usage := domaintenant.UsageSummary{
		Scope:                    scope,
		ConcurrentPipelineRuns:   2,
		ConcurrentDeploymentRuns: 3,
		Runners:                  5,
		ArtifactsTracked:         8,
		LogStorageBytes:          13,
		UpdatedAt:                now,
	}
	if err := store.SaveUsage(ctx, usage); err != nil {
		t.Fatalf("save usage: %v", err)
	}
	if err := store.SaveUsage(ctx, usage); err != nil {
		t.Fatalf("idempotently save usage: %v", err)
	}

	store = NewTenancyStore(db.restart(t))
	loadedQuota, err := store.GetQuota(ctx, scope.Type, scope.ID)
	if err != nil {
		t.Fatalf("reload quota: %v", err)
	}
	if loadedQuota.MaxConcurrentPipelineRuns != 7 || loadedQuota.MaxConcurrentDeploymentRuns != 4 || loadedQuota.MaxLogStorageBytes != 33 {
		t.Fatalf("loaded quota mismatch: %#v", loadedQuota)
	}

	loadedUsage, err := store.GetUsage(ctx, scope.Type, scope.ID)
	if err != nil {
		t.Fatalf("reload usage: %v", err)
	}
	if loadedUsage.ConcurrentPipelineRuns != 2 || loadedUsage.ConcurrentDeploymentRuns != 3 || loadedUsage.Runners != 5 || loadedUsage.ArtifactsTracked != 8 || loadedUsage.LogStorageBytes != 13 {
		t.Fatalf("loaded usage mismatch: %#v", loadedUsage)
	}
}
