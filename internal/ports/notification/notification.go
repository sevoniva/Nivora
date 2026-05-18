package notification

import "context"

type Message struct {
	To      []string
	Subject string
	Body    string
}

type Provider interface {
	Send(ctx context.Context, message Message) error
}
