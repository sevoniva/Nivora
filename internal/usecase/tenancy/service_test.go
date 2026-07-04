package tenancy

import (
	"context"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	domaintenant "github.com/sevoniva/nivora/internal/domain/tenant"
)

func TestScopedSubjectCannotAccessAnotherProject(t *testing.T) {
	service := NewService()
	decision := service.CanAccessScope(domainauth.Subject{ID: "sa-a", ScopeType: domaintenant.ScopeProject, ScopeID: "project-a"}, domaintenant.ScopeProject, "project-b")
	if decision.Allowed {
		t.Fatalf("expected cross-project access denial")
	}
}

func TestCredentialScopeAllowedForMatchingProject(t *testing.T) {
	service := NewService()
	decision := service.CanAccessScope(domainauth.Subject{ID: "sa-a", ScopeType: domaintenant.ScopeProject, ScopeID: "project-a"}, domaintenant.ScopeProject, "project-a")
	if !decision.Allowed {
		t.Fatalf("expected matching project access, got %q", decision.Reason)
	}
}

func TestRunnerGroupScopeEnforced(t *testing.T) {
	service := NewService()
	decision := service.CanUseRunnerGroup(
		domainauth.Subject{ID: "runner-admin", ScopeType: domaintenant.ScopeProject, ScopeID: "project-a"},
		domainrunner.RunnerGroup{ID: "group-b", ProjectID: "project-b"},
	)
	if decision.Allowed {
		t.Fatalf("expected runner group project mismatch to be denied")
	}
}

func TestQuotaExceeded(t *testing.T) {
	service := NewService()
	scope := domaintenant.Scope{Type: domaintenant.ScopeProject, ID: "project-a"}
	if _, err := service.SetQuota(context.Background(), QuotaUpdateInput{ScopeType: scope.Type, ScopeID: scope.ID, MaxConcurrentPipelineRuns: 1}); err != nil {
		t.Fatalf("set quota: %v", err)
	}
	if _, err := service.RecordUsage(context.Background(), UsageUpdate{Scope: scope, ConcurrentPipelineRuns: 2}); err != nil {
		t.Fatalf("record usage: %v", err)
	}
	check, err := service.CheckQuota(context.Background(), scope)
	if err != nil {
		t.Fatalf("check quota: %v", err)
	}
	if check.Allowed {
		t.Fatalf("expected quota denial")
	}
}

func TestQuotaAndUsageRecoverFromStore(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	scope := domaintenant.Scope{Type: domaintenant.ScopeProject, ID: "project-recover"}
	service := NewServiceWithStore(store)
	if _, err := service.SetQuota(ctx, QuotaUpdateInput{
		ScopeType:                   scope.Type,
		ScopeID:                     scope.ID,
		MaxConcurrentPipelineRuns:   7,
		MaxConcurrentDeploymentRuns: 4,
		MaxRunners:                  11,
		MaxArtifactsTracked:         22,
		MaxLogStorageBytes:          33,
	}); err != nil {
		t.Fatalf("set quota: %v", err)
	}
	if _, err := service.RecordUsage(ctx, UsageUpdate{
		Scope:                    scope,
		ConcurrentPipelineRuns:   2,
		ConcurrentDeploymentRuns: 3,
		Runners:                  5,
		ArtifactsTracked:         8,
		LogStorageBytes:          13,
	}); err != nil {
		t.Fatalf("record usage: %v", err)
	}

	restarted := NewServiceWithStore(store)
	quota, err := restarted.Quota(ctx, ScopeInput{ScopeType: scope.Type, ScopeID: scope.ID})
	if err != nil {
		t.Fatalf("recover quota from store: %v", err)
	}
	if quota.MaxConcurrentPipelineRuns != 7 || quota.MaxConcurrentDeploymentRuns != 4 || quota.MaxLogStorageBytes != 33 {
		t.Fatalf("recovered quota mismatch: %#v", quota)
	}
	usage, err := restarted.Usage(ctx, ScopeInput{ScopeType: scope.Type, ScopeID: scope.ID})
	if err != nil {
		t.Fatalf("recover usage from store: %v", err)
	}
	if usage.ConcurrentPipelineRuns != 2 || usage.ConcurrentDeploymentRuns != 3 || usage.Runners != 5 || usage.ArtifactsTracked != 8 || usage.LogStorageBytes != 13 {
		t.Fatalf("recovered usage mismatch: %#v", usage)
	}
}
