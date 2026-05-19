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
