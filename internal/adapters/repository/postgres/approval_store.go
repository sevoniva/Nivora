package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
)

type ApprovalStore struct {
	pool *pgxpool.Pool
}

var _ approvalusecase.Store = (*ApprovalStore)(nil)

func NewApprovalStore(pool *pgxpool.Pool) *ApprovalStore {
	return &ApprovalStore{pool: pool}
}

// --- Approvals ---

func (s *ApprovalStore) SaveApproval(ctx context.Context, req domainapproval.ApprovalRequest) error {
	if req.ID == "" {
		return errors.New("approval id is required")
	}
	participantsJSON, _ := json.Marshal(req.Participants)
	decisionsJSON, _ := json.Marshal(req.Decisions)
	_, err := s.pool.Exec(ctx, `INSERT INTO approval_requests (id, subject_type, subject_id, environment_id, target_type, target_id, severity, policy_result_id, required_by_policy, status, requested_by, requested_at, expires_at, reason, participants, decisions, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		ON CONFLICT (id) DO UPDATE SET status=EXCLUDED.status, decisions=EXCLUDED.decisions, updated_at=EXCLUDED.updated_at`,
		req.ID, req.SubjectType, req.SubjectID, req.EnvironmentID, req.TargetType, req.TargetID, req.Severity, req.PolicyResultID, req.RequiredByPolicy, req.Status, req.RequestedBy, req.RequestedAt, req.ExpiresAt, req.Reason, participantsJSON, decisionsJSON, req.RequestedAt, req.RequestedAt)
	return err
}

func (s *ApprovalStore) GetApproval(ctx context.Context, id string) (domainapproval.ApprovalRequest, error) {
	var req domainapproval.ApprovalRequest
	var participantsJSON, decisionsJSON []byte
	err := s.pool.QueryRow(ctx, `SELECT id, subject_type, subject_id, environment_id, target_type, target_id, severity, policy_result_id, required_by_policy, status, requested_by, requested_at, expires_at, reason, participants, decisions, created_at, updated_at FROM approval_requests WHERE id=$1`, id).
		Scan(&req.ID, &req.SubjectType, &req.SubjectID, &req.EnvironmentID, &req.TargetType, &req.TargetID, &req.Severity, &req.PolicyResultID, &req.RequiredByPolicy, &req.Status, &req.RequestedBy, &req.RequestedAt, &req.ExpiresAt, &req.Reason, &participantsJSON, &decisionsJSON, &req.RequestedAt, &req.RequestedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainapproval.ApprovalRequest{}, approvalusecase.ErrApprovalNotFound
	}
	if err != nil {
		return domainapproval.ApprovalRequest{}, err
	}
	json.Unmarshal(participantsJSON, &req.Participants)
	json.Unmarshal(decisionsJSON, &req.Decisions)
	return req, nil
}

