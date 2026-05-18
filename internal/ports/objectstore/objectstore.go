package objectstore

import (
	"context"
	"io"
)

type ObjectStore interface {
	PutObject(ctx context.Context, key string, body io.Reader) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	DeleteObject(ctx context.Context, key string) error
}
