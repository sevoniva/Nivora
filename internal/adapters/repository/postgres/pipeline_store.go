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
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

type PipelineStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

var (
	_ pipelineusecase.Store = (*PipelineStore)(nil)
	_ RecoveryQueries       = (*PipelineStore)(nil)
)

type RecoveryQueries interface {
	ListQueuedPipelineRuns(ctx context.Context, limit int) ([]pipelineusecase.RunRecord, error)
	ListStaleRunningPipelineRuns(ctx context.Context, olderThan time.Time, limit int) ([]pipelineusecase.RunRecord, error)
	ListExpiredJobClaims(ctx context.Context, now time.Time, limit int) ([]pipelineusecase.JobClaim, error)
}

type IdempotencyResult struct {
	ResourceType string
	ResourceID   string
	RequestHash  string
	CreatedAt    time.Time
}

func NewPipelineStore(pool *pgxpool.Pool) *PipelineStore {
	return &PipelineStore{pool: pool, now: time.Now}
}

func (s *PipelineStore) Save(ctx context.Context, record pipelineusecase.RunRecord) error {
	if record.Run.ID == "" {
		return errors.New("pipeline run id is required")
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		return s.saveRecord(ctx, tx, record)
	})
}

func (s *PipelineStore) Get(ctx context.Context, id string) (pipelineusecase.RunRecord, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT record FROM runtime_pipeline_runs WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return pipelineusecase.RunRecord{}, pipelineusecase.ErrRunNotFound
	}
	if err != nil {
		return pipelineusecase.RunRecord{}, err
	}
	return decodeRunRecord(raw)
}

func (s *PipelineStore) List(ctx context.Context) ([]pipelineusecase.RunRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_pipeline_runs ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunRecords(rows)
}

func (s *PipelineStore) ListFiltered(ctx context.Context, scopeType, scopeID string) ([]pipelineusecase.RunRecord, error) {
	if scopeType == "" {
		return s.List(ctx)
	}
	var rows pgx.Rows
	var err error
	if scopeType == "project" {
		rows, err = s.pool.Query(ctx, `SELECT record FROM runtime_pipeline_runs WHERE record->'pipeline'->>'projectId' = $1 ORDER BY created_at, id`, scopeID)
	} else {
		return []pipelineusecase.RunRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunRecords(rows)
}

func (s *PipelineStore) ListByStatus(ctx context.Context, status domainpipeline.PipelineRunStatus) ([]pipelineusecase.RunRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_pipeline_runs WHERE status = $1 ORDER BY created_at, id`, string(status))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunRecords(rows)
}

func (s *PipelineStore) AppendLog(ctx context.Context, runID string, log event.LogChunk) error {
	if log.ID == "" {
		return errors.New("log chunk id is required")
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, runID)
		if err != nil {
			return err
		}
		var next int64
		err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM runtime_log_chunks WHERE pipeline_run_id = $1`, runID).Scan(&next)
		if err != nil {
			return err
		}
		log.PipelineRunID = runID
		log.Sequence = next
		record.Logs = append(record.Logs, log)
		if err := s.insertLog(ctx, tx, log); err != nil {
			return err
		}
		return s.saveRecord(ctx, tx, record)
	})
}

func (s *PipelineStore) LogsByPipelineRun(ctx context.Context, runID string) ([]event.LogChunk, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, pipeline_run_id, stage_run_id, job_run_id, step_run_id, stream, sequence, content, created_at FROM runtime_log_chunks WHERE pipeline_run_id = $1 ORDER BY sequence, created_at, id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLogs(rows)
}

