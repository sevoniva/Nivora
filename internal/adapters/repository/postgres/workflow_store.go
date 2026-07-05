package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
		id, workflow_id, repository_id, source_path, ref, name, content_hash, plan, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	ON CONFLICT (id) DO UPDATE SET
		workflow_id=EXCLUDED.workflow_id,
		repository_id=EXCLUDED.repository_id,
		source_path=EXCLUDED.source_path,
		ref=EXCLUDED.ref,
		name=EXCLUDED.name,
		content_hash=EXCLUDED.content_hash,
		plan=EXCLUDED.plan,
		created_at=EXCLUDED.created_at`,
		record.ID, record.WorkflowID, record.RepositoryID, record.Path, record.Ref, record.Name, record.ContentHash, jsonBytes(record.Plan), record.CreatedAt)
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

func workflowPlanRecordSelect() string {
	return `SELECT id, workflow_id, repository_id, source_path, ref, name, content_hash, plan, created_at FROM workflow_plan_records`
}

func scanWorkflowPlanRecord(row scanner) (workflowusecase.PlanRecord, error) {
	var record workflowusecase.PlanRecord
	var plan []byte
	if err := row.Scan(&record.ID, &record.WorkflowID, &record.RepositoryID, &record.Path, &record.Ref, &record.Name, &record.ContentHash, &plan, &record.CreatedAt); err != nil {
		return workflowusecase.PlanRecord{}, err
	}
	readJSON(plan, &record.Plan)
	return record, nil
}
