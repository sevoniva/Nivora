package notification

import (
	"context"

	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

type Provider interface {
	Send(ctx context.Context, notification domainnotification.Notification) error
}
