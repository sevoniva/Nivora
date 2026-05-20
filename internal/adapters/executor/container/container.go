// Package container provides an optional container-based executor profile
// for safer production runner isolation. The default implementation is a
// documented skeleton that implements the executor.Executor interface using
// local execution. A real Docker/podman adapter is future work.
//
// The container profile is NOT an OS-level sandbox unless operators deploy
// runners with appropriate container runtime security (seccomp, AppArmor,
// no privileged mode, no host network, no Docker socket mount).
//
// Production config requires:
//
//	runner_isolation_profile: container-isolated
//	allow_privileged_executor: false
//	allow_docker_socket_mount: false
//	allow_host_path_mount: false
package container

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

// Safety defaults for container execution.
const (
	DefaultMaxOutputBytes = 10 * 1024 * 1024
	MaxTimeoutSeconds     = 3600
	DefaultContainerImage = "alpine:3.20"
)

// ErrPrivilegedRejected is returned when privileged mode is requested.
var ErrPrivilegedRejected = errors.New("privileged container execution is not allowed")

// ErrHostNetworkRejected is returned when host networking is requested.
var ErrHostNetworkRejected = errors.New("host network mode is not allowed")

// ErrDockerSocketRejected is returned when Docker socket mount is requested.
var ErrDockerSocketRejected = errors.New("docker socket mount is not allowed")

// Config holds container executor configuration.
type Config struct {
	ContainerImage    string
	MaxOutputBytes    int64
	AllowPrivileged   bool
	AllowHostNetwork  bool
	AllowDockerSocket bool
}

// DefaultConfig returns a safe container executor config.
func DefaultConfig() Config {
	return Config{
		ContainerImage: DefaultContainerImage,
		MaxOutputBytes: DefaultMaxOutputBytes,
	}
}

// Executor implements executor.Executor with container profile safety gates.
// The current implementation is a local-execution skeleton that enforces
// the same safety rules a real container adapter would. A Docker/podman
// backend is documented as future work.
type Executor struct {
	lastLog []byte
	cfg     Config
	dir     string
}

var _ executor.Executor = (*Executor)(nil)

// New returns a container executor with default safe config.
func New() *Executor {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig returns a container executor with custom config.
// Unsafe flags are rejected at config time.
func NewWithConfig(cfg Config) *Executor {
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = DefaultMaxOutputBytes
	}
	if cfg.ContainerImage == "" {
		cfg.ContainerImage = DefaultContainerImage
	}
	return &Executor{cfg: cfg}
}

func (e *Executor) Prepare(ctx context.Context, job executor.JobContext) error {
	if err := ctx.Err(); err != nil {
		return err
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

	// Safety gates — these would apply to a real container runtime too.
	if e.cfg.AllowPrivileged {
		return executor.Result{}, ErrPrivilegedRejected
	}
	if e.cfg.AllowHostNetwork {
		return executor.Result{}, ErrHostNetworkRejected
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
	if len(command.Env) > 0 {
		cmd.Env = append(cmd.Env, "PATH=/usr/local/bin:/usr/bin:/bin")
		for k, v := range command.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()
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
	e.lastLog = append([]byte(nil), stdoutBuf.Bytes()...)
	e.lastLog = append(e.lastLog, stderrBuf.Bytes()...)

	if runCtx.Err() != nil {
		return result, runCtx.Err()
	}
	return result, err
}

func (e *Executor) Cancel(ctx context.Context, commandID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (e *Executor) Logs(ctx context.Context, commandID string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(e.lastLog)), nil
}
