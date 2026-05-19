package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domaintenant "github.com/sevoniva/nivora/internal/domain/tenant"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
)

type TenancyStore struct {
	pool *pgxpool.Pool
}

var _ tenancyusecase.Store = (*TenancyStore)(nil)

func NewTenancyStore(pool *pgxpool.Pool) *TenancyStore {
	return &TenancyStore{pool: pool}
}

func (s *TenancyStore) SaveQuota(ctx context.Context, quota domaintenant.Quota) error {
	payload, _ := json.Marshal(quota)
	_, err := s.pool.Exec(ctx, `INSERT INTO tenancy_quotas (id, scope_type, scope_id, max_pipelines_per_hour, max_deployments_per_hour, max_runners, max_parallel_jobs, max_secrets, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (scope_type, scope_id) DO UPDATE SET max_pipelines_per_hour=EXCLUDED.max_pipelines_per_hour, max_deployments_per_hour=EXCLUDED.max_deployments_per_hour, max_runners=EXCLUDED.max_runners, max_parallel_jobs=EXCLUDED.max_parallel_jobs, max_secrets=EXCLUDED.max_secrets, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`,
		quota.Scope.Type+"/"+quota.Scope.ID, quota.Scope.Type, quota.Scope.ID, quota.MaxConcurrentPipelineRuns, quota.MaxConcurrentDeploymentRuns, quota.MaxRunners, quota.DeploymentConcurrency, quota.MaxArtifactsTracked, payload, quota.UpdatedAt, quota.UpdatedAt)
	return err
}

func (s *TenancyStore) GetQuota(ctx context.Context, scopeType, scopeID string) (domaintenant.Quota, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT metadata FROM tenancy_quotas WHERE scope_type=$1 AND scope_id=$2`, scopeType, scopeID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaintenant.Quota{}, tenancyusecase.ErrQuotaNotFound
	}
	if err != nil {
		return domaintenant.Quota{}, err
	}
	var q domaintenant.Quota
	if err := json.Unmarshal(raw, &q); err != nil {
		return domaintenant.Quota{}, err
	}
	return q, nil
}

func (s *TenancyStore) SaveUsage(ctx context.Context, usage domaintenant.UsageSummary) error {
	payload, _ := json.Marshal(usage)
	_, err := s.pool.Exec(ctx, `INSERT INTO tenancy_usage_records (id, scope_type, scope_id, resource_type, resource_count, window_start, window_end, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		usage.Scope.Type+"/"+usage.Scope.ID+"-"+usage.UpdatedAt.Format("20060102T150405"), usage.Scope.Type, usage.Scope.ID, "summary", usage.ConcurrentPipelineRuns+usage.ConcurrentDeploymentRuns, usage.UpdatedAt, usage.UpdatedAt, usage.UpdatedAt)
	_ = payload
	return err
}

func (s *TenancyStore) GetUsage(ctx context.Context, scopeType, scopeID string) (domaintenant.UsageSummary, error) {
	var u domaintenant.UsageSummary
	err := s.pool.QueryRow(ctx, `SELECT scope_type, scope_id, resource_count, created_at FROM tenancy_usage_records WHERE scope_type=$1 AND scope_id=$2 ORDER BY created_at DESC LIMIT 1`, scopeType, scopeID).
		Scan(&u.Scope.Type, &u.Scope.ID, &u.ConcurrentPipelineRuns, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaintenant.UsageSummary{}, nil
	}
	if err != nil {
		return domaintenant.UsageSummary{}, err
	}
	return u, nil
}
