package policy

import "context"

type Request struct {
	Subject string
	Action  string
	Context map[string]any
}

type Result struct {
	Allowed bool
	Reasons []string
}

type Engine interface {
	Evaluate(ctx context.Context, request Request) (Result, error)
}
