package builtin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sevoniva/nivora/internal/domain/credential"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

var ErrSecretNotFound = errors.New("secret not found")

type Store struct {
	mu       sync.RWMutex
	values   map[string][]byte
	refs     map[string]credential.SecretRef
	keyIndex map[string]string
	usages   []credential.SecretUsage
	now      func() time.Time
}

func New() *Store {
	return &Store{
		values:   make(map[string][]byte),
		refs:     make(map[string]credential.SecretRef),
		keyIndex: make(map[string]string),
		now:      time.Now,
	}
}

func (s *Store) ValidateProvider(ctx context.Context) (portsecret.ProviderStatus, error) {
	select {
	case <-ctx.Done():
		return portsecret.ProviderStatus{}, ctx.Err()
	default:
	}
	return portsecret.ProviderStatus{
		Provider:     "builtin",
		Configured:   true,
		Reachable:    true,
		Capabilities: []string{"put", "get", "delete", "rotate", "list", "usage_audit"},
		Message:      "builtin development secret provider is available",
	}, nil
}

func (s *Store) PutSecret(ctx context.Context, request portsecret.PutRequest) (credential.SecretRef, error) {
	select {
	case <-ctx.Done():
		return credential.SecretRef{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ref := request.Ref
	if ref.Key == "" {
		return credential.SecretRef{}, errors.New("secret key is required")
	}
	now := s.now()
	if ref.ID == "" {
		if existingID := s.keyIndex[ref.Key]; existingID != "" {
			ref.ID = existingID
		} else {
			ref.ID = fmt.Sprintf("secret-%d", now.UnixNano())
		}
	}
	if ref.Provider == "" {
		ref.Provider = "builtin"
	}
	if ref.ScopeType == "" {
		ref.ScopeType = credential.ScopeGlobal
	}
	if ref.Version == "" {
		ref.Version = "1"
	}
	if ref.CreatedAt.IsZero() {
		ref.CreatedAt = now
	}
	ref.UpdatedAt = now
	s.values[ref.ID] = append([]byte(nil), request.Value...)
	s.refs[ref.ID] = cloneRef(ref)
	s.keyIndex[ref.Key] = ref.ID
	return cloneRef(ref), nil
}

func (s *Store) GetSecret(ctx context.Context, ref credential.SecretRef) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.values[s.resolveID(ref)]
	if !ok {
		return nil, ErrSecretNotFound
	}
	return append([]byte(nil), value...), nil
}

func (s *Store) DeleteSecret(ctx context.Context, ref credential.SecretRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.resolveID(ref)
	stored, ok := s.refs[id]
	if !ok {
		return ErrSecretNotFound
	}
	delete(s.values, id)
	delete(s.refs, id)
	delete(s.keyIndex, stored.Key)
	return nil
}

func (s *Store) RotateSecret(ctx context.Context, ref credential.SecretRef, newValue []byte) (credential.SecretRef, error) {
	select {
	case <-ctx.Done():
		return credential.SecretRef{}, ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := s.resolveID(ref)
	stored, ok := s.refs[id]
	if !ok {
		return credential.SecretRef{}, ErrSecretNotFound
	}
	now := s.now()
	stored.Version = fmt.Sprintf("%d", now.UnixNano())
	stored.UpdatedAt = now
	s.refs[id] = cloneRef(stored)
	s.values[id] = append([]byte(nil), newValue...)
	return cloneRef(stored), nil
}

func (s *Store) ListSecretRefs(ctx context.Context, scope portsecret.Scope) ([]credential.SecretRef, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	refs := make([]credential.SecretRef, 0, len(s.refs))
	for _, ref := range s.refs {
		if scope.ScopeType != "" && ref.ScopeType != scope.ScopeType {
			continue
		}
		if scope.ScopeID != "" && ref.ScopeID != scope.ScopeID {
			continue
		}
		refs = append(refs, cloneRef(ref))
	}
	return refs, nil
}

func (s *Store) RecordUsage(ctx context.Context, usage credential.SecretUsage) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if usage.ID == "" {
		usage.ID = fmt.Sprintf("usage-%d", s.now().UnixNano())
	}
	if usage.CreatedAt.IsZero() {
		usage.CreatedAt = s.now()
	}
	s.usages = append(s.usages, usage)
	return nil
}

func (s *Store) resolveID(ref credential.SecretRef) string {
	if ref.ID != "" {
		return ref.ID
	}
	return s.keyIndex[ref.Key]
}

func cloneRef(ref credential.SecretRef) credential.SecretRef {
	ref.Metadata = cloneMap(ref.Metadata)
	return ref
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
