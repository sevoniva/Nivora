package local

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	root string
}

func New(root string) *Store {
	return &Store{root: root}
}

func (s *Store) PutObject(ctx context.Context, key string, body io.Reader) error {
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, readerWithContext{ctx: ctx, r: body})
	return err
}

func (s *Store) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	path, err := s.pathFor(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *Store) DeleteObject(ctx context.Context, key string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	path, err := s.pathFor(key)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (s *Store) pathFor(key string) (string, error) {
	clean := filepath.Clean(key)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", errors.New("invalid object key")
	}
	return filepath.Join(s.root, clean), nil
}

type readerWithContext struct {
	ctx context.Context
	r   io.Reader
}

func (r readerWithContext) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}
