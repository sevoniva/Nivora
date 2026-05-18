package eventbus

import (
	"context"

	"github.com/sevoniva/nivora/internal/domain/event"
)

type EventBus interface {
	Publish(ctx context.Context, evt event.Event) error
	Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error)
}
