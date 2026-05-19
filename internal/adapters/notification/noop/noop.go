package noop

import (
	"context"

	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

type Provider struct {
	Sent []domainnotification.Notification
}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Send(ctx context.Context, notification domainnotification.Notification) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	p.Sent = append(p.Sent, notification)
	return nil
}