func (s *PipelineStore) LogsByJobRun(ctx context.Context, jobRunID string) ([]event.LogChunk, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, pipeline_run_id, stage_run_id, job_run_id, step_run_id, stream, sequence, content, created_at FROM runtime_log_chunks WHERE job_run_id = $1 ORDER BY sequence, created_at, id`, jobRunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLogs(rows)
}

func (s *PipelineStore) ClaimJob(ctx context.Context, runnerID string, leaseUntil time.Time) (pipelineusecase.JobClaim, error) {
	return s.claimJobAt(ctx, runnerID, leaseUntil, s.now())
}

func (s *PipelineStore) UpdateJobStatus(ctx context.Context, jobRunID string, status domainpipeline.JobRunStatus, reason string, at time.Time) (pipelineusecase.RunRecord, error) {
	var runID string
	err := s.pool.QueryRow(ctx, `SELECT pipeline_run_id FROM runtime_job_runs WHERE id = $1`, jobRunID).Scan(&runID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pipelineusecase.RunRecord{}, pipelineusecase.ErrJobNotFound
	}
	if err != nil {
		return pipelineusecase.RunRecord{}, err
	}
	var out pipelineusecase.RunRecord
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, runID)
		if err != nil {
			return err
		}
		if !updateRecordJobStatus(&record, jobRunID, status, reason, at) {
			return pipelineusecase.ErrJobNotFound
		}
		if err := s.saveRecord(ctx, tx, record); err != nil {
			return err
		}
		out = record
		return nil
	})
	return out, err
}

func (s *PipelineStore) RequestCancel(ctx context.Context, pipelineRunID string, at time.Time) (pipelineusecase.RunRecord, error) {
	var out pipelineusecase.RunRecord
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, pipelineRunID)
		if err != nil {
			return err
		}
		record.Run.CancelRequested = true
		record.Run.UpdatedAt = at
		if err := s.saveRecord(ctx, tx, record); err != nil {
			return err
		}
		out = record
		return nil
	})
	return out, err
}

func (s *PipelineStore) AppendEvent(ctx context.Context, runID string, evt event.Event) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, runID)
		if err != nil {
			return err
		}
		record.Events = upsertEvent(record.Events, evt)
		if err := s.insertEvent(ctx, tx, runID, evt); err != nil {
			return err
		}
		return s.saveRecord(ctx, tx, record)
	})
}

func (s *PipelineStore) EventsByPipelineRun(ctx context.Context, runID string) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM runtime_events WHERE pipeline_run_id = $1 ORDER BY created_at, id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []event.Event
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var evt event.Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, rows.Err()
}

func (s *PipelineStore) AppendOutbox(ctx context.Context, item pipelineusecase.EventOutboxRecord) error {
	raw, err := json.Marshal(item.Payload)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO runtime_event_outbox (id, event_type, subject, payload, status, retry_count, next_attempt_at, last_error, created_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, retry_count = EXCLUDED.retry_count, next_attempt_at = EXCLUDED.next_attempt_at, last_error = EXCLUDED.last_error, published_at = EXCLUDED.published_at`,
		item.ID, item.EventType, item.Subject, raw, item.Status, item.RetryCount, item.NextAttemptAt, item.LastError, item.CreatedAt, item.PublishedAt)
	return err
}

func (s *PipelineStore) ListPendingOutbox(ctx context.Context, limit int) ([]pipelineusecase.EventOutboxRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT id, event_type, subject, payload, status, retry_count, next_attempt_at, last_error, created_at, published_at
		FROM runtime_event_outbox
		WHERE status = 'pending' OR (status = 'failed' AND (next_attempt_at IS NULL OR next_attempt_at <= now()))
		ORDER BY created_at, id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []pipelineusecase.EventOutboxRecord
	for rows.Next() {
		var item pipelineusecase.EventOutboxRecord
		var raw []byte
		if err := rows.Scan(&item.ID, &item.EventType, &item.Subject, &raw, &item.Status, &item.RetryCount, &item.NextAttemptAt, &item.LastError, &item.CreatedAt, &item.PublishedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.Payload); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PipelineStore) MarkOutboxPublished(ctx context.Context, id string, at time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE runtime_event_outbox SET status = 'published', published_at = $2, last_error = '' WHERE id = $1`, id, at)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pipelineusecase.ErrOutboxNotFound
	}
	return nil
}

func (s *PipelineStore) MarkOutboxFailed(ctx context.Context, id string, retryCount int, nextAttemptAt time.Time, reason string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE runtime_event_outbox SET status = 'failed', retry_count = $2, next_attempt_at = $3, last_error = $4 WHERE id = $1`, id, retryCount, nextAttemptAt, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pipelineusecase.ErrOutboxNotFound
	}
	return nil
}

func (s *PipelineStore) AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error {
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, runID)
		if err != nil {
			return err
		}
		record.Audits = upsertAudit(record.Audits, entry)
		if err := s.insertAudit(ctx, tx, runID, entry); err != nil {
			return err
		}
		return s.saveRecord(ctx, tx, record)
	})
	if err != nil {
		return err
	}
	return AppendHashChainedAudit(ctx, s.pool, "pipeline", entry)
}

func (s *PipelineStore) AuditBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, org_id, actor_id, action, subject, created_at FROM runtime_audit_logs WHERE subject = $1 ORDER BY created_at, id`, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []audit.AuditLog
	for rows.Next() {
		var entry audit.AuditLog
		if err := rows.Scan(&entry.ID, &entry.OrgID, &entry.ActorID, &entry.Action, &entry.Subject, &entry.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *PipelineStore) SaveRunnerGroup(ctx context.Context, group domainrunner.RunnerGroup) error {
	labels, err := json.Marshal(nonNilMap(group.Labels))
	if err != nil {
		return err
	}
	environmentIDs, err := json.Marshal(group.EnvironmentIDs)
	if err != nil {
		return err
	}
	executors, err := json.Marshal(group.Executors)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO runtime_runner_groups (id, project_id, environment_ids, name, labels, max_concurrency, executors, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET project_id = EXCLUDED.project_id, environment_ids = EXCLUDED.environment_ids, name = EXCLUDED.name, labels = EXCLUDED.labels, max_concurrency = EXCLUDED.max_concurrency, executors = EXCLUDED.executors, updated_at = EXCLUDED.updated_at, version = runtime_runner_groups.version + 1`,
		group.ID, group.ProjectID, environmentIDs, group.Name, labels, group.MaxConcurrency, executors, group.CreatedAt, group.UpdatedAt)
	return err
}

