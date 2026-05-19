package tenancy

import (
	"context"
	"errors"
	"sync"

	domaintenant "github.com/sevoniva/nivora/internal/domain/tenant"
)

var ErrQuotaNotFound = errors.New("quota not found")

type Store interface {
	SaveQuota(ctx context.Context, quota domaintenant.Quota) error
	GetQuota(ctx context.Context, scopeType, scopeID string) (domaintenant.Quota, error)
	SaveUsage(ctx context.Context, usage domaintenant.UsageSummary) error
	GetUsage(ctx context.Context, scopeType, scopeID string) (domaintenant.UsageSummary, error)
}

type MemoryStore struct {
	mu     sync.RWMutex
	quotas map[string]domaintenant.Quota
	usage  map[string]domaintenant.UsageSummary
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		quotas: make(map[string]domaintenant.Quota),
		usage:  make(map[string]domaintenant.UsageSummary),
	}
}

func (s *MemoryStore) SaveQuota(ctx context.Context, quota domaintenant.Quota) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quotas[quota.Scope.Type+"/"+quota.Scope.ID] = quota
	return nil
}

func (s *MemoryStore) GetQuota(ctx context.Context, scopeType, scopeID string) (domaintenant.Quota, error) {
	if err := ctx.Err(); err != nil {
		return domaintenant.Quota{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := scopeType + "/" + scopeID
	q, ok := s.quotas[key]
	if !ok {
		return domaintenant.Quota{}, ErrQuotaNotFound
	}
	return q, nil
}

func (s *MemoryStore) SaveUsage(ctx context.Context, usage domaintenant.UsageSummary) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usage[usage.Scope.Type+"/"+usage.Scope.ID] = usage
	return nil
}

func (s *MemoryStore) GetUsage(ctx context.Context, scopeType, scopeID string) (domaintenant.UsageSummary, error) {
	if err := ctx.Err(); err != nil {
		return domaintenant.UsageSummary{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	us, ok := s.usage[scopeType+"/"+scopeID]
	if !ok {
		return domaintenant.UsageSummary{}, nil
	}
	return us, nil
}
