package email

import (
	"context"
	"errors"

	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

type Provider struct {
	Configured bool
}

func New(configured bool) Provider {
	return Provider{Configured: configured}
}

func (p Provider) Send(ctx context.Context, notification domainnotification.Notification) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if !p.Configured {
		return errors.New("email notification provider is a placeholder and is not configured")
	}
	return nil
}
