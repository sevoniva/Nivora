package tenancy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	domaintenant "github.com/sevoniva/nivora/internal/domain/tenant"
)

var ErrScopeDenied = errors.New("tenant scope denied")

type Service struct {
	mu     sync.RWMutex
	store  Store
	quotas map[string]domaintenant.Quota
	usage  map[string]domaintenant.UsageSummary
	now    func() time.Time
}

func NewService() *Service {
	return NewServiceWithStore(NewMemoryStore())
}

func NewServiceWithStore(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{
		store:  store,
		quotas: make(map[string]domaintenant.Quota),
		usage:  make(map[string]domaintenant.UsageSummary),
		now:    time.Now,
	}
}

func (s *Service) Quota(ctx context.Context, input ScopeInput) (domaintenant.Quota, error) {
	if err := ctx.Err(); err != nil {
		return domaintenant.Quota{}, err
	}
	scope := normalizeScope(input.ScopeType, input.ScopeID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if quota, ok := s.quotas[scopeKey(scope)]; ok {
		return quota, nil
	}
	return defaultQuota(scope, s.now()), nil
}

func (s *Service) SetQuota(ctx context.Context, input QuotaUpdateInput) (domaintenant.Quota, error) {
	if err := ctx.Err(); err != nil {
		return domaintenant.Quota{}, err
	}
	scope := normalizeScope(input.ScopeType, input.ScopeID)
	quota := defaultQuota(scope, s.now())
	quota.MaxConcurrentPipelineRuns = overrideInt(quota.MaxConcurrentPipelineRuns, input.MaxConcurrentPipelineRuns)
	quota.MaxConcurrentDeploymentRuns = overrideInt(quota.MaxConcurrentDeploymentRuns, input.MaxConcurrentDeploymentRuns)
	quota.MaxRunners = overrideInt(quota.MaxRunners, input.MaxRunners)
	quota.MaxArtifactsTracked = overrideInt(quota.MaxArtifactsTracked, input.MaxArtifactsTracked)
	quota.MaxLogStorageBytes = overrideInt64(quota.MaxLogStorageBytes, input.MaxLogStorageBytes)
	quota.APITokenRequestsPerMinute = overrideInt(quota.APITokenRequestsPerMinute, input.APITokenRequestsPerMinute)
	quota.RunnerHeartbeatPerMinute = overrideInt(quota.RunnerHeartbeatPerMinute, input.RunnerHeartbeatPerMinute)
	quota.JobClaimRequestsPerMinute = overrideInt(quota.JobClaimRequestsPerMinute, input.JobClaimRequestsPerMinute)
	quota.DeploymentConcurrency = overrideInt(quota.DeploymentConcurrency, input.DeploymentConcurrency)
	quota.PipelineConcurrency = overrideInt(quota.PipelineConcurrency, input.PipelineConcurrency)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.quotas[scopeKey(scope)] = quota
	if s.store != nil {
		_ = s.store.SaveQuota(ctx, quota)
	}
	return quota, nil
}

func (s *Service) Usage(ctx context.Context, input ScopeInput) (domaintenant.UsageSummary, error) {
	if err := ctx.Err(); err != nil {
		return domaintenant.UsageSummary{}, err
	}
	scope := normalizeScope(input.ScopeType, input.ScopeID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	if usage, ok := s.usage[scopeKey(scope)]; ok {
		return usage, nil
	}
	return domaintenant.UsageSummary{Scope: scope, UpdatedAt: s.now()}, nil
}

func (s *Service) RecordUsage(ctx context.Context, update UsageUpdate) (domaintenant.UsageSummary, error) {
	if err := ctx.Err(); err != nil {
		return domaintenant.UsageSummary{}, err
	}
	update.Scope = normalizeScope(update.Scope.Type, update.Scope.ID)
	usage := domaintenant.UsageSummary{
		Scope:                    update.Scope,
		ConcurrentPipelineRuns:   update.ConcurrentPipelineRuns,
		ConcurrentDeploymentRuns: update.ConcurrentDeploymentRuns,
		Runners:                  update.Runners,
		ArtifactsTracked:         update.ArtifactsTracked,
		LogStorageBytes:          update.LogStorageBytes,
		UpdatedAt:                s.now(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage[scopeKey(update.Scope)] = usage
	if s.store != nil {
		_ = s.store.SaveUsage(ctx, usage)
	}
	return usage, nil
}

func (s *Service) CheckQuota(ctx context.Context, scope domaintenant.Scope) (domaintenant.QuotaCheck, error) {
	quota, err := s.Quota(ctx, ScopeInput{ScopeType: scope.Type, ScopeID: scope.ID})
	if err != nil {
		return domaintenant.QuotaCheck{}, err
	}
	usage, err := s.Usage(ctx, ScopeInput{ScopeType: scope.Type, ScopeID: scope.ID})
	if err != nil {
		return domaintenant.QuotaCheck{}, err
	}
	checks := []struct {
		name  string
		used  int64
		limit int64
	}{
		{"concurrent pipeline runs", int64(usage.ConcurrentPipelineRuns), int64(quota.MaxConcurrentPipelineRuns)},
		{"concurrent deployment runs", int64(usage.ConcurrentDeploymentRuns), int64(quota.MaxConcurrentDeploymentRuns)},
		{"runners", int64(usage.Runners), int64(quota.MaxRunners)},
		{"artifacts tracked", int64(usage.ArtifactsTracked), int64(quota.MaxArtifactsTracked)},
		{"log storage bytes", usage.LogStorageBytes, quota.MaxLogStorageBytes},
	}
	for _, check := range checks {
		if check.limit >= 0 && check.used > check.limit {
			return domaintenant.QuotaCheck{Allowed: false, Reason: fmt.Sprintf("%s quota exceeded", check.name), Limit: check.limit, Used: check.used}, nil
		}
	}
	return domaintenant.QuotaCheck{Allowed: true}, nil
}

func (s *Service) CanAccessScope(subject domainauth.Subject, scopeType string, scopeID string) domaintenant.IsolationDecision {
	target := normalizeScope(scopeType, scopeID)
	if subject.ScopeType == "" || subject.ScopeType == domaintenant.ScopeGlobal {
		return domaintenant.IsolationDecision{Allowed: true, Reason: "subject is not tenant-scoped"}
	}
	if target.Type == "" || target.Type == domaintenant.ScopeGlobal {
		return domaintenant.IsolationDecision{Allowed: true, Reason: "resource is not tenant-scoped"}
	}
	if subject.ScopeType == target.Type && subject.ScopeID == target.ID {
		return domaintenant.IsolationDecision{Allowed: true, Reason: "scope matched"}
	}
	return domaintenant.IsolationDecision{Allowed: false, Reason: "subject scope does not match resource scope"}
}

func (s *Service) CanUseRunnerGroup(subject domainauth.Subject, group domainrunner.RunnerGroup) domaintenant.IsolationDecision {
	if subject.ScopeType == "" || subject.ScopeType == domaintenant.ScopeGlobal {
		return domaintenant.IsolationDecision{Allowed: true, Reason: "subject is not tenant-scoped"}
	}
	if subject.ScopeType == domaintenant.ScopeProject && group.ProjectID != "" && subject.ScopeID != group.ProjectID {
		return domaintenant.IsolationDecision{Allowed: false, Reason: "runner group project does not match subject scope"}
	}
	if subject.ScopeType == domaintenant.ScopeEnvironment && len(group.EnvironmentIDs) > 0 {
		for _, environmentID := range group.EnvironmentIDs {
			if environmentID == subject.ScopeID {
				return domaintenant.IsolationDecision{Allowed: true, Reason: "runner group environment matched"}
			}
		}
		return domaintenant.IsolationDecision{Allowed: false, Reason: "runner group environment does not match subject scope"}
	}
	return domaintenant.IsolationDecision{Allowed: true, Reason: "runner group has no conflicting scope"}
}

func defaultQuota(scope domaintenant.Scope, now time.Time) domaintenant.Quota {
	return domaintenant.Quota{
		Scope:                       scope,
		MaxConcurrentPipelineRuns:   5,
		MaxConcurrentDeploymentRuns: 3,
		MaxRunners:                  10,
		MaxArtifactsTracked:         1000,
		MaxLogStorageBytes:          1 << 30,
		APITokenRequestsPerMinute:   600,
		RunnerHeartbeatPerMinute:    120,
		JobClaimRequestsPerMinute:   120,
		DeploymentConcurrency:       3,
		PipelineConcurrency:         5,
		UpdatedAt:                   now,
	}
}

func normalizeScope(scopeType string, scopeID string) domaintenant.Scope {
	if scopeType == "" {
		scopeType = domaintenant.ScopeGlobal
	}
	return domaintenant.Scope{Type: scopeType, ID: scopeID}
}

func scopeKey(scope domaintenant.Scope) string {
	return scope.Type + ":" + scope.ID
}

func overrideInt(current int, next int) int {
	if next == 0 {
		return current
	}
	return next
}

func overrideInt64(current int64, next int64) int64 {
	if next == 0 {
		return current
	}
	return next
}
