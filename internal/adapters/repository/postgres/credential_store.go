package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	"github.com/sevoniva/nivora/internal/domain/event"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
)

type CredentialStore struct {
	pool *pgxpool.Pool
}

var _ credentialusecase.Store = (*CredentialStore)(nil)

func NewCredentialStore(pool *pgxpool.Pool) *CredentialStore {
	return &CredentialStore{pool: pool}
}

func (s *CredentialStore) SaveCredential(ctx context.Context, cred domaincredential.Credential) error {
	if cred.ID == "" {
		return errors.New("credential id is required")
	}
	metadata, _ := json.Marshal(cred.Metadata)
	_, err := s.pool.Exec(ctx, `INSERT INTO credential_records (id, name, credential_type, scope_type, scope_id, provider, secret_key, secret_ref_id, secret_version, metadata, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, credential_type=EXCLUDED.credential_type, scope_type=EXCLUDED.scope_type, scope_id=EXCLUDED.scope_id, provider=EXCLUDED.provider, secret_key=EXCLUDED.secret_key, secret_ref_id=EXCLUDED.secret_ref_id, secret_version=EXCLUDED.secret_version, metadata=EXCLUDED.metadata, status=EXCLUDED.status, updated_at=EXCLUDED.updated_at`,
		cred.ID, cred.Name, cred.Type, cred.ScopeType, cred.ScopeID, cred.SecretRef.Provider, cred.SecretRef.Key, cred.SecretRef.ID, cred.SecretRef.Version, metadata, cred.Status, cred.CreatedAt, cred.UpdatedAt)
	return err
}

func (s *CredentialStore) GetCredential(ctx context.Context, id string) (domaincredential.Credential, error) {
	var c domaincredential.Credential
	var rawMetadata []byte
	err := s.pool.QueryRow(ctx, `SELECT id, name, credential_type, scope_type, scope_id, provider, secret_key, secret_ref_id, secret_version, metadata, status, created_at, updated_at FROM credential_records WHERE id=$1`, id).
		Scan(&c.ID, &c.Name, &c.Type, &c.ScopeType, &c.ScopeID, &c.SecretRef.Provider, &c.SecretRef.Key, &c.SecretRef.ID, &c.SecretRef.Version, &rawMetadata, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domaincredential.Credential{}, credentialusecase.ErrCredentialNotFound
	}
	if err != nil {
		return domaincredential.Credential{}, err
	}
	json.Unmarshal(rawMetadata, &c.Metadata)
	c.SecretRef.Name = c.Name
	c.SecretRef.ScopeType = c.ScopeType
	c.SecretRef.ScopeID = c.ScopeID
	return c, nil
}

func (s *CredentialStore) ListCredentials(ctx context.Context) ([]domaincredential.Credential, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, credential_type, scope_type, scope_id, provider, secret_key, secret_ref_id, secret_version, metadata, status, created_at, updated_at FROM credential_records ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domaincredential.Credential
	for rows.Next() {
		var c domaincredential.Credential
		var rawMetadata []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.ScopeType, &c.ScopeID, &c.SecretRef.Provider, &c.SecretRef.Key, &c.SecretRef.ID, &c.SecretRef.Version, &rawMetadata, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(rawMetadata, &c.Metadata)
		c.SecretRef.Name = c.Name
		c.SecretRef.ScopeType = c.ScopeType
		c.SecretRef.ScopeID = c.ScopeID
		out = append(out, c)
	}
	if out == nil {
		out = []domaincredential.Credential{}
	}
	return out, rows.Err()
}

func (s *CredentialStore) DeleteCredential(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM credential_records WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return credentialusecase.ErrCredentialNotFound
	}
	return nil
}

func (s *CredentialStore) AppendEvent(ctx context.Context, evt event.Event) error {
	payload, _ := json.Marshal(evt)
	_, err := s.pool.Exec(ctx, `INSERT INTO governance_event_logs (id, source, event_type, subject, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		evt.ID, "credential", evt.Type, evt.Subject, payload, evt.Time)
	return err
}

func (s *CredentialStore) Events(ctx context.Context) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_event_logs WHERE source='credential' ORDER BY created_at`)
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

func (s *CredentialStore) AppendAudit(ctx context.Context, entry audit.AuditLog) error {
	return AppendHashChainedAudit(ctx, s.pool, "credential", entry)
}

func (s *CredentialStore) Audits(ctx context.Context) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_audit_logs WHERE source='credential' ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []audit.AuditLog
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var entry audit.AuditLog
		json.Unmarshal(raw, &entry)
		out = append(out, entry)
	}
	if out == nil {
		out = []audit.AuditLog{}
	}
	return out, rows.Err()
}