func (s *PipelineStore) GetRunnerGroup(ctx context.Context, id string) (domainrunner.RunnerGroup, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, environment_ids, name, labels, max_concurrency, executors, created_at, updated_at FROM runtime_runner_groups WHERE id = $1`, id)
	if err != nil {
		return domainrunner.RunnerGroup{}, err
	}
	defer rows.Close()
	groups, err := scanRunnerGroups(rows)
	if err != nil {
		return domainrunner.RunnerGroup{}, err
	}
	if len(groups) == 0 {
		return domainrunner.RunnerGroup{}, pipelineusecase.ErrRunnerGroupNotFound
	}
	return groups[0], nil
}

func (s *PipelineStore) ListRunnerGroups(ctx context.Context) ([]domainrunner.RunnerGroup, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, project_id, environment_ids, name, labels, max_concurrency, executors, created_at, updated_at FROM runtime_runner_groups ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunnerGroups(rows)
}

func (s *PipelineStore) RegisterRunner(ctx context.Context, runner domainrunner.Runner) error {
	labels, err := json.Marshal(nonNilMap(runner.Labels))
	if err != nil {
		return err
	}
	executors, err := json.Marshal(runner.Executors)
	if err != nil {
		return err
	}
	capabilities, err := json.Marshal(runner.Capabilities)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO runtime_runners (id, name, group_id, status, labels, executors, capabilities, max_concurrency, token_id, token_hash, token_created_at, token_rotated_at, token_revoked_at, last_heartbeat_at, last_seen_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, group_id = EXCLUDED.group_id, status = EXCLUDED.status, labels = EXCLUDED.labels, executors = EXCLUDED.executors, capabilities = EXCLUDED.capabilities, max_concurrency = EXCLUDED.max_concurrency, token_id = EXCLUDED.token_id, token_hash = EXCLUDED.token_hash, token_created_at = EXCLUDED.token_created_at, token_rotated_at = EXCLUDED.token_rotated_at, token_revoked_at = EXCLUDED.token_revoked_at, last_heartbeat_at = EXCLUDED.last_heartbeat_at, last_seen_at = EXCLUDED.last_seen_at, updated_at = EXCLUDED.updated_at, version = runtime_runners.version + 1`,
		runner.ID, runner.Name, runner.GroupID, runner.Status, labels, executors, capabilities, runner.MaxConcurrency, runner.TokenID, runner.TokenHash, runner.TokenCreatedAt, runner.TokenRotatedAt, runner.TokenRevokedAt, runner.LastHeartbeatAt, runner.LastSeenAt, runner.CreatedAt, runner.UpdatedAt)
	return err
}

func (s *PipelineStore) Heartbeat(ctx context.Context, runnerID string, at time.Time) (domainrunner.Runner, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE runtime_runners SET status = 'online', last_heartbeat_at = $2, last_seen_at = $2, updated_at = $2, version = version + 1 WHERE id = $1`, runnerID, at)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainrunner.Runner{}, pipelineusecase.ErrRunnerNotFound
	}
	return s.GetRunner(ctx, runnerID)
}

func (s *PipelineStore) GetRunner(ctx context.Context, id string) (domainrunner.Runner, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, group_id, status, labels, executors, capabilities, max_concurrency, token_id, token_hash, token_created_at, token_rotated_at, token_revoked_at, last_heartbeat_at, last_seen_at, created_at, updated_at FROM runtime_runners WHERE id = $1`, id)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	defer rows.Close()
	runners, err := scanRunners(rows)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	if len(runners) == 0 {
		return domainrunner.Runner{}, pipelineusecase.ErrRunnerNotFound
	}
	return runners[0], nil
}

func (s *PipelineStore) ListRunners(ctx context.Context) ([]domainrunner.Runner, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, group_id, status, labels, executors, capabilities, max_concurrency, token_id, token_hash, token_created_at, token_rotated_at, token_revoked_at, last_heartbeat_at, last_seen_at, created_at, updated_at FROM runtime_runners ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunners(rows)
}

