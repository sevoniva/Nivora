package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
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

func (s *ComplianceStore) AppendAuditLog(ctx context.Context, entry audit.AuditLog) error {
	return AppendHashChainedAudit(ctx, s.pool, "mcp", entry)
}

func (s *ComplianceStore) SearchAuditLogs(ctx context.Context, input complianceusecase.AuditSearchInput) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_audit_logs
		WHERE source = 'mcp'
		  AND ($1 = '' OR actor_id = $1)
		  AND ($2 = '' OR action ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR subject ILIKE '%' || $3 || '%')
		ORDER BY created_at ASC, id ASC`, input.ActorID, input.Action, input.Subject)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
		if input.ScopeType != "" && entry.ScopeType != input.ScopeType {
			continue
		}
		if input.ScopeID != "" && entry.ScopeID != input.ScopeID {
			continue
		}
		if input.SubjectType != "" && entry.SubjectType != input.SubjectType {
			continue
		}
		if input.SubjectID != "" && entry.SubjectID != input.SubjectID {
			continue
		}
		if input.RequestID != "" && entry.RequestID != input.RequestID {
			continue
		}
		if input.CorrelationID != "" && entry.CorrelationID != input.CorrelationID {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
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

func (s *ComplianceStore) PreviewRetention(ctx context.Context, policy domaincompliance.RetentionPolicy, now time.Time) ([]domaincompliance.RetentionTargetResult, error) {
	return s.retentionTargets(ctx, policy, now, false)
}

func (s *ComplianceStore) ApplyRetention(ctx context.Context, policy domaincompliance.RetentionPolicy, now time.Time) ([]domaincompliance.RetentionTargetResult, error) {
	return s.retentionTargets(ctx, policy, now, true)
}

func (s *ComplianceStore) retentionTargets(ctx context.Context, policy domaincompliance.RetentionPolicy, now time.Time, apply bool) ([]domaincompliance.RetentionTargetResult, error) {
	logs := postgresUnsupportedRetentionTarget(domaincompliance.RetentionTargetLogs, policy.LogDays, now, "log retention spans runtime stores and is preview-only in this foundation")
	audit := postgresSupportedRetentionTarget(domaincompliance.RetentionTargetAudit, policy.AuditDays, now)
	audit.Immutable = true
	audit.Warnings = append(audit.Warnings, "audit records are immutable in this foundation; retention reports candidates but does not delete them")
	if !audit.Cutoff.IsZero() {
		count, err := s.countExpiredAudit(ctx, policy, audit.Cutoff)
		if err != nil {
			return nil, err
		}
		audit.Candidates = count
	}
	events := postgresUnsupportedRetentionTarget(domaincompliance.RetentionTargetEvents, policy.EventDays, now, "event retention spans runtime stores and is preview-only in this foundation")
	evidence := postgresSupportedRetentionTarget(domaincompliance.RetentionTargetEvidence, policy.EvidenceDays, now)
	if !evidence.Cutoff.IsZero() {
		count, err := s.countExpiredEvidence(ctx, policy, evidence.Cutoff)
		if err != nil {
			return nil, err
		}
		evidence.Candidates = count
		if apply && count > 0 {
			deleted, err := s.deleteExpiredEvidence(ctx, policy, evidence.Cutoff)
			if err != nil {
				return nil, err
			}
			evidence.Deleted = deleted
		}
	}
	return []domaincompliance.RetentionTargetResult{logs, audit, events, evidence}, nil
}

func (s *ComplianceStore) countExpiredEvidence(ctx context.Context, policy domaincompliance.RetentionPolicy, cutoff time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM compliance_evidence_bundles
		WHERE ($1 = '' OR $1 = 'global' OR scope_type = $1)
		  AND ($2 = '' OR scope_id = $2)
		  AND generated_at < $3`, policy.ScopeType, policy.ScopeID, cutoff).Scan(&count)
	return count, err
}

func (s *ComplianceStore) deleteExpiredEvidence(ctx context.Context, policy domaincompliance.RetentionPolicy, cutoff time.Time) (int, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM compliance_evidence_bundles
		WHERE ($1 = '' OR $1 = 'global' OR scope_type = $1)
		  AND ($2 = '' OR scope_id = $2)
		  AND generated_at < $3`, policy.ScopeType, policy.ScopeID, cutoff)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (s *ComplianceStore) countExpiredAudit(ctx context.Context, policy domaincompliance.RetentionPolicy, cutoff time.Time) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM compliance_audit_records
		WHERE ($1 = '' OR $1 = 'global' OR scope_type = $1)
		  AND ($2 = '' OR scope_id = $2)
		  AND created_at < $3`, policy.ScopeType, policy.ScopeID, cutoff).Scan(&count)
	return count, err
}

