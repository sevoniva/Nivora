package builtin

import (
	"context"
	"errors"
	"sync"
)

type Store struct {
	mu      sync.RWMutex
	secrets map[string][]byte
}

func New() *Store {
	return &Store{secrets: make(map[string][]byte)}
}

func (s *Store) GetSecret(ctx context.Context, key string) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.secrets[key]
	if !ok {
		return nil, errors.New("secret not found")
	}
	return append([]byte(nil), value...), nil
}

func (s *Store) PutSecret(ctx context.Context, key string, value []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[key] = append([]byte(nil), value...)
	return nil
}

func (s *Store) DeleteSecret(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, key)
	return nil
}
