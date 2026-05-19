package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domaincompliance "github.com/sevoniva/nivora/internal/domain/compliance"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
)

type ComplianceStore struct {
	pool *pgxpool.Pool
}

var _ complianceusecase.Store = (*ComplianceStore)(nil)

func NewComplianceStore(pool *pgxpool.Pool) *ComplianceStore {
	return &ComplianceStore{pool: pool}
}

func (s *ComplianceStore) SaveEvidenceBundle(ctx context.Context, bundle domaincompliance.EvidenceBundle) error {
	raw, err := json.Marshal(bundle)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO compliance_evidence_bundles
		(id, subject_type, subject_id, scope_type, scope_id, summary, payload, generated_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		ON CONFLICT (id) DO UPDATE SET
			subject_type = EXCLUDED.subject_type,
			subject_id = EXCLUDED.subject_id,
			scope_type = EXCLUDED.scope_type,
			scope_id = EXCLUDED.scope_id,
			summary = EXCLUDED.summary,
			payload = EXCLUDED.payload,
			generated_at = EXCLUDED.generated_at,
			updated_at = now()`,
		bundle.ID, bundle.SubjectType, bundle.SubjectID, bundle.ScopeType, bundle.ScopeID, bundle.Summary, raw, bundle.GeneratedAt)
	return err
}

func (s *ComplianceStore) GetEvidenceBundle(ctx context.Context, id string) (domaincompliance.EvidenceBundle, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM compliance_evidence_bundles WHERE id = $1`, id).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincompliance.EvidenceBundle{}, complianceusecase.ErrEvidenceBundleNotFound
	}
	if err != nil {
		return domaincompliance.EvidenceBundle{}, err
	}
	return decodeEvidenceBundle(raw)
}

func (s *ComplianceStore) SearchEvidenceBundles(ctx context.Context, subjectType string, subjectID string) ([]domaincompliance.EvidenceBundle, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM compliance_evidence_bundles
		WHERE ($1 = '' OR subject_type = $1)
		  AND ($2 = '' OR subject_id = $2)
		ORDER BY generated_at DESC, id`, subjectType, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bundles []domaincompliance.EvidenceBundle
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		bundle, err := decodeEvidenceBundle(raw)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}
	return bundles, rows.Err()
}

func (s *ComplianceStore) SaveRetentionPolicy(ctx context.Context, policy domaincompliance.RetentionPolicy) error {
	raw, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO compliance_retention_policies
		(id, scope_type, scope_id, log_days, audit_days, event_days, evidence_days, immutable_audit, payload, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (scope_type, scope_id) DO UPDATE SET
			id = EXCLUDED.id,
			log_days = EXCLUDED.log_days,
			audit_days = EXCLUDED.audit_days,
			event_days = EXCLUDED.event_days,
			evidence_days = EXCLUDED.evidence_days,
			immutable_audit = EXCLUDED.immutable_audit,
			payload = EXCLUDED.payload,
			updated_at = EXCLUDED.updated_at`,
		policy.ID, policy.ScopeType, policy.ScopeID, policy.LogDays, policy.AuditDays, policy.EventDays, policy.EvidenceDays, policy.ImmutableAudit, raw, policy.UpdatedAt)
	return err
}

func (s *ComplianceStore) GetRetentionPolicy(ctx context.Context, scopeType string, scopeID string) (domaincompliance.RetentionPolicy, error) {
	var raw []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM compliance_retention_policies WHERE scope_type = $1 AND scope_id = $2`, scopeType, scopeID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincompliance.RetentionPolicy{}, complianceusecase.ErrRetentionPolicyNotFound
	}
	if err != nil {
		return domaincompliance.RetentionPolicy{}, err
	}
	var policy domaincompliance.RetentionPolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return domaincompliance.RetentionPolicy{}, err
	}
	return policy, nil
}

func decodeEvidenceBundle(raw []byte) (domaincompliance.EvidenceBundle, error) {
	var bundle domaincompliance.EvidenceBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return domaincompliance.EvidenceBundle{}, err
	}
	return bundle, nil
}
