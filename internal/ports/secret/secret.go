package secret

import "context"

type Provider interface {
	GetSecret(ctx context.Context, key string) ([]byte, error)
	PutSecret(ctx context.Context, key string, value []byte) error
	DeleteSecret(ctx context.Context, key string) error
}