func (s *ApprovalStore) ListApprovals(ctx context.Context) ([]domainapproval.ApprovalRequest, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, subject_type, subject_id, environment_id, target_type, target_id, severity, policy_result_id, required_by_policy, status, requested_by, requested_at, expires_at, reason, participants, decisions, created_at, updated_at FROM approval_requests ORDER BY requested_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainapproval.ApprovalRequest
	for rows.Next() {
		var req domainapproval.ApprovalRequest
		var participantsJSON, decisionsJSON []byte
		if err := rows.Scan(&req.ID, &req.SubjectType, &req.SubjectID, &req.EnvironmentID, &req.TargetType, &req.TargetID, &req.Severity, &req.PolicyResultID, &req.RequiredByPolicy, &req.Status, &req.RequestedBy, &req.RequestedAt, &req.ExpiresAt, &req.Reason, &participantsJSON, &decisionsJSON, &req.RequestedAt, &req.RequestedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(participantsJSON, &req.Participants)
		json.Unmarshal(decisionsJSON, &req.Decisions)
		out = append(out, req)
	}
	if out == nil {
		out = []domainapproval.ApprovalRequest{}
	}
	return out, rows.Err()
}

// --- Change Windows ---

func (s *ApprovalStore) SaveChangeWindow(ctx context.Context, window domainapproval.ChangeWindow) error {
	if window.ID == "" {
		return errors.New("change window id is required")
	}
	daysJSON, _ := json.Marshal(window.DaysOfWeek)
	metadataJSON, _ := json.Marshal(window.Metadata)
	_, err := s.pool.Exec(ctx, `INSERT INTO approval_change_windows (id, name, environment_id, timezone, start_time, end_time, days_of_week, allowed, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, environment_id=EXCLUDED.environment_id, timezone=EXCLUDED.timezone, start_time=EXCLUDED.start_time, end_time=EXCLUDED.end_time, days_of_week=EXCLUDED.days_of_week, allowed=EXCLUDED.allowed, metadata=EXCLUDED.metadata, updated_at=EXCLUDED.updated_at`,
		window.ID, window.Name, window.EnvironmentID, window.Timezone, window.StartTime, window.EndTime, daysJSON, window.Allowed, metadataJSON, window.CreatedAt, window.UpdatedAt)
	return err
}

func (s *ApprovalStore) GetChangeWindow(ctx context.Context, id string) (domainapproval.ChangeWindow, error) {
	var w domainapproval.ChangeWindow
	var daysJSON, metadataJSON []byte
	err := s.pool.QueryRow(ctx, `SELECT id, name, environment_id, timezone, start_time, end_time, days_of_week, allowed, metadata, created_at, updated_at FROM approval_change_windows WHERE id=$1`, id).
		Scan(&w.ID, &w.Name, &w.EnvironmentID, &w.Timezone, &w.StartTime, &w.EndTime, &daysJSON, &w.Allowed, &metadataJSON, &w.CreatedAt, &w.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domainapproval.ChangeWindow{}, approvalusecase.ErrChangeWindowNotFound
	}
	if err != nil {
		return domainapproval.ChangeWindow{}, err
	}
	json.Unmarshal(daysJSON, &w.DaysOfWeek)
	json.Unmarshal(metadataJSON, &w.Metadata)
	return w, nil
}

func (s *ApprovalStore) ListChangeWindows(ctx context.Context) ([]domainapproval.ChangeWindow, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, environment_id, timezone, start_time, end_time, days_of_week, allowed, metadata, created_at, updated_at FROM approval_change_windows ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainapproval.ChangeWindow
	for rows.Next() {
		var w domainapproval.ChangeWindow
		var daysJSON, metadataJSON []byte
		if err := rows.Scan(&w.ID, &w.Name, &w.EnvironmentID, &w.Timezone, &w.StartTime, &w.EndTime, &daysJSON, &w.Allowed, &metadataJSON, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(daysJSON, &w.DaysOfWeek)
		json.Unmarshal(metadataJSON, &w.Metadata)
		out = append(out, w)
	}
	if out == nil {
		out = []domainapproval.ChangeWindow{}
	}
	return out, rows.Err()
}

// --- Notifications ---

func (s *ApprovalStore) SaveNotification(ctx context.Context, n domainnotification.Notification) error {
	recipientsJSON, _ := json.Marshal(n.Recipients)
	metadataJSON, _ := json.Marshal(n.Metadata)
	_, err := s.pool.Exec(ctx, `INSERT INTO approval_notifications (id, notification_type, channel, subject, body_text, recipients, metadata, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		n.ID, n.Type, n.Channel, n.Subject, n.Body, recipientsJSON, metadataJSON, n.CreatedAt)
	return err
}

func (s *ApprovalStore) ListNotifications(ctx context.Context) ([]domainnotification.Notification, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, notification_type, channel, subject, body_text, recipients, metadata, created_at FROM approval_notifications ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domainnotification.Notification
	for rows.Next() {
		var n domainnotification.Notification
		var recipientsJSON, metadataJSON []byte
		if err := rows.Scan(&n.ID, &n.Type, &n.Channel, &n.Subject, &n.Body, &recipientsJSON, &metadataJSON, &n.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(recipientsJSON, &n.Recipients)
		json.Unmarshal(metadataJSON, &n.Metadata)
		out = append(out, n)
	}
	if out == nil {
		out = []domainnotification.Notification{}
	}
	return out, rows.Err()
}

// --- Events & Audit ---

func (s *ApprovalStore) AppendEvent(ctx context.Context, evt event.Event) error {
	payload, _ := json.Marshal(evt)
	_, err := s.pool.Exec(ctx, `INSERT INTO governance_event_logs (id, source, event_type, subject, payload, created_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		evt.ID, "approval", evt.Type, evt.Subject, payload, evt.Time)
	return err
}

func (s *ApprovalStore) Events(ctx context.Context) ([]event.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_event_logs WHERE source='approval' ORDER BY created_at`)
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

func (s *ApprovalStore) AppendAudit(ctx context.Context, entry audit.AuditLog) error {
	return AppendHashChainedAudit(ctx, s.pool, "approval", entry)
}

func (s *ApprovalStore) Audits(ctx context.Context) ([]audit.AuditLog, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM governance_audit_logs WHERE source='approval' ORDER BY created_at`)
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
