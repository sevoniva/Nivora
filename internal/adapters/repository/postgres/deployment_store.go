package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

type DeploymentStore struct {
	pool *pgxpool.Pool
}

var _ deploymentusecase.Store = (*DeploymentStore)(nil)

func NewDeploymentStore(pool *pgxpool.Pool) *DeploymentStore {
	return &DeploymentStore{pool: pool}
}

func (s *DeploymentStore) Save(ctx context.Context, record deploymentusecase.RunRecord) error {
	if record.Run.ID == "" {
		return errors.New("deployment run id is required")
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		return s.saveRecord(ctx, tx, record)
	})
}

func (s *DeploymentStore) Get(ctx context.Context, id string) (deploymentusecase.RunRecord, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT record FROM runtime_deployment_runs WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return deploymentusecase.RunRecord{}, deploymentusecase.ErrRunNotFound
	}
	if err != nil {
		return deploymentusecase.RunRecord{}, err
	}
	return decodeDeploymentRecord(raw)
}

func (s *DeploymentStore) List(ctx context.Context) ([]deploymentusecase.RunRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_deployment_runs ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDeploymentRecords(rows)
}

func (s *DeploymentStore) SaveHostGroup(ctx context.Context, group deploymentusecase.HostGroup) error {
	raw, err := json.Marshal(group)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO runtime_deployment_host_groups (id, environment_id, name, payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET environment_id = EXCLUDED.environment_id, name = EXCLUDED.name, payload = EXCLUDED.payload, updated_at = EXCLUDED.updated_at`,
		group.ID, group.EnvironmentID, group.Name, raw, group.CreatedAt, group.UpdatedAt)
	return err
}

func (s *DeploymentStore) GetHostGroup(ctx context.Context, id string) (deploymentusecase.HostGroup, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM runtime_deployment_host_groups WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return deploymentusecase.HostGroup{}, deploymentusecase.ErrHostGroupNotFound
	}
	if err != nil {
		return deploymentusecase.HostGroup{}, err
	}
	var group deploymentusecase.HostGroup
	if err := json.Unmarshal(raw, &group); err != nil {
		return deploymentusecase.HostGroup{}, err
	}
	return group, nil
}

func (s *DeploymentStore) ListHostGroups(ctx context.Context) ([]deploymentusecase.HostGroup, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM runtime_deployment_host_groups ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []deploymentusecase.HostGroup
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var group deploymentusecase.HostGroup
		if err := json.Unmarshal(raw, &group); err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (s *DeploymentStore) AppendLog(ctx context.Context, runID string, log event.LogChunk) error {
	if log.ID == "" {
		return errors.New("deployment log id is required")
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getForUpdate(ctx, tx, runID)
		if err != nil {
			return err
		}
		var next int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence), 0) + 1 FROM runtime_deployment_logs WHERE deployment_run_id = $1`, runID).Scan(&next); err != nil {
			return err
		}
		log.DeploymentRunID = runID
		log.Sequence = next
		record.Logs = append(record.Logs, log)
		if err := s.insertLog(ctx, tx, log); err != nil {
			return err
		}
		return s.saveRecord(ctx, tx, record)
	})
}

