package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
)

type WorkflowStore struct {
	pool *pgxpool.Pool
}

var _ workflowusecase.Store = (*WorkflowStore)(nil)

func NewWorkflowStore(pool *pgxpool.Pool) *WorkflowStore {
	return &WorkflowStore{pool: pool}
}

func (s *WorkflowStore) SavePlan(ctx context.Context, record workflowusecase.PlanRecord) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO workflow_plan_records (
		id, workflow_id, repository_id, repository_snapshot_id, source_path, ref, name, content_hash, plan, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	ON CONFLICT (id) DO UPDATE SET
		workflow_id=EXCLUDED.workflow_id,
		repository_id=EXCLUDED.repository_id,
		repository_snapshot_id=EXCLUDED.repository_snapshot_id,
		source_path=EXCLUDED.source_path,
		ref=EXCLUDED.ref,
		name=EXCLUDED.name,
		content_hash=EXCLUDED.content_hash,
		plan=EXCLUDED.plan,
		created_at=EXCLUDED.created_at`,
		record.ID, record.WorkflowID, record.RepositoryID, record.RepositorySnapshotID, record.Path, record.Ref, record.Name, record.ContentHash, jsonBytes(record.Plan), record.CreatedAt)
	return err
}

func (s *WorkflowStore) GetPlan(ctx context.Context, id string) (workflowusecase.PlanRecord, error) {
	record, err := scanWorkflowPlanRecord(s.pool.QueryRow(ctx, workflowPlanRecordSelect()+` WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return workflowusecase.PlanRecord{}, fmt.Errorf("%w: workflow plan %q", workflowusecase.ErrNotFound, id)
	}
	return record, err
}

func (s *WorkflowStore) GetLatestPlan(ctx context.Context, workflowID string) (workflowusecase.PlanRecord, error) {
	record, err := scanWorkflowPlanRecord(s.pool.QueryRow(ctx, workflowPlanRecordSelect()+` WHERE workflow_id=$1 ORDER BY created_at DESC, id DESC LIMIT 1`, workflowID))
	if errors.Is(err, pgx.ErrNoRows) {
		return workflowusecase.PlanRecord{}, fmt.Errorf("%w: latest workflow plan for %q", workflowusecase.ErrNotFound, workflowID)
	}
	return record, err
}

func (s *WorkflowStore) ListPlans(ctx context.Context, filter workflowusecase.PlanListFilter) ([]workflowusecase.PlanRecord, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, workflowPlanRecordSelect()+` WHERE ($1='' OR repository_id=$1) AND ($2='' OR workflow_id=$2) ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`, filter.RepositoryID, filter.WorkflowID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []workflowusecase.PlanRecord
	for rows.Next() {
		record, err := scanWorkflowPlanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return nonNil(out), rows.Err()
}

func (s *WorkflowStore) SaveRun(ctx context.Context, record workflowusecase.RunRecord) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO workflow_run_records (
		id, workflow_id, workflow_plan_id, repository_id, repository_snapshot_id, pipeline_run_id, pipeline_id, project_id, environment_id, ref, status, warnings, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	ON CONFLICT (id) DO UPDATE SET
		workflow_id=EXCLUDED.workflow_id,
		workflow_plan_id=EXCLUDED.workflow_plan_id,
		repository_id=EXCLUDED.repository_id,
		repository_snapshot_id=EXCLUDED.repository_snapshot_id,
		pipeline_run_id=EXCLUDED.pipeline_run_id,
		pipeline_id=EXCLUDED.pipeline_id,
		project_id=EXCLUDED.project_id,
		environment_id=EXCLUDED.environment_id,
		ref=EXCLUDED.ref,
		status=EXCLUDED.status,
		warnings=EXCLUDED.warnings,
		updated_at=EXCLUDED.updated_at`,
		record.ID, record.WorkflowID, record.WorkflowPlanID, record.RepositoryID, record.RepositorySnapshotID, record.PipelineRunID, record.PipelineID, record.ProjectID, record.EnvironmentID, record.Ref, string(record.Status), jsonBytes(record.Warnings), record.CreatedAt, record.UpdatedAt)
	return err
}

func (s *WorkflowStore) GetRun(ctx context.Context, id string) (workflowusecase.RunRecord, error) {
	record, err := scanWorkflowRunRecord(s.pool.QueryRow(ctx, workflowRunRecordSelect()+` WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return workflowusecase.RunRecord{}, fmt.Errorf("%w: workflow run %q", workflowusecase.ErrNotFound, id)
	}
	return record, err
}

