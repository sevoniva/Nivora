package log

import (
	"context"
	"log/slog"

	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

type Provider struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) Provider {
	if logger == nil {
		logger = slog.Default()
	}
	return Provider{logger: logger}
}

func (p Provider) Send(ctx context.Context, notification domainnotification.Notification) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	p.logger.Info("notification recorded",
		"id", notification.ID,
		"type", notification.Type,
		"channel", notification.Channel,
		"subject", notification.Subject,
		"recipient_count", len(notification.Recipients),
	)
	return nil
}
