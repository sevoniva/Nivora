package memory

import (
	"context"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/event"
)

type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]chan event.Event
}

func New() *Bus {
	return &Bus{subscribers: make(map[string][]chan event.Event)}
}

func (b *Bus) Publish(ctx context.Context, evt event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, topic := range []string{evt.Type, "*"} {
		for _, ch := range b.subscribers[topic] {
			select {
			case ch <- evt:
			default:
			}
		}
	}
	return nil
}

func (b *Bus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event, 16)

	b.mu.Lock()
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	b.mu.Unlock()

	go func() {
		<-ctx.Done()
		b.mu.Lock()
		defer b.mu.Unlock()
		subs := b.subscribers[eventType]
		for i, sub := range subs {
			if sub == ch {
				b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, nil
}