type AuditRecord struct {
	ID            string    `json:"id"`
	ActorID       string    `json:"actor_id"`
	Action        string    `json:"action"`
	SubjectType   string    `json:"subject_type"`
	SubjectID     string    `json:"subject_id"`
	Subject       string    `json:"subject"`
	ScopeType     string    `json:"scope_type"`
	ScopeID       string    `json:"scope_id"`
	CorrelationID string    `json:"correlation_id"`
	RequestID     string    `json:"request_id"`
	PreviousHash  string    `json:"previous_hash"`
	RecordHash    string    `json:"record_hash"`
	Payload       []byte    `json:"payload"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *ComplianceStore) AppendAuditRecord(ctx context.Context, record AuditRecord) error {
	if len(record.Payload) == 0 {
		record.Payload = []byte("{}")
	}
	prevHash, err := s.latestAuditHash(ctx, record.ScopeType, record.ScopeID)
	if err != nil {
		return err
	}
	record.PreviousHash = prevHash

	canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
		prevHash, record.ActorID, record.Action, record.SubjectType, record.SubjectID, record.ScopeType, record.ScopeID, record.CreatedAt.UTC().Format(time.RFC3339Nano))
	hash := sha256.Sum256([]byte(canonical))
	record.RecordHash = hex.EncodeToString(hash[:])

	_, err = s.pool.Exec(ctx, `INSERT INTO compliance_audit_records
		(id, actor_id, action, subject_type, subject_id, subject, scope_type, scope_id, correlation_id, request_id, previous_hash, record_hash, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		record.ID, record.ActorID, record.Action, record.SubjectType, record.SubjectID, record.Subject,
		record.ScopeType, record.ScopeID, record.CorrelationID, record.RequestID,
		record.PreviousHash, record.RecordHash, record.Payload, record.CreatedAt)
	return err
}

func (s *ComplianceStore) VerifyAuditChain(ctx context.Context, scopeType, scopeID string) (valid bool, firstBroken string, err error) {
	rows, err := s.pool.Query(ctx, `SELECT id, actor_id, action, subject_type, subject_id, subject, scope_type, scope_id, previous_hash, record_hash, payload, created_at
		FROM compliance_audit_records WHERE scope_type=$1 AND ($2='' OR scope_id=$2) ORDER BY created_at, id`, scopeType, scopeID)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()

	var records []AuditRecord
	for rows.Next() {
		var r AuditRecord
		if err := rows.Scan(&r.ID, &r.ActorID, &r.Action, &r.SubjectType, &r.SubjectID, &r.Subject, &r.ScopeType, &r.ScopeID, &r.PreviousHash, &r.RecordHash, &r.Payload, &r.CreatedAt); err != nil {
			return false, "", err
		}
		records = append(records, r)
	}
	if rows.Err() != nil {
		return false, "", rows.Err()
	}

	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].ID < records[j].ID
		}
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})

	var expectedPrev string
	for i, r := range records {
		if i == 0 {
			expectedPrev = ""
		}
		if r.PreviousHash != expectedPrev {
			return false, r.ID, nil
		}
		canonical := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s",
			r.PreviousHash, r.ActorID, r.Action, r.SubjectType, r.SubjectID, r.ScopeType, r.ScopeID, r.CreatedAt.UTC().Format(time.RFC3339Nano))
		hash := sha256.Sum256([]byte(canonical))
		expectedHash := hex.EncodeToString(hash[:])
		if r.RecordHash != expectedHash {
			return false, r.ID, nil
		}
		expectedPrev = r.RecordHash
	}
	return true, "", nil
}

func (s *ComplianceStore) latestAuditHash(ctx context.Context, scopeType, scopeID string) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT record_hash FROM compliance_audit_records WHERE scope_type=$1 AND ($2='' OR scope_id=$2) ORDER BY created_at DESC, id DESC LIMIT 1`, scopeType, scopeID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return hash, err
}

func decodeEvidenceBundle(raw []byte) (domaincompliance.EvidenceBundle, error) {
	var bundle domaincompliance.EvidenceBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return domaincompliance.EvidenceBundle{}, err
	}
	return bundle, nil
}

func postgresRetentionCutoff(days int, now time.Time) time.Time {
	if days <= 0 {
		return time.Time{}
	}
	return now.AddDate(0, 0, -days)
}

func postgresSupportedRetentionTarget(target string, days int, now time.Time) domaincompliance.RetentionTargetResult {
	result := domaincompliance.RetentionTargetResult{Target: target, Supported: true, RetentionDays: days, Cutoff: postgresRetentionCutoff(days, now)}
	if days <= 0 {
		result.Warnings = append(result.Warnings, "retention is disabled for this target because retentionDays is zero")
	}
	return result
}

func postgresUnsupportedRetentionTarget(target string, days int, now time.Time, warning string) domaincompliance.RetentionTargetResult {
	result := postgresSupportedRetentionTarget(target, days, now)
	result.Supported = false
	result.Warnings = append(result.Warnings, warning)
	return result
}
