package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

const DefaultMaxOutputBytes = 10 * 1024 * 1024
const MaxTimeoutSeconds = 3600

type Config struct {
	MaxOutputBytes int64
}

type Executor struct {
	lastLog []byte
	cfg     Config
}

func New() *Executor {
	return NewWithConfig(Config{})
}

func NewWithConfig(cfg Config) *Executor {
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = DefaultMaxOutputBytes
	}
	return &Executor{cfg: cfg}
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
		timeout := command.Timeout
		if timeout.Seconds() > MaxTimeoutSeconds {
			timeout = time.Duration(MaxTimeoutSeconds) * time.Second
		}
		runCtx, cancel = context.WithTimeout(ctx, timeout)
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

	stdoutStr := stdout.String()
	stderrStr := stderr.String()
	if int64(len(stdoutStr)) > e.cfg.MaxOutputBytes {
		stdoutStr = stdoutStr[:e.cfg.MaxOutputBytes] + "\n[output truncated]"
	}
	if int64(len(stderrStr)) > e.cfg.MaxOutputBytes {
		stderrStr = stderrStr[:e.cfg.MaxOutputBytes] + "\n[output truncated]"
	}

	result := executor.Result{
		ExitCode: exitCode,
		Stdout:   stdoutStr,
		Stderr:   stderrStr,
	}
	e.lastLog = append([]byte(nil), append(stdout.Bytes(), stderr.Bytes()...)...)

	if runCtx.Err() != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return result, fmt.Errorf("command timed out after %v: %w", command.Timeout, runCtx.Err())
		}
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
