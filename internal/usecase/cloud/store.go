package cloud

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	"github.com/sevoniva/nivora/internal/domain/event"
)

var ErrAccountNotFound = errors.New("cloud account not found")

type Store interface {
	SaveAccount(ctx context.Context, account domaincloud.CloudAccount) error
	GetAccount(ctx context.Context, id string) (domaincloud.CloudAccount, error)
	ListAccounts(ctx context.Context) ([]domaincloud.CloudAccount, error)
	SaveSnapshot(ctx context.Context, snapshot domaincloud.CloudInventorySnapshot) error
	GetSnapshot(ctx context.Context, accountID string) (domaincloud.CloudInventorySnapshot, error)
	AppendEvent(ctx context.Context, evt event.Event) error
	AppendAudit(ctx context.Context, entry audit.AuditLog) error
}

type MemoryStore struct {
	mu        sync.RWMutex
	accounts  map[string]domaincloud.CloudAccount
	snapshots map[string]domaincloud.CloudInventorySnapshot
	events    []event.Event
	audits    []audit.AuditLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{accounts: map[string]domaincloud.CloudAccount{}, snapshots: map[string]domaincloud.CloudInventorySnapshot{}}
}

func (s *MemoryStore) SaveAccount(ctx context.Context, account domaincloud.CloudAccount) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accounts[account.ID] = cloneAccount(account)
	return nil
}

func (s *MemoryStore) GetAccount(ctx context.Context, id string) (domaincloud.CloudAccount, error) {
	select {
	case <-ctx.Done():
		return domaincloud.CloudAccount{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	account, ok := s.accounts[id]
	if !ok {
		return domaincloud.CloudAccount{}, ErrAccountNotFound
	}
	return cloneAccount(account), nil
}

func (s *MemoryStore) ListAccounts(ctx context.Context) ([]domaincloud.CloudAccount, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	accounts := make([]domaincloud.CloudAccount, 0, len(s.accounts))
	for _, account := range s.accounts {
		accounts = append(accounts, cloneAccount(account))
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].CreatedAt.Before(accounts[j].CreatedAt) })
	return accounts, nil
}

func (s *MemoryStore) SaveSnapshot(ctx context.Context, snapshot domaincloud.CloudInventorySnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[snapshot.AccountID] = snapshot
	return nil
}

func (s *MemoryStore) GetSnapshot(ctx context.Context, accountID string) (domaincloud.CloudInventorySnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshots[accountID], nil
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

func cloneAccount(account domaincloud.CloudAccount) domaincloud.CloudAccount {
	if account.Metadata != nil {
		metadata := make(map[string]string, len(account.Metadata))
		for k, v := range account.Metadata {
			metadata[k] = v
		}
		account.Metadata = metadata
	}
	if account.Config.Metadata != nil {
		metadata := make(map[string]string, len(account.Config.Metadata))
		for k, v := range account.Config.Metadata {
			metadata[k] = v
		}
		account.Config.Metadata = metadata
	}
	return account
}
