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
var ErrTokenNotFound = errors.New("api token not found")

type Store interface {
	SaveUser(ctx context.Context, user domainauth.User) error
	ListUsers(ctx context.Context) ([]domainauth.User, error)
	SaveMembership(ctx context.Context, membership domainauth.Membership) error
	ListMemberships(ctx context.Context, scopeType string, scopeID string) ([]domainauth.Membership, error)
	DeleteMembership(ctx context.Context, id string) error
	SaveServiceAccount(ctx context.Context, account domainauth.ServiceAccount) error
	ListServiceAccounts(ctx context.Context, scopeType string, scopeID string) ([]domainauth.ServiceAccount, error)
	GetServiceAccount(ctx context.Context, id string) (domainauth.ServiceAccount, error)
	SaveToken(ctx context.Context, token domainauth.TokenMetadata) error
	GetToken(ctx context.Context, id string) (domainauth.TokenMetadata, error)
	FindTokenByHash(ctx context.Context, hash string) (domainauth.TokenMetadata, error)
	ListTokens(ctx context.Context, subjectID string) ([]domainauth.TokenMetadata, error)
	AppendEvent(ctx context.Context, evt event.Event) error
	AppendAudit(ctx context.Context, entry audit.AuditLog) error
}

type MemoryStore struct {
	mu              sync.RWMutex
	users           map[string]domainauth.User
	memberships     map[string]domainauth.Membership
	serviceAccounts map[string]domainauth.ServiceAccount
	tokens          map[string]domainauth.TokenMetadata
	events          []event.Event
	audits          []audit.AuditLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:           make(map[string]domainauth.User),
		memberships:     make(map[string]domainauth.Membership),
		serviceAccounts: make(map[string]domainauth.ServiceAccount),
		tokens:          make(map[string]domainauth.TokenMetadata),
	}
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

func (s *MemoryStore) SaveServiceAccount(ctx context.Context, account domainauth.ServiceAccount) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serviceAccounts[account.ID] = account
	return nil
}

func (s *MemoryStore) ListServiceAccounts(ctx context.Context, scopeType string, scopeID string) ([]domainauth.ServiceAccount, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	accounts := make([]domainauth.ServiceAccount, 0, len(s.serviceAccounts))
	for _, account := range s.serviceAccounts {
		if scopeType != "" && account.ScopeType != scopeType {
			continue
		}
		if scopeID != "" && account.ScopeID != scopeID {
			continue
		}
		accounts = append(accounts, account)
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Name < accounts[j].Name })
	return accounts, nil
}

func (s *MemoryStore) GetServiceAccount(ctx context.Context, id string) (domainauth.ServiceAccount, error) {
	select {
	case <-ctx.Done():
		return domainauth.ServiceAccount{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.serviceAccounts[id]
	if !ok {
		return domainauth.ServiceAccount{}, errors.New("service account not found")
	}
	return account, nil
}

func (s *MemoryStore) SaveToken(ctx context.Context, token domainauth.TokenMetadata) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token.ID] = token
	return nil
}

func (s *MemoryStore) GetToken(ctx context.Context, id string) (domainauth.TokenMetadata, error) {
	select {
	case <-ctx.Done():
		return domainauth.TokenMetadata{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, ok := s.tokens[id]
	if !ok {
		return domainauth.TokenMetadata{}, errors.New("api token not found")
	}
	return token, nil
}

func (s *MemoryStore) FindTokenByHash(ctx context.Context, hash string) (domainauth.TokenMetadata, error) {
	select {
	case <-ctx.Done():
		return domainauth.TokenMetadata{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, token := range s.tokens {
		if token.TokenHash == hash {
			return token, nil
		}
	}
	return domainauth.TokenMetadata{}, ErrTokenNotFound
}

func (s *MemoryStore) ListTokens(ctx context.Context, subjectID string) ([]domainauth.TokenMetadata, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	tokens := make([]domainauth.TokenMetadata, 0, len(s.tokens))
	for _, token := range s.tokens {
		if subjectID != "" && token.SubjectID != subjectID {
			continue
		}
		token.TokenHash = ""
		tokens = append(tokens, token)
	}
	sort.Slice(tokens, func(i, j int) bool { return tokens[i].IssuedAt.Before(tokens[j].IssuedAt) })
	return tokens, nil
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
