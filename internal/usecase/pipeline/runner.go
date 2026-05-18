package pipeline

import (
	"context"
	"time"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

type Runner interface {
	ID() string
	RunShellStep(ctx context.Context, jobRunID string, command string, timeout time.Duration) (executor.Result, error)
}

type LocalRunner struct {
	id       string
	executor executor.Executor
}

func NewLocalRunner(id string, exec executor.Executor) *LocalRunner {
	return &LocalRunner{id: id, executor: exec}
}

func (r *LocalRunner) ID() string {
	return r.id
}

func (r *LocalRunner) RunShellStep(ctx context.Context, jobRunID string, command string, timeout time.Duration) (executor.Result, error) {
	if err := r.executor.Prepare(ctx, executor.JobContext{JobRunID: jobRunID, RunnerID: r.id}); err != nil {
		return executor.Result{}, err
	}
	return r.executor.Run(ctx, executor.Command{
		Name:    "sh",
		Args:    []string{"-c", command},
		Timeout: timeout,
	})
}
