package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
)

type ReleaseOrchestrationStore struct {
	pool *pgxpool.Pool
}

var _ releaseorchestration.Store = (*ReleaseOrchestrationStore)(nil)

func NewReleaseOrchestrationStore(pool *pgxpool.Pool) *ReleaseOrchestrationStore {
	return &ReleaseOrchestrationStore{pool: pool}
}

func (s *ReleaseOrchestrationStore) SavePlan(ctx context.Context, record releaseorchestration.PlanRecord) error {
	if record.Plan.ID == "" {
		return errors.New("release plan id is required")
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		return s.savePlan(ctx, tx, record)
	})
}

func (s *ReleaseOrchestrationStore) GetPlan(ctx context.Context, id string) (releaseorchestration.PlanRecord, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT record FROM runtime_release_plans WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return releaseorchestration.PlanRecord{}, releaseorchestration.ErrPlanNotFound
	}
	if err != nil {
		return releaseorchestration.PlanRecord{}, err
	}
	return decodeReleasePlanRecord(raw)
}

func (s *ReleaseOrchestrationStore) GetLatestPlanForRelease(ctx context.Context, releaseID string) (releaseorchestration.PlanRecord, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT record FROM runtime_release_plans WHERE release_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, releaseID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return releaseorchestration.PlanRecord{}, releaseorchestration.ErrPlanNotFound
	}
	if err != nil {
		return releaseorchestration.PlanRecord{}, err
	}
	return decodeReleasePlanRecord(raw)
}

func (s *ReleaseOrchestrationStore) SaveExecution(ctx context.Context, record releaseorchestration.ExecutionRecord) error {
	if record.Execution.ID == "" {
		return errors.New("release execution id is required")
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		return s.saveExecution(ctx, tx, record)
	})
}

func (s *ReleaseOrchestrationStore) GetExecution(ctx context.Context, id string) (releaseorchestration.ExecutionRecord, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT record FROM runtime_release_executions WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return releaseorchestration.ExecutionRecord{}, releaseorchestration.ErrExecutionNotFound
	}
	if err != nil {
		return releaseorchestration.ExecutionRecord{}, err
	}
	return decodeReleaseExecutionRecord(raw)
}

func (s *ReleaseOrchestrationStore) ListExecutions(ctx context.Context, releaseID string) ([]releaseorchestration.ExecutionRecord, error) {
	query := `SELECT record FROM runtime_release_executions ORDER BY created_at, id`
	args := []any{}
	if releaseID != "" {
		query = `SELECT record FROM runtime_release_executions WHERE release_id = $1 ORDER BY created_at, id`
		args = append(args, releaseID)
	}
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReleaseExecutionRecords(rows)
}

func (s *ReleaseOrchestrationStore) AppendEvent(ctx context.Context, executionID string, evt event.Event) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getExecutionForUpdate(ctx, tx, executionID)
		if err != nil {
			return err
		}
		record.Events = upsertEvent(record.Events, evt)
		if err := s.insertExecutionEvent(ctx, tx, executionID, evt); err != nil {
			return err
		}
		return s.saveExecution(ctx, tx, record)
	})
}

func (s *ReleaseOrchestrationStore) Events(ctx context.Context, executionID string) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM runtime_release_execution_events WHERE execution_id = $1 ORDER BY created_at, id`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (s *ReleaseOrchestrationStore) AppendAudit(ctx context.Context, executionID string, entry audit.AuditLog) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getExecutionForUpdate(ctx, tx, executionID)
		if err != nil {
			return err
		}
		record.Audits = upsertAudit(record.Audits, entry)
		if err := s.insertExecutionAudit(ctx, tx, executionID, entry); err != nil {
			return err
		}
		return s.saveExecution(ctx, tx, record)
	})
}

func (s *ReleaseOrchestrationStore) Audits(ctx context.Context, executionID string) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM runtime_release_execution_audit_logs WHERE execution_id = $1 ORDER BY created_at, id`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAudits(rows)
}

func (s *ReleaseOrchestrationStore) ListNonTerminalReleaseExecutions(ctx context.Context, limit int) ([]releaseorchestration.ExecutionRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_release_executions WHERE status NOT IN ('Succeeded', 'Failed', 'Canceled', 'RolledBack') ORDER BY created_at, id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReleaseExecutionRecords(rows)
}