func (s *PipelineStore) SelectRunner(ctx context.Context, executor string, labels map[string]string) (domainrunner.Runner, error) {
	runners, err := s.ListRunners(ctx)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	for _, runner := range runners {
		if runner.Status != "online" || (!contains(runner.Executors, executor) && !contains(runner.Capabilities, executor)) || !labelsMatch(runner.Labels, labels) {
			continue
		}
		active, err := s.CountActiveJobs(ctx, runner.ID)
		if err != nil {
			return domainrunner.Runner{}, err
		}
		if runner.MaxConcurrency > 0 && active >= runner.MaxConcurrency {
			continue
		}
		runner.ActiveJobs = active
		return runner, nil
	}
	return domainrunner.Runner{}, pipelineusecase.ErrRunnerNotFound
}

func (s *PipelineStore) RotateRunnerToken(ctx context.Context, runnerID string, tokenID string, tokenHash string, at time.Time) (domainrunner.Runner, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE runtime_runners
		SET token_id = $2, token_hash = $3, token_rotated_at = CASE WHEN token_created_at IS NULL THEN NULL ELSE $4 END, token_created_at = COALESCE(token_created_at, $4), token_revoked_at = NULL, updated_at = $4, version = version + 1
		WHERE id = $1`, runnerID, tokenID, tokenHash, at)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainrunner.Runner{}, pipelineusecase.ErrRunnerNotFound
	}
	return s.GetRunner(ctx, runnerID)
}

func (s *PipelineStore) RevokeRunnerToken(ctx context.Context, runnerID string, at time.Time) (domainrunner.Runner, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE runtime_runners SET token_hash = '', token_revoked_at = $2, updated_at = $2, version = version + 1 WHERE id = $1`, runnerID, at)
	if err != nil {
		return domainrunner.Runner{}, err
	}
	if tag.RowsAffected() == 0 {
		return domainrunner.Runner{}, pipelineusecase.ErrRunnerNotFound
	}
	return s.GetRunner(ctx, runnerID)
}

func (s *PipelineStore) CountActiveJobs(ctx context.Context, runnerID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM runtime_job_runs WHERE runner_id = $1 AND status IN ($2, $3)`, runnerID, string(domainpipeline.JobRunAssigned), string(domainpipeline.JobRunRunning)).Scan(&count)
	return count, err
}

func (s *PipelineStore) CountActiveJobsByRunnerGroup(ctx context.Context, groupID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*)
		FROM runtime_job_runs jobs
		JOIN runtime_runners runners ON runners.id = jobs.runner_id
		WHERE runners.group_id = $1 AND jobs.status IN ($2, $3)`,
		groupID, string(domainpipeline.JobRunAssigned), string(domainpipeline.JobRunRunning)).Scan(&count)
	return count, err
}

func (s *PipelineStore) MarkOfflineRunners(ctx context.Context, cutoff time.Time, at time.Time) ([]domainrunner.Runner, error) {
	rows, err := s.pool.Query(ctx, `UPDATE runtime_runners
		SET status = 'offline', updated_at = $2, version = version + 1
		WHERE status = 'online' AND (last_heartbeat_at IS NULL OR last_heartbeat_at < $1)
		RETURNING id, name, group_id, status, labels, executors, capabilities, max_concurrency, token_id, token_hash, token_created_at, token_rotated_at, token_revoked_at, last_heartbeat_at, last_seen_at, created_at, updated_at`, cutoff, at)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunners(rows)
}

func (s *PipelineStore) RecordIdempotencyKey(ctx context.Context, scope string, key string, result IdempotencyResult) (IdempotencyResult, bool, error) {
	if scope == "" || key == "" {
		return IdempotencyResult{}, false, errors.New("idempotency scope and key are required")
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = s.now()
	}
	tag, err := s.pool.Exec(ctx, `INSERT INTO idempotency_keys (scope, key, resource_type, resource_id, request_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`,
		scope, key, result.ResourceType, result.ResourceID, result.RequestHash, result.CreatedAt)
	if err != nil {
		return IdempotencyResult{}, false, err
	}
	if tag.RowsAffected() == 1 {
		return result, true, nil
	}
	existing, err := s.GetIdempotencyResult(ctx, scope, key)
	return existing, false, err
}

func (s *PipelineStore) GetIdempotencyResult(ctx context.Context, scope string, key string) (IdempotencyResult, error) {
	var result IdempotencyResult
	err := s.pool.QueryRow(ctx, `SELECT resource_type, resource_id, request_hash, created_at FROM idempotency_keys WHERE scope = $1 AND key = $2`, scope, key).
		Scan(&result.ResourceType, &result.ResourceID, &result.RequestHash, &result.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return IdempotencyResult{}, pgx.ErrNoRows
	}
	return result, err
}