func (s *WorkflowStore) ListRuns(ctx context.Context, filter workflowusecase.RunListFilter) ([]workflowusecase.RunRecord, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, workflowRunRecordSelect()+` WHERE ($1='' OR repository_id=$1) AND ($2='' OR workflow_id=$2) AND ($3='' OR project_id=$3) AND ($4='' OR status=$4) ORDER BY created_at DESC, id DESC LIMIT $5 OFFSET $6`,
		filter.RepositoryID, filter.WorkflowID, filter.ProjectID, string(filter.Status), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []workflowusecase.RunRecord
	for rows.Next() {
		record, err := scanWorkflowRunRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return nonNil(out), rows.Err()
}

func (s *WorkflowStore) AppendEvent(ctx context.Context, subject string, evt event.Event) error {
	if evt.Time.IsZero() {
		evt.Time = time.Now().UTC()
	}
	if evt.Subject == "" {
		evt.Subject = subject
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO governance_event_logs (id, source, event_type, subject, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (id) DO NOTHING`,
		evt.ID, "workflow", evt.Type, subject, payload, evt.Time)
	return err
}

func (s *WorkflowStore) EventsBySubject(ctx context.Context, subject string) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_event_logs WHERE source='workflow' AND subject=$1 ORDER BY created_at, id`, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []event.Event
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var evt event.Event
		if err := json.Unmarshal(payload, &evt); err != nil {
			return nil, err
		}
		out = append(out, evt)
	}
	return nonNil(out), rows.Err()
}

func (s *WorkflowStore) AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error {
	if entry.Subject == "" {
		entry.Subject = subject
	}
	return AppendHashChainedAudit(ctx, s.pool, "workflow", entry)
}

func (s *WorkflowStore) AuditsBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_audit_logs WHERE source='workflow' AND subject=$1 ORDER BY created_at, id`, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []audit.AuditLog
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var entry audit.AuditLog
		if err := json.Unmarshal(payload, &entry); err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	return nonNil(out), rows.Err()
}

func workflowPlanRecordSelect() string {
	return `SELECT id, workflow_id, repository_id, repository_snapshot_id, source_path, ref, name, content_hash, plan, created_at FROM workflow_plan_records`
}

func workflowRunRecordSelect() string {
	return `SELECT id, workflow_id, workflow_plan_id, repository_id, repository_snapshot_id, pipeline_run_id, pipeline_id, project_id, environment_id, ref, status, warnings, created_at, updated_at FROM workflow_run_records`
}

func scanWorkflowPlanRecord(row scanner) (workflowusecase.PlanRecord, error) {
	var record workflowusecase.PlanRecord
	var plan []byte
	if err := row.Scan(&record.ID, &record.WorkflowID, &record.RepositoryID, &record.RepositorySnapshotID, &record.Path, &record.Ref, &record.Name, &record.ContentHash, &plan, &record.CreatedAt); err != nil {
		return workflowusecase.PlanRecord{}, err
	}
	readJSON(plan, &record.Plan)
	return record, nil
}

func scanWorkflowRunRecord(row scanner) (workflowusecase.RunRecord, error) {
	var record workflowusecase.RunRecord
	var status string
	var warnings []byte
	if err := row.Scan(&record.ID, &record.WorkflowID, &record.WorkflowPlanID, &record.RepositoryID, &record.RepositorySnapshotID, &record.PipelineRunID, &record.PipelineID, &record.ProjectID, &record.EnvironmentID, &record.Ref, &status, &warnings, &record.CreatedAt, &record.UpdatedAt); err != nil {
		return workflowusecase.RunRecord{}, err
	}
	record.Status = workflowusecase.RunStatus(status)
	readJSON(warnings, &record.Warnings)
	return record, nil
}
