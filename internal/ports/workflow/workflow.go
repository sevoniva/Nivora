package workflow

import "context"

type Instance struct {
	ID     string
	Status string
}

type Runtime interface {
	Start(ctx context.Context, workflowName string, input map[string]any) (Instance, error)
	Signal(ctx context.Context, instanceID string, signal string, payload map[string]any) error
	Cancel(ctx context.Context, instanceID string) error
	Get(ctx context.Context, instanceID string) (Instance, error)
}