func (s *PipelineStore) ListQueuedPipelineRuns(ctx context.Context, limit int) ([]pipelineusecase.RunRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_pipeline_runs WHERE status = $1 ORDER BY created_at, id LIMIT $2`, string(domainpipeline.PipelineRunQueued), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunRecords(rows)
}

func (s *PipelineStore) ListStaleRunningPipelineRuns(ctx context.Context, olderThan time.Time, limit int) ([]pipelineusecase.RunRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_pipeline_runs WHERE status = $1 AND (updated_at < $2 OR lease_expires_at < $2) ORDER BY updated_at, id LIMIT $3`, string(domainpipeline.PipelineRunRunning), olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRunRecords(rows)
}

func (s *PipelineStore) AcquirePipelineRunLease(ctx context.Context, id string, ownerID string, leaseUntil time.Time, at time.Time) (pipelineusecase.RunRecord, error) {
	var out pipelineusecase.RunRecord
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		record.Run.OwnerID = ownerID
		record.Run.LeaseExpiresAt = &leaseUntil
		record.Run.HeartbeatAt = &at
		record.Run.UpdatedAt = at
		record.Run.Attempt++
		if record.Run.Attempt <= 0 {
			record.Run.Attempt = 1
		}
		if err := s.saveRecord(ctx, tx, record); err != nil {
			return err
		}
		out = record
		return nil
	})
	return out, err
}

func (s *PipelineStore) HeartbeatPipelineRunLease(ctx context.Context, id string, ownerID string, leaseUntil time.Time, at time.Time) (pipelineusecase.RunRecord, error) {
	var out pipelineusecase.RunRecord
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, id)
		if err != nil {
			return err
		}
		if record.Run.OwnerID != "" && record.Run.OwnerID != ownerID {
			return errors.New("pipeline run lease is owned by another worker")
		}
		record.Run.OwnerID = ownerID
		record.Run.LeaseExpiresAt = &leaseUntil
		record.Run.HeartbeatAt = &at
		record.Run.UpdatedAt = at
		if err := s.saveRecord(ctx, tx, record); err != nil {
			return err
		}
		out = record
		return nil
	})
	return out, err
}

func (s *PipelineStore) ListExpiredJobClaims(ctx context.Context, now time.Time, limit int) ([]pipelineusecase.JobClaim, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT j.pipeline_run_id, j.stage_run_id, j.id, j.runner_id, j.status, j.attempt, j.lease_expires_at, r.record
		FROM runtime_job_runs j
		JOIN runtime_pipeline_runs r ON r.id = j.pipeline_run_id
		WHERE j.status = $1 AND j.lease_expires_at < $2
		ORDER BY j.lease_expires_at, j.id LIMIT $3`, string(domainpipeline.JobRunAssigned), now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var claims []pipelineusecase.JobClaim
	for rows.Next() {
		var claim pipelineusecase.JobClaim
		var raw []byte
		var status string
		if err := rows.Scan(&claim.PipelineRunID, &claim.StageRunID, &claim.JobRunID, &claim.RunnerID, &status, &claim.Attempt, &claim.LeaseExpiresAt, &raw); err != nil {
			return nil, err
		}
		record, err := decodeRunRecord(raw)
		if err != nil {
			return nil, err
		}
		claim.Status = domainpipeline.JobRunStatus(status)
		claim.CancelRequested = record.Run.CancelRequested
		claim.StepRunIDs, claim.Commands, claim.Executor = claimDetails(record, claim.JobRunID)
		claims = append(claims, claim)
	}
	return claims, rows.Err()
}

func (s *PipelineStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
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

func (s *PipelineStore) getForUpdate(ctx context.Context, tx pgx.Tx, id string) (pipelineusecase.RunRecord, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT record FROM runtime_pipeline_runs WHERE id = $1 FOR UPDATE`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return pipelineusecase.RunRecord{}, pipelineusecase.ErrRunNotFound
	}
	if err != nil {
		return pipelineusecase.RunRecord{}, err
	}
	return decodeRunRecord(raw)
}

