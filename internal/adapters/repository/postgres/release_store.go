package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
)

type ReleaseStore struct {
	pool *pgxpool.Pool
}

var _ artifactusecase.Store = (*ReleaseStore)(nil)

func NewReleaseStore(pool *pgxpool.Pool) *ReleaseStore {
	return &ReleaseStore{pool: pool}
}

func (s *ReleaseStore) SaveRelease(ctx context.Context, record artifactusecase.ReleaseRecord) error {
	if record.Release.ID == "" {
		return errors.New("release id is required")
	}
	return s.withTx(ctx, func(tx pgx.Tx) error {
		return s.saveRelease(ctx, tx, record)
	})
}

func (s *ReleaseStore) GetRelease(ctx context.Context, id string) (artifactusecase.ReleaseRecord, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT record FROM runtime_releases WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return artifactusecase.ReleaseRecord{}, artifactusecase.ErrReleaseNotFound
	}
	if err != nil {
		return artifactusecase.ReleaseRecord{}, err
	}
	return decodeReleaseRecord(raw)
}

func (s *ReleaseStore) ListReleases(ctx context.Context) ([]artifactusecase.ReleaseRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT record FROM runtime_releases ORDER BY created_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []artifactusecase.ReleaseRecord
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		record, err := decodeReleaseRecord(raw)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *ReleaseStore) AppendEvent(ctx context.Context, subject string, evt event.Event) error {
	return s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getReleaseForUpdate(ctx, tx, subject)
		if err != nil {
			return err
		}
		record.Events = upsertEvent(record.Events, evt)
		if err := s.insertReleaseEvent(ctx, tx, subject, evt); err != nil {
			return err
		}
		return s.saveRelease(ctx, tx, record)
	})
}

func (s *ReleaseStore) AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error {
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		record, err := s.getReleaseForUpdate(ctx, tx, subject)
		if err != nil {
			return err
		}
		record.Audits = upsertAudit(record.Audits, entry)
		if err := s.insertReleaseAudit(ctx, tx, subject, entry); err != nil {
			return err
		}
		return s.saveRelease(ctx, tx, record)
	})
	if err != nil {
		return err
	}
	return AppendHashChainedAudit(ctx, s.pool, "release", entry)
}

func (s *ReleaseStore) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
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

func (s *ReleaseStore) getReleaseForUpdate(ctx context.Context, tx pgx.Tx, id string) (artifactusecase.ReleaseRecord, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT record FROM runtime_releases WHERE id = $1 FOR UPDATE`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return artifactusecase.ReleaseRecord{}, artifactusecase.ErrReleaseNotFound
	}
	if err != nil {
		return artifactusecase.ReleaseRecord{}, err
	}
	return decodeReleaseRecord(raw)
}

func (s *ReleaseStore) saveRelease(ctx context.Context, tx pgx.Tx, record artifactusecase.ReleaseRecord) error {
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_releases (id, name, version_name, application_id, environment_id, status, record, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, version_name = EXCLUDED.version_name, application_id = EXCLUDED.application_id, environment_id = EXCLUDED.environment_id, status = EXCLUDED.status, record = EXCLUDED.record, updated_at = EXCLUDED.updated_at, version = runtime_releases.version + 1`,
		record.Release.ID, record.Release.Name, record.Release.Version, record.Release.ApplicationID, record.Release.EnvironmentID, record.Release.Status, raw, record.Release.CreatedAt, record.Release.UpdatedAt)
	if err != nil {
		return err
	}
	for _, binding := range record.Bindings {
		bindingRaw, err := json.Marshal(binding)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO runtime_release_artifacts (id, release_id, artifact_id, name, artifact_type, reference, digest, digest_reference, payload, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET artifact_id = EXCLUDED.artifact_id, name = EXCLUDED.name, artifact_type = EXCLUDED.artifact_type, reference = EXCLUDED.reference, digest = EXCLUDED.digest, digest_reference = EXCLUDED.digest_reference, payload = EXCLUDED.payload, updated_at = EXCLUDED.updated_at`,
			binding.ID, binding.ReleaseID, binding.ArtifactID, binding.Name, binding.Type, binding.Reference, binding.Digest, binding.DigestReference, bindingRaw, binding.CreatedAt, binding.UpdatedAt)
		if err != nil {
			return err
		}
	}
	for _, evt := range record.Events {
		if err := s.insertReleaseEvent(ctx, tx, record.Release.ID, evt); err != nil {
			return err
		}
	}
	for _, entry := range record.Audits {
		if err := s.insertReleaseAudit(ctx, tx, record.Release.ID, entry); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReleaseStore) insertReleaseEvent(ctx context.Context, tx pgx.Tx, releaseID string, evt event.Event) error {
	raw, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_release_events (id, release_id, event_type, source, subject, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (id) DO NOTHING`,
		evt.ID, releaseID, evt.Type, evt.Source, evt.Subject, raw, evt.Time)
	return err
}

func (s *ReleaseStore) insertReleaseAudit(ctx context.Context, tx pgx.Tx, releaseID string, entry audit.AuditLog) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO runtime_release_audit_logs (id, release_id, org_id, actor_id, action, subject, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (id) DO NOTHING`,
		entry.ID, releaseID, entry.OrgID, entry.ActorID, entry.Action, entry.Subject, raw, entry.CreatedAt)
	return err
}

func decodeReleaseRecord(raw []byte) (artifactusecase.ReleaseRecord, error) {
	var record artifactusecase.ReleaseRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return artifactusecase.ReleaseRecord{}, err
	}
	return record, nil
}
