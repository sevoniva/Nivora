package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

type SecurityStore struct {
	pool *pgxpool.Pool
}

var _ securityusecase.Store = (*SecurityStore)(nil)

func NewSecurityStore(pool *pgxpool.Pool) *SecurityStore {
	return &SecurityStore{pool: pool}
}

func (s *SecurityStore) Save(ctx context.Context, record securityusecase.ScanRecord) error {
	if record.Scan.ID == "" {
		return errors.New("scan id is required")
	}
	findingsJSON, _ := json.Marshal(record.Scan.Findings)
	policyFindingsJSON, _ := json.Marshal(record.Policy.Findings)
	warningsJSON, _ := json.Marshal(record.Warnings)
	_, err := s.pool.Exec(ctx, `INSERT INTO security_scans (id, subject_type, subject_id, scanner, status, summary_total, summary_low, summary_medium, summary_high, summary_critical, findings, policy_decision, policy_reason, policy_findings, signature_subject, signature_status, signature_result, signature_key_ref, signature_identity, signature_issuer, sbom_format, sbom_storage_ref, sbom_digest, warnings, started_at, finished_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
		ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status, summary_total=EXCLUDED.summary_total, summary_low=EXCLUDED.summary_low, summary_medium=EXCLUDED.summary_medium, summary_high=EXCLUDED.summary_high, summary_critical=EXCLUDED.summary_critical, findings=EXCLUDED.findings, policy_decision=EXCLUDED.policy_decision, policy_reason=EXCLUDED.policy_reason, policy_findings=EXCLUDED.policy_findings, finished_at=EXCLUDED.finished_at`,
		record.Scan.ID, record.Scan.SubjectType, record.Scan.SubjectID, record.Scan.Scanner, record.Scan.Status, record.Scan.Summary.Total, record.Scan.Summary.Low, record.Scan.Summary.Medium, record.Scan.Summary.High, record.Scan.Summary.Critical, findingsJSON, record.Policy.Decision, record.Policy.Reason, policyFindingsJSON, record.Signature.Subject, record.Signature.Status, record.Signature.Result, record.Signature.KeyRef, record.Signature.CertificateIdentity, record.Signature.Issuer, record.SBOM.Format, record.SBOM.StorageRef, record.SBOM.Digest, warningsJSON, record.Scan.StartedAt, record.Scan.FinishedAt, record.Scan.CreatedAt)
	return err
}

func (s *SecurityStore) Get(ctx context.Context, id string) (securityusecase.ScanRecord, error) {
	var rec securityusecase.ScanRecord
	var findingsJSON, policyFindingsJSON, warningsJSON []byte
	err := s.pool.QueryRow(ctx, `SELECT id, subject_type, subject_id, scanner, status, summary_total, summary_low, summary_medium, summary_high, summary_critical, findings, policy_decision, policy_reason, policy_findings, signature_subject, signature_status, signature_result, signature_key_ref, signature_identity, signature_issuer, sbom_format, sbom_storage_ref, sbom_digest, warnings, started_at, finished_at, created_at FROM security_scans WHERE id=$1`, id).
		Scan(&rec.Scan.ID, &rec.Scan.SubjectType, &rec.Scan.SubjectID, &rec.Scan.Scanner, &rec.Scan.Status, &rec.Scan.Summary.Total, &rec.Scan.Summary.Low, &rec.Scan.Summary.Medium, &rec.Scan.Summary.High, &rec.Scan.Summary.Critical, &findingsJSON, &rec.Policy.Decision, &rec.Policy.Reason, &policyFindingsJSON, &rec.Signature.Subject, &rec.Signature.Status, &rec.Signature.Result, &rec.Signature.KeyRef, &rec.Signature.CertificateIdentity, &rec.Signature.Issuer, &rec.SBOM.Format, &rec.SBOM.StorageRef, &rec.SBOM.Digest, &warningsJSON, &rec.Scan.StartedAt, &rec.Scan.FinishedAt, &rec.Scan.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return securityusecase.ScanRecord{}, securityusecase.ErrScanNotFound
	}
	if err != nil {
		return securityusecase.ScanRecord{}, err
	}
	json.Unmarshal(findingsJSON, &rec.Scan.Findings)
	json.Unmarshal(policyFindingsJSON, &rec.Policy.Findings)
	json.Unmarshal(warningsJSON, &rec.Warnings)
	return rec, nil
}

func (s *SecurityStore) List(ctx context.Context) ([]securityusecase.ScanRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, subject_type, subject_id, scanner, status, summary_total, summary_low, summary_medium, summary_high, summary_critical, findings, policy_decision, policy_reason, policy_findings, signature_subject, signature_status, signature_result, signature_key_ref, signature_identity, signature_issuer, sbom_format, sbom_storage_ref, sbom_digest, warnings, started_at, finished_at, created_at FROM security_scans ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []securityusecase.ScanRecord
	for rows.Next() {
		var rec securityusecase.ScanRecord
		var findingsJSON, policyFindingsJSON, warningsJSON []byte
		if err := rows.Scan(&rec.Scan.ID, &rec.Scan.SubjectType, &rec.Scan.SubjectID, &rec.Scan.Scanner, &rec.Scan.Status, &rec.Scan.Summary.Total, &rec.Scan.Summary.Low, &rec.Scan.Summary.Medium, &rec.Scan.Summary.High, &rec.Scan.Summary.Critical, &findingsJSON, &rec.Policy.Decision, &rec.Policy.Reason, &policyFindingsJSON, &rec.Signature.Subject, &rec.Signature.Status, &rec.Signature.Result, &rec.Signature.KeyRef, &rec.Signature.CertificateIdentity, &rec.Signature.Issuer, &rec.SBOM.Format, &rec.SBOM.StorageRef, &rec.SBOM.Digest, &warningsJSON, &rec.Scan.StartedAt, &rec.Scan.FinishedAt, &rec.Scan.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(findingsJSON, &rec.Scan.Findings)
		json.Unmarshal(policyFindingsJSON, &rec.Policy.Findings)
		json.Unmarshal(warningsJSON, &rec.Warnings)
		out = append(out, rec)
	}
	if out == nil {
		out = []securityusecase.ScanRecord{}
	}
	return out, rows.Err()
}

func (s *SecurityStore) AppendEvent(ctx context.Context, scanID string, evt event.Event) error {
	payload, _ := json.Marshal(evt)
	_, err := s.pool.Exec(ctx, `INSERT INTO governance_event_logs (id, source, event_type, subject, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		evt.ID, "security", evt.Type, scanID, payload, evt.Time)
	return err
}

func (s *SecurityStore) Events(ctx context.Context, scanID string) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_event_logs WHERE source='security' AND subject=$1 ORDER BY created_at`, scanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []event.Event
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var evt event.Event
		json.Unmarshal(raw, &evt)
		out = append(out, evt)
	}
	if out == nil {
		out = []event.Event{}
	}
	return out, rows.Err()
}

func (s *SecurityStore) AppendAudit(ctx context.Context, scanID string, entry audit.AuditLog) error {
	payload, _ := json.Marshal(entry)
	_, err := s.pool.Exec(ctx, `INSERT INTO governance_audit_logs (id, source, actor_id, action, subject, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		entry.ID, "security", entry.ActorID, entry.Action, scanID, payload, entry.CreatedAt)
	return err
}