func (s *PipelineStore) saveRecord(ctx context.Context, tx pgx.Tx, record pipelineusecase.RunRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_pipeline_runs (id, pipeline_id, status, correlation_id, cancel_requested, owner_id, lease_expires_at, attempt, heartbeat_at, record, created_at, updated_at, started_at, finished_at, failure_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, correlation_id = EXCLUDED.correlation_id, cancel_requested = EXCLUDED.cancel_requested, owner_id = EXCLUDED.owner_id, lease_expires_at = EXCLUDED.lease_expires_at, attempt = EXCLUDED.attempt, heartbeat_at = EXCLUDED.heartbeat_at, record = EXCLUDED.record, updated_at = EXCLUDED.updated_at, started_at = EXCLUDED.started_at, finished_at = EXCLUDED.finished_at, failure_reason = EXCLUDED.failure_reason, version = runtime_pipeline_runs.version + 1`,
		record.Run.ID, record.Run.PipelineID, string(record.Run.Status), record.Run.CorrelationID, record.Run.CancelRequested, record.Run.OwnerID, record.Run.LeaseExpiresAt, record.Run.Attempt, record.Run.HeartbeatAt, raw, record.Run.CreatedAt, record.Run.UpdatedAt, record.Run.StartedAt, record.Run.FinishedAt, record.Run.FailureReason)
	if err != nil {
		return err
	}
	if err := s.replaceJobs(ctx, tx, record); err != nil {
		return err
	}
	for _, log := range record.Logs {
		if err := s.insertLog(ctx, tx, log); err != nil {
			return err
		}
	}
	for _, evt := range record.Events {
		if err := s.insertEvent(ctx, tx, record.Run.ID, evt); err != nil {
			return err
		}
	}
	for _, entry := range record.Audits {
		if err := s.insertAudit(ctx, tx, record.Run.ID, entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *PipelineStore) replaceJobs(ctx context.Context, tx pgx.Tx, record pipelineusecase.RunRecord) error {
	_, err := tx.Exec(ctx, `DELETE FROM runtime_job_runs WHERE pipeline_run_id = $1`, record.Run.ID)
	if err != nil {
		return err
	}
	for _, stage := range record.Stages {
		for _, job := range stage.Jobs {
			_, err := tx.Exec(ctx, `INSERT INTO runtime_job_runs (id, pipeline_run_id, stage_run_id, runner_id, name, status, attempt, max_retries, lease_expires_at, created_at, updated_at, started_at, finished_at, failure_reason)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
				job.Job.ID, record.Run.ID, stage.Stage.ID, job.Job.RunnerID, job.Job.Name, string(job.Job.Status), job.Job.Attempt, job.Job.MaxRetries, job.Job.LeaseExpiresAt, job.Job.CreatedAt, job.Job.UpdatedAt, job.Job.StartedAt, job.Job.FinishedAt, job.Job.FailureReason)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *PipelineStore) insertLog(ctx context.Context, tx pgx.Tx, log event.LogChunk) error {
	_, err := tx.Exec(ctx, `INSERT INTO runtime_log_chunks (id, pipeline_run_id, stage_run_id, job_run_id, step_run_id, stream, sequence, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO NOTHING`,
		log.ID, log.PipelineRunID, log.StageRunID, log.JobRunID, log.StepRunID, log.Stream, log.Sequence, log.Content, log.CreatedAt)
	return err
}

