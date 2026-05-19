package auth

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/domain/event"
)

var ErrMembershipNotFound = errors.New("membership not found")

type Store interface {
	SaveUser(ctx context.Context, user domainauth.User) error
	ListUsers(ctx context.Context) ([]domainauth.User, error)
	SaveMembership(ctx context.Context, membership domainauth.Membership) error
	ListMemberships(ctx context.Context, scopeType string, scopeID string) ([]domainauth.Membership, error)
	DeleteMembership(ctx context.Context, id string) error
	AppendEvent(ctx context.Context, evt event.Event) error
	AppendAudit(ctx context.Context, entry audit.AuditLog) error
}

type MemoryStore struct {
	mu          sync.RWMutex
	users       map[string]domainauth.User
	memberships map[string]domainauth.Membership
	events      []event.Event
	audits      []audit.AuditLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{users: make(map[string]domainauth.User), memberships: make(map[string]domainauth.Membership)}
}

func (s *MemoryStore) SaveUser(ctx context.Context, user domainauth.User) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user.ID] = user
	return nil
}

func (s *MemoryStore) ListUsers(ctx context.Context) ([]domainauth.User, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]domainauth.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Username < users[j].Username })
	return users, nil
}

func (s *MemoryStore) SaveMembership(ctx context.Context, membership domainauth.Membership) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memberships[membership.ID] = membership
	return nil
}

func (s *MemoryStore) ListMemberships(ctx context.Context, scopeType string, scopeID string) ([]domainauth.Membership, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	memberships := make([]domainauth.Membership, 0, len(s.memberships))
	for _, membership := range s.memberships {
		if scopeType != "" && membership.ScopeType != scopeType {
			continue
		}
		if scopeID != "" && membership.ScopeID != scopeID {
			continue
		}
		memberships = append(memberships, membership)
	}
	sort.Slice(memberships, func(i, j int) bool { return memberships[i].CreatedAt.Before(memberships[j].CreatedAt) })
	return memberships, nil
}

func (s *MemoryStore) DeleteMembership(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.memberships[id]; !ok {
		return ErrMembershipNotFound
	}
	delete(s.memberships, id)
	return nil
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
