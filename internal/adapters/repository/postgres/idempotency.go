package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (s *DeploymentStore) RecordIdempotencyKey(ctx context.Context, scope string, key string, result IdempotencyResult) (IdempotencyResult, bool, error) {
	return recordIdempotencyKey(ctx, s.pool, scope, key, result)
}

func (s *DeploymentStore) GetIdempotencyResult(ctx context.Context, scope string, key string) (IdempotencyResult, error) {
	return getIdempotencyResult(ctx, s.pool, scope, key)
}

func (s *ReleaseStore) RecordIdempotencyKey(ctx context.Context, scope string, key string, result IdempotencyResult) (IdempotencyResult, bool, error) {
	return recordIdempotencyKey(ctx, s.pool, scope, key, result)
}

func (s *ReleaseStore) GetIdempotencyResult(ctx context.Context, scope string, key string) (IdempotencyResult, error) {
	return getIdempotencyResult(ctx, s.pool, scope, key)
}

func (s *ReleaseOrchestrationStore) RecordIdempotencyKey(ctx context.Context, scope string, key string, result IdempotencyResult) (IdempotencyResult, bool, error) {
	return recordIdempotencyKey(ctx, s.pool, scope, key, result)
}

func (s *ReleaseOrchestrationStore) GetIdempotencyResult(ctx context.Context, scope string, key string) (IdempotencyResult, error) {
	return getIdempotencyResult(ctx, s.pool, scope, key)
}

func recordIdempotencyKey(ctx context.Context, pool *pgxpool.Pool, scope string, key string, result IdempotencyResult) (IdempotencyResult, bool, error) {
	if scope == "" || key == "" {
		return IdempotencyResult{}, false, errors.New("idempotency scope and key are required")
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now()
	}
	tag, err := pool.Exec(ctx, `INSERT INTO idempotency_keys (scope, key, resource_type, resource_id, request_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`,
		scope, key, result.ResourceType, result.ResourceID, result.RequestHash, result.CreatedAt)
	if err != nil {
		return IdempotencyResult{}, false, err
	}
	if tag.RowsAffected() == 1 {
		return result, true, nil
	}
	existing, err := getIdempotencyResult(ctx, pool, scope, key)
	return existing, false, err
}

func getIdempotencyResult(ctx context.Context, pool *pgxpool.Pool, scope string, key string) (IdempotencyResult, error) {
	var result IdempotencyResult
	err := pool.QueryRow(ctx, `SELECT resource_type, resource_id, request_hash, created_at FROM idempotency_keys WHERE scope = $1 AND key = $2`, scope, key).
		Scan(&result.ResourceType, &result.ResourceID, &result.RequestHash, &result.CreatedAt)
	if err == pgx.ErrNoRows {
		return IdempotencyResult{}, pgx.ErrNoRows
	}
	return result, err
}