func (s *PipelineStore) insertEvent(ctx context.Context, tx pgx.Tx, runID string, evt event.Event) error {
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_events (id, pipeline_run_id, event_type, source, subject, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`,
		evt.ID, runID, evt.Type, evt.Source, evt.Subject, raw, evt.Time)
	return err
}

func (s *PipelineStore) insertAudit(ctx context.Context, tx pgx.Tx, runID string, entry audit.AuditLog) error {
	_, err := tx.Exec(ctx, `INSERT INTO runtime_audit_logs (id, pipeline_run_id, org_id, actor_id, action, subject, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING`,
		entry.ID, runID, entry.OrgID, entry.ActorID, entry.Action, entry.Subject, entry.CreatedAt)
	return err
}

func (s *PipelineStore) claimJobAt(ctx context.Context, runnerID string, leaseUntil time.Time, now time.Time) (pipelineusecase.JobClaim, error) {
	var claim pipelineusecase.JobClaim
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		runner, err := s.GetRunner(ctx, runnerID)
		if err != nil {
			return err
		}
		if runner.Status != "online" {
			return pipelineusecase.ErrRunnerNotFound
		}
		group := domainrunner.RunnerGroup{}
		if runner.GroupID != "" {
			group, err = s.GetRunnerGroup(ctx, runner.GroupID)
			if err != nil {
				return err
			}
		}
		active, err := s.CountActiveJobs(ctx, runnerID)
		if err != nil {
			return err
		}
		if runner.MaxConcurrency > 0 && active >= runner.MaxConcurrency {
			return pipelineusecase.ErrRunnerConcurrencyLimit
		}
		if group.ID != "" && group.MaxConcurrency > 0 {
			groupActive, err := s.CountActiveJobsByRunnerGroup(ctx, group.ID)
			if err != nil {
				return err
			}
			if groupActive >= group.MaxConcurrency {
				return pipelineusecase.ErrRunnerConcurrencyLimit
			}
		}
		rows, err := tx.Query(ctx, `SELECT record FROM runtime_pipeline_runs WHERE status IN ($1, $2) ORDER BY created_at, id FOR UPDATE SKIP LOCKED`, string(domainpipeline.PipelineRunQueued), string(domainpipeline.PipelineRunRunning))
		if err != nil {
			return err
		}
		var claimedRecord pipelineusecase.RunRecord
		found := false
		for rows.Next() {
			var raw []byte
			if err := rows.Scan(&raw); err != nil {
				rows.Close()
				return err
			}
			record, err := decodeRunRecord(raw)
			if err != nil {
				rows.Close()
				return err
			}
			next, ok := claimRecordJob(&record, runner, group, leaseUntil, now)
			if !ok {
				continue
			}
			claimedRecord = record
			claim = next
			found = true
			break
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if found {
			if err := s.saveRecord(ctx, tx, claimedRecord); err != nil {
				return err
			}
			return nil
		}
		return pipelineusecase.ErrNoClaimableJob
	})
	return claim, err
}

func scanRunRecords(rows pgx.Rows) ([]pipelineusecase.RunRecord, error) {
	var records []pipelineusecase.RunRecord
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		record, err := decodeRunRecord(raw)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func decodeRunRecord(raw []byte) (pipelineusecase.RunRecord, error) {
	var record pipelineusecase.RunRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return pipelineusecase.RunRecord{}, err
	}
	return record, nil
}

func scanLogs(rows pgx.Rows) ([]event.LogChunk, error) {
	var logs []event.LogChunk
	for rows.Next() {
		var log event.LogChunk
		if err := rows.Scan(&log.ID, &log.PipelineRunID, &log.StageRunID, &log.JobRunID, &log.StepRunID, &log.Stream, &log.Sequence, &log.Content, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func scanRunners(rows pgx.Rows) ([]domainrunner.Runner, error) {
	var runners []domainrunner.Runner
	for rows.Next() {
		var runner domainrunner.Runner
		var labelsRaw []byte
		var executorsRaw []byte
		var capabilitiesRaw []byte
		if err := rows.Scan(&runner.ID, &runner.Name, &runner.GroupID, &runner.Status, &labelsRaw, &executorsRaw, &capabilitiesRaw, &runner.MaxConcurrency, &runner.TokenID, &runner.TokenHash, &runner.TokenCreatedAt, &runner.TokenRotatedAt, &runner.TokenRevokedAt, &runner.LastHeartbeatAt, &runner.LastSeenAt, &runner.CreatedAt, &runner.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(labelsRaw, &runner.Labels); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(executorsRaw, &runner.Executors); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(capabilitiesRaw, &runner.Capabilities); err != nil {
			return nil, err
		}
		runners = append(runners, runner)
	}
	return runners, rows.Err()
}

func scanRunnerGroups(rows pgx.Rows) ([]domainrunner.RunnerGroup, error) {
	var groups []domainrunner.RunnerGroup
	for rows.Next() {
		var group domainrunner.RunnerGroup
		var environmentIDsRaw []byte
		var labelsRaw []byte
		var executorsRaw []byte
		if err := rows.Scan(&group.ID, &group.ProjectID, &environmentIDsRaw, &group.Name, &labelsRaw, &group.MaxConcurrency, &executorsRaw, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(environmentIDsRaw, &group.EnvironmentIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(labelsRaw, &group.Labels); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(executorsRaw, &group.Executors); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func updateRecordJobStatus(record *pipelineusecase.RunRecord, jobRunID string, status domainpipeline.JobRunStatus, reason string, at time.Time) bool {
	for stageIndex := range record.Stages {
		for jobIndex := range record.Stages[stageIndex].Jobs {
			job := &record.Stages[stageIndex].Jobs[jobIndex].Job
			if job.ID != jobRunID {
				continue
			}
			job.Status = status
			job.FailureReason = reason
			job.UpdatedAt = at
			if status == domainpipeline.JobRunRunning && job.StartedAt == nil {
				job.StartedAt = &at
			}
			if terminalJob(status) {
				job.FinishedAt = &at
				job.LeaseExpiresAt = nil
			}
			record.Run.UpdatedAt = at
			if status == domainpipeline.JobRunFailed || status == domainpipeline.JobRunCanceled {
				record.Run.Status = domainpipeline.PipelineRunFailed
				record.Run.FinishedAt = &at
			}
			return true
		}
	}
	return false
}

func claimRecordJob(record *pipelineusecase.RunRecord, runner domainrunner.Runner, group domainrunner.RunnerGroup, leaseUntil time.Time, now time.Time) (pipelineusecase.JobClaim, bool) {
	if !runnerCanClaimPipelineRecord(runner, group, *record) {
		return pipelineusecase.JobClaim{}, false
	}
	if record.Run.Status == domainpipeline.PipelineRunQueued {
		record.Run.Status = domainpipeline.PipelineRunRunning
		record.Run.StartedAt = &now
		record.Run.UpdatedAt = now
	}
	for stageIndex := range record.Stages {
		for jobIndex := range record.Stages[stageIndex].Jobs {
			job := &record.Stages[stageIndex].Jobs[jobIndex].Job
			executor := "shell"
			if stageIndex < len(record.Definition.Spec.Stages) && jobIndex < len(record.Definition.Spec.Stages[stageIndex].Jobs) {
				executor = record.Definition.Spec.Stages[stageIndex].Jobs[jobIndex].Executor
				if executor == "" {
					executor = "shell"
				}
			}
			if group.ID != "" && len(group.Executors) > 0 && !contains(group.Executors, executor) {
				continue
			}
			if !contains(runner.Executors, executor) && !contains(runner.Capabilities, executor) {
				continue
			}
			claimable := job.Status == domainpipeline.JobRunPending || job.Status == domainpipeline.JobRunRetrying
			leaseExpired := job.Status == domainpipeline.JobRunAssigned && job.LeaseExpiresAt != nil && job.LeaseExpiresAt.Before(now)
			if !claimable && !leaseExpired {
				continue
			}
			job.Status = domainpipeline.JobRunAssigned
			job.RunnerID = runner.ID
			job.LeaseExpiresAt = &leaseUntil
			job.UpdatedAt = now
			if job.Attempt <= 0 {
				job.Attempt = 1
			}
			stepIDs, commands, executor := claimDetails(*record, job.ID)
			return pipelineusecase.JobClaim{
				PipelineRunID:   record.Run.ID,
				StageRunID:      record.Stages[stageIndex].Stage.ID,
				JobRunID:        job.ID,
				StepRunIDs:      stepIDs,
				RunnerID:        runner.ID,
				Executor:        executor,
				Commands:        commands,
				Attempt:         job.Attempt,
				LeaseExpiresAt:  leaseUntil,
				CancelRequested: record.Run.CancelRequested,
				Status:          job.Status,
			}, true
		}
	}
	return pipelineusecase.JobClaim{}, false
}

func runnerCanClaimPipelineRecord(runner domainrunner.Runner, group domainrunner.RunnerGroup, record pipelineusecase.RunRecord) bool {
	if projectID := runner.Labels["projectId"]; projectID != "" && record.Pipeline.ProjectID != projectID {
		return false
	}
	if environmentID := runner.Labels["environmentId"]; environmentID != "" {
		recordEnvironmentID := firstNonEmpty(record.Pipeline.Labels["environmentId"], record.Pipeline.Metadata["environmentId"])
		if recordEnvironmentID != environmentID {
			return false
		}
	}
	if group.ProjectID != "" && record.Pipeline.ProjectID != group.ProjectID {
		return false
	}
	if len(group.EnvironmentIDs) > 0 {
		recordEnvironmentID := firstNonEmpty(record.Pipeline.Labels["environmentId"], record.Pipeline.Metadata["environmentId"])
		if !contains(group.EnvironmentIDs, recordEnvironmentID) {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func claimDetails(record pipelineusecase.RunRecord, jobRunID string) ([]string, []string, string) {
	for stageIndex, stage := range record.Stages {
		for jobIndex, job := range stage.Jobs {
			if job.Job.ID != jobRunID {
				continue
			}
			stepIDs := make([]string, 0, len(job.Steps))
			for _, step := range job.Steps {
				stepIDs = append(stepIDs, step.ID)
			}
			if stageIndex < len(record.Definition.Spec.Stages) && jobIndex < len(record.Definition.Spec.Stages[stageIndex].Jobs) {
				specJob := record.Definition.Spec.Stages[stageIndex].Jobs[jobIndex]
				commands := make([]string, 0, len(specJob.Steps))
				for _, step := range specJob.Steps {
					commands = append(commands, step.Run)
				}
				executor := specJob.Executor
				if executor == "" {
					executor = "shell"
				}
				return stepIDs, commands, executor
			}
			return stepIDs, nil, "shell"
		}
	}
	return nil, nil, "shell"
}

func upsertEvent(events []event.Event, evt event.Event) []event.Event {
	for i := range events {
		if events[i].ID == evt.ID {
			events[i] = evt
			return events
		}
	}
	return append(events, evt)
}

func upsertAudit(entries []audit.AuditLog, entry audit.AuditLog) []audit.AuditLog {
	for i := range entries {
		if entries[i].ID == entry.ID {
			entries[i] = entry
			return entries
		}
	}
	return append(entries, entry)
}

func terminalJob(status domainpipeline.JobRunStatus) bool {
	return status == domainpipeline.JobRunSucceeded || status == domainpipeline.JobRunFailed || status == domainpipeline.JobRunSkipped || status == domainpipeline.JobRunCanceled
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func labelsMatch(have map[string]string, want map[string]string) bool {
	for key, value := range want {
		if have[key] != value {
			return false
		}
	}
	return true
}

func nonNilMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	return values
}
