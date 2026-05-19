package approval

import (
	"context"
	"errors"
	"sort"
	"sync"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

var ErrApprovalNotFound = errors.New("approval request not found")
var ErrChangeWindowNotFound = errors.New("change window not found")

type Store interface {
	SaveApproval(ctx context.Context, request domainapproval.ApprovalRequest) error
	GetApproval(ctx context.Context, id string) (domainapproval.ApprovalRequest, error)
	ListApprovals(ctx context.Context) ([]domainapproval.ApprovalRequest, error)
	SaveChangeWindow(ctx context.Context, window domainapproval.ChangeWindow) error
	GetChangeWindow(ctx context.Context, id string) (domainapproval.ChangeWindow, error)
	ListChangeWindows(ctx context.Context) ([]domainapproval.ChangeWindow, error)
	SaveNotification(ctx context.Context, notification domainnotification.Notification) error
	ListNotifications(ctx context.Context) ([]domainnotification.Notification, error)
	AppendEvent(ctx context.Context, evt event.Event) error
	AppendAudit(ctx context.Context, entry audit.AuditLog) error
	Events(ctx context.Context) ([]event.Event, error)
	Audits(ctx context.Context) ([]audit.AuditLog, error)
}

type MemoryStore struct {
	mu            sync.RWMutex
	approvals     map[string]domainapproval.ApprovalRequest
	windows       map[string]domainapproval.ChangeWindow
	notifications []domainnotification.Notification
	events        []event.Event
	audits        []audit.AuditLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{approvals: map[string]domainapproval.ApprovalRequest{}, windows: map[string]domainapproval.ChangeWindow{}}
}

func (s *MemoryStore) SaveApproval(ctx context.Context, request domainapproval.ApprovalRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.approvals[request.ID] = cloneApproval(request)
	return nil
}

func (s *MemoryStore) GetApproval(ctx context.Context, id string) (domainapproval.ApprovalRequest, error) {
	select {
	case <-ctx.Done():
		return domainapproval.ApprovalRequest{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	request, ok := s.approvals[id]
	if !ok {
		return domainapproval.ApprovalRequest{}, ErrApprovalNotFound
	}
	return cloneApproval(request), nil
}

func (s *MemoryStore) ListApprovals(ctx context.Context) ([]domainapproval.ApprovalRequest, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	approvals := make([]domainapproval.ApprovalRequest, 0, len(s.approvals))
	for _, request := range s.approvals {
		approvals = append(approvals, cloneApproval(request))
	}
	sort.Slice(approvals, func(i, j int) bool { return approvals[i].RequestedAt.Before(approvals[j].RequestedAt) })
	return approvals, nil
}

func (s *MemoryStore) SaveChangeWindow(ctx context.Context, window domainapproval.ChangeWindow) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.windows[window.ID] = cloneWindow(window)
	return nil
}

func (s *MemoryStore) GetChangeWindow(ctx context.Context, id string) (domainapproval.ChangeWindow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	window, ok := s.windows[id]
	if !ok {
		return domainapproval.ChangeWindow{}, ErrChangeWindowNotFound
	}
	return cloneWindow(window), nil
}

func (s *MemoryStore) ListChangeWindows(ctx context.Context) ([]domainapproval.ChangeWindow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	windows := make([]domainapproval.ChangeWindow, 0, len(s.windows))
	for _, window := range s.windows {
		windows = append(windows, cloneWindow(window))
	}
	return windows, nil
}

func (s *MemoryStore) SaveNotification(ctx context.Context, notification domainnotification.Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = append(s.notifications, notification)
	return nil
}

func (s *MemoryStore) ListNotifications(ctx context.Context) ([]domainnotification.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domainnotification.Notification(nil), s.notifications...), nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, evt event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, evt)
	return nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, entry audit.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, entry)
	return nil
}

func (s *MemoryStore) Events(ctx context.Context) ([]event.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]event.Event(nil), s.events...), nil
}

func (s *MemoryStore) Audits(ctx context.Context) ([]audit.AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]audit.AuditLog(nil), s.audits...), nil
}

func cloneApproval(request domainapproval.ApprovalRequest) domainapproval.ApprovalRequest {
	request.Participants = append([]domainapproval.ApprovalParticipant(nil), request.Participants...)
	request.Decisions = append([]domainapproval.ApprovalDecision(nil), request.Decisions...)
	return request
}

func cloneWindow(window domainapproval.ChangeWindow) domainapproval.ChangeWindow {
	window.DaysOfWeek = append([]string(nil), window.DaysOfWeek...)
	if window.Metadata != nil {
		out := make(map[string]string, len(window.Metadata))
		for k, v := range window.Metadata {
			out[k] = v
		}
		window.Metadata = out
	}
	return window
}
