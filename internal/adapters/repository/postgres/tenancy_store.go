package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

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
	metrics := []struct {
		name  string
		count int64
	}{
		{name: "concurrent_pipeline_runs", count: int64(usage.ConcurrentPipelineRuns)},
		{name: "concurrent_deployment_runs", count: int64(usage.ConcurrentDeploymentRuns)},
		{name: "runners", count: int64(usage.Runners)},
		{name: "artifacts_tracked", count: int64(usage.ArtifactsTracked)},
		{name: "log_storage_bytes", count: usage.LogStorageBytes},
	}
	for _, metric := range metrics {
		id := fmt.Sprintf("%s/%s-%s-%s", usage.Scope.Type, usage.Scope.ID, usage.UpdatedAt.UTC().Format("20060102T150405.000000000"), metric.name)
		if _, err := s.pool.Exec(ctx, `INSERT INTO tenancy_usage_records (id, scope_type, scope_id, resource_type, resource_count, window_start, window_end, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			id, usage.Scope.Type, usage.Scope.ID, metric.name, metric.count, usage.UpdatedAt, usage.UpdatedAt, usage.UpdatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (s *TenancyStore) GetUsage(ctx context.Context, scopeType, scopeID string) (domaintenant.UsageSummary, error) {
	var latest sql.NullTime
	if err := s.pool.QueryRow(ctx, `SELECT max(window_start) FROM tenancy_usage_records WHERE scope_type=$1 AND scope_id=$2`, scopeType, scopeID).Scan(&latest); err != nil {
		return domaintenant.UsageSummary{}, err
	}
	if !latest.Valid {
		return domaintenant.UsageSummary{}, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT resource_type, resource_count, created_at FROM tenancy_usage_records WHERE scope_type=$1 AND scope_id=$2 AND window_start=$3`, scopeType, scopeID, latest.Time)
	if err != nil {
		return domaintenant.UsageSummary{}, err
	}
	defer rows.Close()
	u := domaintenant.UsageSummary{Scope: domaintenant.Scope{Type: scopeType, ID: scopeID}, UpdatedAt: latest.Time}
	for rows.Next() {
		var resourceType string
		var count int64
		if err := rows.Scan(&resourceType, &count, &u.UpdatedAt); err != nil {
			return domaintenant.UsageSummary{}, err
		}
		switch resourceType {
		case "concurrent_pipeline_runs":
			u.ConcurrentPipelineRuns = int(count)
		case "concurrent_deployment_runs":
			u.ConcurrentDeploymentRuns = int(count)
		case "runners":
			u.Runners = int(count)
		case "artifacts_tracked":
			u.ArtifactsTracked = int(count)
		case "log_storage_bytes":
			u.LogStorageBytes = count
		case "summary":
			u.ConcurrentPipelineRuns = int(count)
		}
	}
	if err := rows.Err(); err != nil {
		return domaintenant.UsageSummary{}, err
	}
	return u, nil
}