func (s *DeploymentStore) Logs(ctx context.Context, runID string) ([]event.LogChunk, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, deployment_run_id, stream, sequence, content, created_at FROM runtime_deployment_logs WHERE deployment_run_id = $1 ORDER BY sequence, created_at, id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []event.LogChunk
	for rows.Next() {
		var log event.LogChunk
		if err := rows.Scan(&log.ID, &log.DeploymentRunID, &log.Stream, &log.Sequence, &log.Content, &log.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

func (s *DeploymentStore) AppendEvent(ctx context.Context, runID string, evt event.Event) error {
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

func (s *DeploymentStore) Events(ctx context.Context, runID string) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM runtime_deployment_events WHERE deployment_run_id = $1 ORDER BY created_at, id`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

func (s *DeploymentStore) AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error {
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
	return AppendHashChainedAudit(ctx, s.pool, "deployment", entry)
}

func (s *DeploymentStore) Audits(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM runtime_deployment_audit_logs WHERE subject = $1 ORDER BY created_at, id`, subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAudits(rows)
}

func (s *DeploymentStore) ListNonTerminalDeploymentRuns(ctx context.Context, limit int) ([]deploymentusecase.RunRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_deployment_runs WHERE status NOT IN ('Succeeded', 'Failed', 'Canceled', 'RolledBack') ORDER BY created_at, id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDeploymentRecords(rows)
}

func (s *DeploymentStore) ListStaleDeploymentRuns(ctx context.Context, olderThan time.Time, limit int) ([]deploymentusecase.RunRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_deployment_runs WHERE status NOT IN ('Succeeded', 'Failed', 'Canceled', 'RolledBack') AND (updated_at < $1 OR lease_expires_at < $1) ORDER BY updated_at, id LIMIT $2`, olderThan, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDeploymentRecords(rows)
}

func (s *DeploymentStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
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

func (s *DeploymentStore) getForUpdate(ctx context.Context, tx pgx.Tx, id string) (deploymentusecase.RunRecord, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT record FROM runtime_deployment_runs WHERE id = $1 FOR UPDATE`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return deploymentusecase.RunRecord{}, deploymentusecase.ErrRunNotFound
	}
	if err != nil {
		return deploymentusecase.RunRecord{}, err
	}
	return decodeDeploymentRecord(raw)
}

func (s *DeploymentStore) saveRecord(ctx context.Context, tx pgx.Tx, record deploymentusecase.RunRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_deployment_runs (id, release_id, application_id, environment_id, release_target_id, target_type, status, reason, correlation_id, owner_id, lease_expires_at, attempt, heartbeat_at, manifest_snapshot_ref, record, created_at, updated_at, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		ON CONFLICT (id) DO UPDATE SET release_id = EXCLUDED.release_id, application_id = EXCLUDED.application_id, environment_id = EXCLUDED.environment_id, release_target_id = EXCLUDED.release_target_id, target_type = EXCLUDED.target_type, status = EXCLUDED.status, reason = EXCLUDED.reason, correlation_id = EXCLUDED.correlation_id, owner_id = EXCLUDED.owner_id, lease_expires_at = EXCLUDED.lease_expires_at, attempt = EXCLUDED.attempt, heartbeat_at = EXCLUDED.heartbeat_at, manifest_snapshot_ref = EXCLUDED.manifest_snapshot_ref, record = EXCLUDED.record, updated_at = EXCLUDED.updated_at, started_at = EXCLUDED.started_at, finished_at = EXCLUDED.finished_at, version = runtime_deployment_runs.version + 1`,
		record.Run.ID, record.Run.ReleaseID, record.Run.ApplicationID, record.Run.EnvironmentID, record.Run.ReleaseTargetID, record.Run.TargetType, string(record.Run.Status), record.Run.Reason, record.Run.CorrelationID, record.Run.OwnerID, record.Run.LeaseExpiresAt, record.Run.Attempt, record.Run.HeartbeatAt, record.Run.ManifestSnapshotRef, raw, record.Run.CreatedAt, record.Run.UpdatedAt, record.Run.StartedAt, record.Run.FinishedAt)
	if err != nil {
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
	if err := s.replaceResources(ctx, tx, record); err != nil {
		return err
	}
	if !record.Snapshot.CreatedAt.IsZero() || record.Snapshot.ID != "" {
		if err := s.saveSnapshot(ctx, tx, record.Snapshot); err != nil {
			return err
		}
	}
	if !record.RollbackPlan.CreatedAt.IsZero() || record.RollbackPlan.CurrentSnapshotID != "" {
		if err := s.saveRollbackPlan(ctx, tx, record.RollbackPlan); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeploymentStore) insertLog(ctx context.Context, tx pgx.Tx, log event.LogChunk) error {
	_, err := tx.Exec(ctx, `INSERT INTO runtime_deployment_logs (id, deployment_run_id, stream, sequence, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (id) DO NOTHING`,
		log.ID, log.DeploymentRunID, log.Stream, log.Sequence, log.Content, log.CreatedAt)
	return err
}

func (s *DeploymentStore) insertEvent(ctx context.Context, tx pgx.Tx, runID string, evt event.Event) error {
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_deployment_events (id, deployment_run_id, event_type, source, subject, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO NOTHING`,
		evt.ID, runID, evt.Type, evt.Source, evt.Subject, raw, evt.Time)
	return err
}

func (s *DeploymentStore) insertAudit(ctx context.Context, tx pgx.Tx, runID string, entry audit.AuditLog) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_deployment_audit_logs (id, deployment_run_id, org_id, actor_id, action, subject, correlation_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (id) DO NOTHING`,
		entry.ID, runID, entry.OrgID, entry.ActorID, entry.Action, entry.Subject, entry.CorrelationID, raw, entry.CreatedAt)
	return err
}

func (s *DeploymentStore) replaceResources(ctx context.Context, tx pgx.Tx, record deploymentusecase.RunRecord) error {
	if _, err := tx.Exec(ctx, `DELETE FROM runtime_deployment_resources WHERE deployment_run_id = $1`, record.Run.ID); err != nil {
		return err
	}
	resources := map[string][]deploymentusecase.ManifestResourceSummary{
		"plan":    record.Plan.Resources,
		"desired": record.Inventory.Desired,
		"applied": record.Inventory.Applied,
	}
	for inventoryType, items := range resources {
		for index, resource := range items {
			raw, err := json.Marshal(resource)
			if err != nil {
				return err
			}
			key := resourceKey(resource, index)
			_, err = tx.Exec(ctx, `INSERT INTO runtime_deployment_resources (deployment_run_id, inventory_type, resource_key, api_version, kind, namespace, name, desired_hash, health, payload, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
				record.Run.ID, inventoryType, key, resource.APIVersion, resource.Kind, resource.Namespace, resource.Name, resource.DesiredHash, string(resource.Health), raw, record.Run.CreatedAt, record.Run.UpdatedAt)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *DeploymentStore) saveSnapshot(ctx context.Context, tx pgx.Tx, snapshot deploymentusecase.ManifestSnapshot) error {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_manifest_snapshots (id, deployment_run_id, content_hash, document_count, resource_count, storage_ref, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET content_hash = EXCLUDED.content_hash, document_count = EXCLUDED.document_count, resource_count = EXCLUDED.resource_count, storage_ref = EXCLUDED.storage_ref, payload = EXCLUDED.payload`,
		snapshot.ID, snapshot.DeploymentRunID, snapshot.ContentHash, snapshot.DocumentCount, snapshot.ResourceCount, snapshot.StorageRef, raw, snapshot.CreatedAt)
	return err
}

func (s *DeploymentStore) saveRollbackPlan(ctx context.Context, tx pgx.Tx, plan deploymentusecase.RollbackPlan) error {
	raw, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_rollback_plans (deployment_run_id, current_snapshot_id, previous_snapshot_id, target_type, target_name, strategy, executable, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (deployment_run_id) DO UPDATE SET current_snapshot_id = EXCLUDED.current_snapshot_id, previous_snapshot_id = EXCLUDED.previous_snapshot_id, target_type = EXCLUDED.target_type, target_name = EXCLUDED.target_name, strategy = EXCLUDED.strategy, executable = EXCLUDED.executable, payload = EXCLUDED.payload, created_at = EXCLUDED.created_at`,
		plan.DeploymentRunID, plan.CurrentSnapshotID, plan.PreviousSnapshotID, plan.TargetType, plan.TargetName, plan.Strategy, plan.Executable, raw, plan.CreatedAt)
	return err
}

func scanDeploymentRecords(rows pgx.Rows) ([]deploymentusecase.RunRecord, error) {
	var records []deploymentusecase.RunRecord
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		record, err := decodeDeploymentRecord(raw)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func decodeDeploymentRecord(raw []byte) (deploymentusecase.RunRecord, error) {
	var record deploymentusecase.RunRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return deploymentusecase.RunRecord{}, err
	}
	return record, nil
}

func resourceKey(resource deploymentusecase.ManifestResourceSummary, index int) string {
	return fmt.Sprintf("%s/%s/%s/%s/%d", resource.APIVersion, resource.Kind, resource.Namespace, resource.Name, index)
}

func scanEvents(rows pgx.Rows) ([]event.Event, error) {
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
	sort.Slice(events, func(i, j int) bool { return events[i].Time.Before(events[j].Time) })
	return events, rows.Err()
}

func scanAudits(rows pgx.Rows) ([]audit.AuditLog, error) {
	var entries []audit.AuditLog
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var entry audit.AuditLog
		if err := json.Unmarshal(raw, &entry); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].CreatedAt.Before(entries[j].CreatedAt) })
	return entries, rows.Err()
}
