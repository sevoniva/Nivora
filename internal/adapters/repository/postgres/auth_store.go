package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/domain/event"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

type AuthStore struct {
	pool *pgxpool.Pool
}

var _ authusecase.Store = (*AuthStore)(nil)

func NewAuthStore(pool *pgxpool.Pool) *AuthStore {
	return &AuthStore{pool: pool}
}

// --- Users ---

func (s *AuthStore) SaveUser(ctx context.Context, user domainauth.User) error {
	if user.ID == "" {
		return errors.New("user id is required")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO auth_users (id, username, email, display_name, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (id) DO UPDATE SET username=EXCLUDED.username, email=EXCLUDED.email, display_name=EXCLUDED.display_name, status=EXCLUDED.status, updated_at=EXCLUDED.updated_at`,
		user.ID, user.Username, user.Email, user.DisplayName, user.Status, user.CreatedAt, user.UpdatedAt)
	return err
}

func (s *AuthStore) ListUsers(ctx context.Context) ([]domainauth.User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, username, email, display_name, status, created_at, updated_at FROM auth_users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []domainauth.User
	for rows.Next() {
		var u domainauth.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.DisplayName, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if users == nil {
		users = []domainauth.User{}
	}
	return users, rows.Err()
}

// --- Memberships ---

func (s *AuthStore) SaveMembership(ctx context.Context, m domainauth.Membership) error {
	if m.ID == "" {
		return errors.New("membership id is required")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO auth_memberships (id, scope_type, scope_id, user_id, role, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (id) DO UPDATE SET scope_type=EXCLUDED.scope_type, scope_id=EXCLUDED.scope_id, user_id=EXCLUDED.user_id, role=EXCLUDED.role, updated_at=EXCLUDED.updated_at`,
		m.ID, m.ScopeType, m.ScopeID, m.UserID, m.Role, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *AuthStore) ListMemberships(ctx context.Context, scopeType, scopeID string) ([]domainauth.Membership, error) {
	var rows pgx.Rows
	var err error
	if scopeType != "" || scopeID != "" {
		rows, err = s.pool.Query(ctx, `SELECT id, scope_type, scope_id, user_id, role, created_at, updated_at FROM auth_memberships WHERE scope_type=$1 AND ($2='' OR scope_id=$2) ORDER BY created_at`, scopeType, scopeID)
	} else {
		rows, err = s.pool.Query(ctx, `SELECT id, scope_type, scope_id, user_id, role, created_at, updated_at FROM auth_memberships ORDER BY created_at`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainauth.Membership
	for rows.Next() {
		var m domainauth.Membership
		if err := rows.Scan(&m.ID, &m.ScopeType, &m.ScopeID, &m.UserID, &m.Role, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	if out == nil {
		out = []domainauth.Membership{}
	}
	return out, rows.Err()
}

func (s *AuthStore) DeleteMembership(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM auth_memberships WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return authusecase.ErrMembershipNotFound
	}
	return nil
}

// --- Service Accounts ---

func (s *AuthStore) SaveServiceAccount(ctx context.Context, a domainauth.ServiceAccount) error {
	if a.ID == "" {
		return errors.New("service account id is required")
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO auth_service_accounts (id, name, scope_type, scope_id, role, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, scope_type=EXCLUDED.scope_type, scope_id=EXCLUDED.scope_id, role=EXCLUDED.role, status=EXCLUDED.status, updated_at=EXCLUDED.updated_at`,
		a.ID, a.Name, a.ScopeType, a.ScopeID, a.Role, a.Status, a.CreatedAt, a.UpdatedAt)
	return err
}

func (s *AuthStore) ListServiceAccounts(ctx context.Context, scopeType, scopeID string) ([]domainauth.ServiceAccount, error) {
	var rows pgx.Rows
	var err error
	if scopeType != "" || scopeID != "" {
		rows, err = s.pool.Query(ctx, `SELECT id, name, scope_type, scope_id, role, status, created_at, updated_at FROM auth_service_accounts WHERE scope_type=$1 AND ($2='' OR scope_id=$2) ORDER BY name`, scopeType, scopeID)
	} else {
		rows, err = s.pool.Query(ctx, `SELECT id, name, scope_type, scope_id, role, status, created_at, updated_at FROM auth_service_accounts ORDER BY name`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainauth.ServiceAccount
	for rows.Next() {
		var a domainauth.ServiceAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.ScopeType, &a.ScopeID, &a.Role, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if out == nil {
		out = []domainauth.ServiceAccount{}
	}
	return out, rows.Err()
}

func (s *AuthStore) GetServiceAccount(ctx context.Context, id string) (domainauth.ServiceAccount, error) {
	var a domainauth.ServiceAccount
	err := s.pool.QueryRow(ctx, `SELECT id, name, scope_type, scope_id, role, status, created_at, updated_at FROM auth_service_accounts WHERE id=$1`, id).
		Scan(&a.ID, &a.Name, &a.ScopeType, &a.ScopeID, &a.Role, &a.Status, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainauth.ServiceAccount{}, errors.New("service account not found")
	}
	return a, err
}

// --- Tokens ---

func (s *AuthStore) SaveToken(ctx context.Context, token domainauth.TokenMetadata) error {
	if token.ID == "" {
		return errors.New("token id is required")
	}
	if token.TokenHash == "" {
		return errors.New("token hash is required")
	}
	rolesJSON, _ := json.Marshal(token.Roles)
	_, err := s.pool.Exec(ctx, `INSERT INTO auth_api_tokens (id, subject_id, subject_type, name, scope_type, scope_id, role, token_hash, issued_at, expires_at, revoked_at, last_used_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET
			subject_id=EXCLUDED.subject_id,
			subject_type=EXCLUDED.subject_type,
			name=EXCLUDED.name,
			scope_type=EXCLUDED.scope_type,
			scope_id=EXCLUDED.scope_id,
			role=EXCLUDED.role,
			token_hash=EXCLUDED.token_hash,
			issued_at=EXCLUDED.issued_at,
			expires_at=EXCLUDED.expires_at,
			revoked_at=EXCLUDED.revoked_at,
			last_used_at=EXCLUDED.last_used_at`,
		token.ID, token.SubjectID, token.SubjectType, token.Name, token.ScopeType, token.ScopeID, string(rolesJSON), token.TokenHash, token.IssuedAt, token.ExpiresAt, token.RevokedAt, token.LastUsedAt)
	return err
}

func (s *AuthStore) GetToken(ctx context.Context, id string) (domainauth.TokenMetadata, error) {
	var t domainauth.TokenMetadata
	var rolesJSON string
	err := s.pool.QueryRow(ctx, `SELECT id, subject_id, subject_type, name, scope_type, scope_id, role, token_hash, issued_at, expires_at, revoked_at, last_used_at FROM auth_api_tokens WHERE id=$1`, id).
		Scan(&t.ID, &t.SubjectID, &t.SubjectType, &t.Name, &t.ScopeType, &t.ScopeID, &rolesJSON, &t.TokenHash, &t.IssuedAt, &t.ExpiresAt, &t.RevokedAt, &t.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainauth.TokenMetadata{}, authusecase.ErrTokenNotFound
	}
	if err != nil {
		return domainauth.TokenMetadata{}, err
	}
	json.Unmarshal([]byte(rolesJSON), &t.Roles)
	return t, nil
}

func (s *AuthStore) FindTokenByHash(ctx context.Context, hash string) (domainauth.TokenMetadata, error) {
	var t domainauth.TokenMetadata
	var rolesJSON string
	err := s.pool.QueryRow(ctx, `SELECT id, subject_id, subject_type, name, scope_type, scope_id, role, token_hash, issued_at, expires_at, revoked_at, last_used_at FROM auth_api_tokens WHERE token_hash=$1 AND revoked_at IS NULL`, hash).
		Scan(&t.ID, &t.SubjectID, &t.SubjectType, &t.Name, &t.ScopeType, &t.ScopeID, &rolesJSON, &t.TokenHash, &t.IssuedAt, &t.ExpiresAt, &t.RevokedAt, &t.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainauth.TokenMetadata{}, authusecase.ErrTokenNotFound
	}
	if err != nil {
		return domainauth.TokenMetadata{}, err
	}
	json.Unmarshal([]byte(rolesJSON), &t.Roles)
	return t, nil
}

func (s *AuthStore) ListTokens(ctx context.Context, subjectID string) ([]domainauth.TokenMetadata, error) {
	var rows pgx.Rows
	var err error
	if subjectID != "" {
		rows, err = s.pool.Query(ctx, `SELECT id, subject_id, subject_type, name, scope_type, scope_id, role, token_hash, issued_at, expires_at, revoked_at, last_used_at FROM auth_api_tokens WHERE subject_id=$1 ORDER BY issued_at`, subjectID)
	} else {
		rows, err = s.pool.Query(ctx, `SELECT id, subject_id, subject_type, name, scope_type, scope_id, role, token_hash, issued_at, expires_at, revoked_at, last_used_at FROM auth_api_tokens ORDER BY issued_at`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainauth.TokenMetadata
	for rows.Next() {
		var t domainauth.TokenMetadata
		var rolesJSON string
		if err := rows.Scan(&t.ID, &t.SubjectID, &t.SubjectType, &t.Name, &t.ScopeType, &t.ScopeID, &rolesJSON, &t.TokenHash, &t.IssuedAt, &t.ExpiresAt, &t.RevokedAt, &t.LastUsedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(rolesJSON), &t.Roles)
		t.TokenHash = ""
		out = append(out, t)
	}
	if out == nil {
		out = []domainauth.TokenMetadata{}
	}
	return out, rows.Err()
}

// --- Events & Audit ---

func (s *AuthStore) AppendEvent(ctx context.Context, evt event.Event) error {
	payload, _ := json.Marshal(evt)
	_, err := s.pool.Exec(ctx, `INSERT INTO governance_event_logs (id, source, event_type, subject, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		evt.ID, "auth", evt.Type, evt.Subject, payload, evt.Time)
	return err
}

func (s *AuthStore) AppendAudit(ctx context.Context, entry audit.AuditLog) error {
	return AppendHashChainedAudit(ctx, s.pool, "auth", entry)
}
