package policy

import "context"

type Request struct {
	Subject string         `json:"subject"`
	Action  string         `json:"action"`
	Context map[string]any `json:"context,omitempty"`
}

type Result struct {
	Allowed bool     `json:"allowed"`
	Reasons []string `json:"reasons,omitempty"`
}

type Engine interface {
	Evaluate(ctx context.Context, request Request) (Result, error)
}
