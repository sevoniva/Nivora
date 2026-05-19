package credential

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	"github.com/sevoniva/nivora/internal/domain/event"
)

var ErrCredentialNotFound = errors.New("credential not found")

type Store interface {
	SaveCredential(ctx context.Context, cred domaincredential.Credential) error
	GetCredential(ctx context.Context, id string) (domaincredential.Credential, error)
	ListCredentials(ctx context.Context) ([]domaincredential.Credential, error)
	DeleteCredential(ctx context.Context, id string) error
	AppendEvent(ctx context.Context, evt event.Event) error
	Events(ctx context.Context) ([]event.Event, error)
	AppendAudit(ctx context.Context, entry audit.AuditLog) error
	Audits(ctx context.Context) ([]audit.AuditLog, error)
}

type MemoryStore struct {
	mu          sync.RWMutex
	credentials map[string]domaincredential.Credential
	events      []event.Event
	audits      []audit.AuditLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{credentials: make(map[string]domaincredential.Credential)}
}

func (s *MemoryStore) SaveCredential(ctx context.Context, cred domaincredential.Credential) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[cred.ID] = cloneCredential(cred)
	return nil
}

func (s *MemoryStore) GetCredential(ctx context.Context, id string) (domaincredential.Credential, error) {
	select {
	case <-ctx.Done():
		return domaincredential.Credential{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	cred, ok := s.credentials[id]
	if !ok {
		return domaincredential.Credential{}, ErrCredentialNotFound
	}
	return cloneCredential(cred), nil
}

func (s *MemoryStore) ListCredentials(ctx context.Context) ([]domaincredential.Credential, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	credentials := make([]domaincredential.Credential, 0, len(s.credentials))
	for _, cred := range s.credentials {
		credentials = append(credentials, cloneCredential(cred))
	}
	sort.Slice(credentials, func(i, j int) bool { return credentials[i].CreatedAt.Before(credentials[j].CreatedAt) })
	return credentials, nil
}

func (s *MemoryStore) DeleteCredential(ctx context.Context, id string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.credentials[id]; !ok {
		return ErrCredentialNotFound
	}
	delete(s.credentials, id)
	return nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, evt event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, evt)
	return nil
}

func (s *MemoryStore) Events(ctx context.Context) ([]event.Event, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := append([]event.Event(nil), s.events...)
	sort.Slice(events, func(i, j int) bool { return events[i].Time.Before(events[j].Time) })
	return events, nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, entry audit.AuditLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, entry)
	return nil
}

func (s *MemoryStore) Audits(ctx context.Context) ([]audit.AuditLog, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]audit.AuditLog(nil), s.audits...), nil
}

func cloneCredential(cred domaincredential.Credential) domaincredential.Credential {
	cred.Metadata = cloneMap(cred.Metadata)
	cred.SecretRef.Metadata = cloneMap(cred.SecretRef.Metadata)
	return cred
}

func cloneMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
