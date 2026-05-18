package executor

import (
	"context"
	"io"
	"time"
)

type JobContext struct {
	JobRunID string
	RunnerID string
}

type Command struct {
	ID         string
	Name       string
	Args       []string
	WorkingDir string
	Env        map[string]string
	Timeout    time.Duration
}

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

type Executor interface {
	Prepare(ctx context.Context, job JobContext) error
	Run(ctx context.Context, command Command) (Result, error)
	Cancel(ctx context.Context, commandID string) error
	Logs(ctx context.Context, commandID string) (io.ReadCloser, error)
}