func (s *ReleaseOrchestrationStore) ListStaleReleaseExecutions(ctx context.Context, olderThan time.Time, limit int) ([]releaseorchestration.ExecutionRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_release_executions WHERE status NOT IN ('Succeeded', 'Failed', 'Canceled', 'RolledBack') AND (updated_at < $1 OR lease_expires_at < $1) ORDER BY updated_at, id LIMIT $2`, olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReleaseExecutionRecords(rows)
}

func (s *ReleaseOrchestrationStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ReleaseOrchestrationStore) getExecutionForUpdate(ctx context.Context, tx pgx.Tx, id string) (releaseorchestration.ExecutionRecord, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT record FROM runtime_release_executions WHERE id = $1 FOR UPDATE`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return releaseorchestration.ExecutionRecord{}, releaseorchestration.ErrExecutionNotFound
	}
	if err != nil {
		return releaseorchestration.ExecutionRecord{}, err
	}
	return decodeReleaseExecutionRecord(raw)
}

func (s *ReleaseOrchestrationStore) savePlan(ctx context.Context, tx pgx.Tx, record releaseorchestration.PlanRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_release_plans (id, release_id, environment_id, strategy, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET release_id = EXCLUDED.release_id, environment_id = EXCLUDED.environment_id, strategy = EXCLUDED.strategy, record = EXCLUDED.record`,
		record.Plan.ID, record.Plan.ReleaseID, record.Plan.EnvironmentID, string(record.Plan.Strategy), raw, record.Plan.CreatedAt)
	return err
}

func (s *ReleaseOrchestrationStore) saveExecution(ctx context.Context, tx pgx.Tx, record releaseorchestration.ExecutionRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_release_executions (id, release_id, environment_id, status, reason, correlation_id, owner_id, lease_expires_at, attempt, heartbeat_at, record, created_at, updated_at, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET release_id = EXCLUDED.release_id, environment_id = EXCLUDED.environment_id, status = EXCLUDED.status, reason = EXCLUDED.reason, correlation_id = EXCLUDED.correlation_id, owner_id = EXCLUDED.owner_id, lease_expires_at = EXCLUDED.lease_expires_at, attempt = EXCLUDED.attempt, heartbeat_at = EXCLUDED.heartbeat_at, record = EXCLUDED.record, updated_at = EXCLUDED.updated_at, started_at = EXCLUDED.started_at, finished_at = EXCLUDED.finished_at, version = runtime_release_executions.version + 1`,
		record.Execution.ID, record.Execution.ReleaseID, record.Execution.EnvironmentID, string(record.Execution.Status), record.Execution.Reason, record.Execution.CorrelationID, record.Execution.OwnerID, record.Execution.LeaseExpiresAt, record.Execution.Attempt, record.Execution.HeartbeatAt, raw, record.Execution.CreatedAt, record.Execution.UpdatedAt, record.Execution.StartedAt, record.Execution.FinishedAt)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM runtime_release_execution_targets WHERE execution_id = $1`, record.Execution.ID); err != nil {
		return err
	}
	for _, target := range record.Execution.Targets {
		targetRaw, err := json.Marshal(target)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO runtime_release_execution_targets (execution_id, target_id, target_name, target_type, deployment_run_id, status, target_order, payload, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			record.Execution.ID, target.TargetID, target.TargetName, target.TargetType, target.DeploymentRunID, string(target.Status), target.Order, targetRaw, record.Execution.UpdatedAt)
		if err != nil {
			return err
		}
	}
	for _, evt := range record.Events {
		if err := s.insertExecutionEvent(ctx, tx, record.Execution.ID, evt); err != nil {
			return err
		}
	}
	for _, entry := range record.Audits {
		if err := s.insertExecutionAudit(ctx, tx, record.Execution.ID, entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReleaseOrchestrationStore) insertExecutionEvent(ctx context.Context, tx pgx.Tx, executionID string, evt event.Event) error {
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_release_execution_events (id, execution_id, event_type, source, subject, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO NOTHING`,
		evt.ID, executionID, evt.Type, evt.Source, evt.Subject, raw, evt.Time)
	return err
}

func (s *ReleaseOrchestrationStore) insertExecutionAudit(ctx context.Context, tx pgx.Tx, executionID string, entry audit.AuditLog) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_release_execution_audit_logs (id, execution_id, org_id, actor_id, action, subject, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id) DO NOTHING`,
		entry.ID, executionID, entry.OrgID, entry.ActorID, entry.Action, entry.Subject, raw, entry.CreatedAt)
	return err
}

func scanReleaseExecutionRecords(rows pgx.Rows) ([]releaseorchestration.ExecutionRecord, error) {
	var records []releaseorchestration.ExecutionRecord
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		record, err := decodeReleaseExecutionRecord(raw)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func decodeReleasePlanRecord(raw []byte) (releaseorchestration.PlanRecord, error) {
	var record releaseorchestration.PlanRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return releaseorchestration.PlanRecord{}, err
	}
	return record, nil
}

func decodeReleaseExecutionRecord(raw []byte) (releaseorchestration.ExecutionRecord, error) {
	var record releaseorchestration.ExecutionRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return releaseorchestration.ExecutionRecord{}, err
	}
	return record, nil
}
