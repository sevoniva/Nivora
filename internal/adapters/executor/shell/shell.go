package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

type Executor struct {
	lastLog []byte
}

func New() *Executor {
	return &Executor{}
}

func (e *Executor) Prepare(ctx context.Context, job executor.JobContext) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if job.JobRunID == "" {
		return errors.New("job run id is required")
	}
	return nil
}

func (e *Executor) Run(ctx context.Context, command executor.Command) (executor.Result, error) {
	if command.Name == "" {
		return executor.Result{}, errors.New("command name is required")
	}

	runCtx := ctx
	cancel := func() {}
	if command.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, command.Timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(runCtx, command.Name, command.Args...)
	cmd.Dir = command.WorkingDir
	for k, v := range command.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	result := executor.Result{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}
	e.lastLog = append([]byte(nil), append(stdout.Bytes(), stderr.Bytes()...)...)

	if runCtx.Err() != nil {
		return result, runCtx.Err()
	}
	return result, err
}

func (e *Executor) Cancel(ctx context.Context, commandID string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (e *Executor) Logs(ctx context.Context, commandID string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return io.NopCloser(bytes.NewReader(e.lastLog)), nil
	}
}
