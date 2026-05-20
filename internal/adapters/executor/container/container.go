// Package container provides a container-based executor profile for safer
// production runner isolation. Auto-detects docker or podman at runtime.
// Falls back to local execution skeleton if no container runtime is available.
//
// The container profile is NOT an OS-level sandbox unless operators deploy
// runners with appropriate container runtime security (seccomp, AppArmor,
// no privileged mode, no host network, no Docker socket mount).
package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

const (
	DefaultMaxOutputBytes = 10 * 1024 * 1024
	MaxTimeoutSeconds     = 3600
	DefaultContainerImage = "alpine:3.20"
	DefaultMemoryLimit    = "512m"
	DefaultCPULimit       = "1"
)

var (
	ErrPrivilegedRejected  = errors.New("privileged container execution is not allowed")
	ErrHostNetworkRejected = errors.New("host network mode is not allowed")
	ErrNoContainerRuntime  = errors.New("no container runtime (docker/podman) found")
)

type Config struct {
	ContainerImage    string
	MaxOutputBytes    int64
	MemoryLimit       string
	CPULimit          string
	AllowPrivileged   bool
	AllowHostNetwork  bool
	AllowDockerSocket bool
	WorkspaceRoot     string
	ExtraRunFlags     []string
}

func DefaultConfig() Config {
	return Config{
		ContainerImage: DefaultContainerImage,
		MaxOutputBytes: DefaultMaxOutputBytes,
		MemoryLimit:    DefaultMemoryLimit,
		CPULimit:       DefaultCPULimit,
	}
}

type Executor struct {
	lastLog    []byte
	cfg        Config
	runtime    string
	workspaces map[string]string
}

var _ executor.Executor = (*Executor)(nil)

func New() *Executor {
	return NewWithConfig(DefaultConfig())
}

// NewLocal returns a container executor that runs locally without Docker/podman.
// Useful for testing when container runtime is unavailable.
func NewLocal() *Executor {
	e := NewWithConfig(DefaultConfig())
	e.runtime = ""
	return e
}

func NewWithConfig(cfg Config) *Executor {
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = DefaultMaxOutputBytes
	}
	if cfg.ContainerImage == "" {
		cfg.ContainerImage = DefaultContainerImage
	}
	if cfg.MemoryLimit == "" {
		cfg.MemoryLimit = DefaultMemoryLimit
	}
	if cfg.CPULimit == "" {
		cfg.CPULimit = DefaultCPULimit
	}
	return &Executor{
		cfg:        cfg,
		runtime:    detectRuntime(),
		workspaces: make(map[string]string),
	}
}

func detectRuntime() string {
	for _, bin := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(bin); err == nil {
			// Verify the runtime is actually functional.
			if err := exec.Command(bin, "version").Run(); err == nil {
				return bin
			}
		}
	}
	return ""
}

func (e *Executor) hasRuntime() bool     { return e.runtime != "" }
func (e *Executor) RuntimeName() string  { return e.runtime }
func (e *Executor) UsesContainers() bool { return e.runtime != "" }

func (e *Executor) Prepare(ctx context.Context, job executor.JobContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if job.JobRunID == "" {
		return errors.New("job run id is required")
	}

	// Create isolated workspace directory if configured.
	if e.cfg.WorkspaceRoot != "" {
		dir, err := os.MkdirTemp(e.cfg.WorkspaceRoot, "nivora-job-*")
		if err != nil {
			return fmt.Errorf("create workspace: %w", err)
		}
		e.workspaces[job.JobRunID] = dir
	}
	return nil
}

func (e *Executor) Run(ctx context.Context, command executor.Command) (executor.Result, error) {
	if command.Name == "" {
		return executor.Result{}, errors.New("command name is required")
	}
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

	// Use real container runtime if available.
	if e.hasRuntime() {
		return e.runContainerized(runCtx, command)
	}
	return e.runLocal(runCtx, command)
}

func (e *Executor) runContainerized(ctx context.Context, command executor.Command) (executor.Result, error) {
	workspace := e.workspaces[command.ID]
	if workspace == "" && e.cfg.WorkspaceRoot != "" {
		workspace = command.WorkingDir
	}
	if workspace == "" {
		var err error
		workspace, err = os.MkdirTemp("", "nivora-container-workspace-*")
		if err != nil {
			return executor.Result{}, fmt.Errorf("create temp workspace: %w", err)
		}
		defer os.RemoveAll(workspace)
	}

	// Build docker/podman run arguments.
	args := []string{"run", "--rm"}

	// Safety flags.
	args = append(args,
		"--read-only",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--network=none",
		"--user=65532:65532",
		"--memory="+e.cfg.MemoryLimit,
		"--cpus="+e.cfg.CPULimit,
		"--tmpfs=/tmp:rw,noexec,nosuid,size=256m",
	)

	// Workspace mount.
	workDir := "/workspace"
	args = append(args, "--tmpfs="+workDir+":rw,noexec,nosuid,size=1g")
	args = append(args, "--workdir="+workDir)
	args = append(args, "-v", workspace+":"+workDir+":rw")

	// Environment variables.
	if len(command.Env) > 0 {
		for k, v := range command.Env {
			args = append(args, "-e", k+"="+v)
		}
	}

	// Extra flags from config.
	args = append(args, e.cfg.ExtraRunFlags...)

	// Image and command.
	args = append(args, e.cfg.ContainerImage)
	args = append(args, command.Name)
	args = append(args, command.Args...)

	cmd := exec.CommandContext(ctx, e.runtime, args...)
	cmd.Dir = command.WorkingDir

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

	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}

func (e *Executor) runLocal(ctx context.Context, command executor.Command) (executor.Result, error) {
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
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

	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, err
}

func (e *Executor) Cancel(ctx context.Context, commandID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Clean up workspace.
	workspace, ok := e.workspaces[commandID]
	if ok {
		_ = os.RemoveAll(workspace)
		delete(e.workspaces, commandID)
	}
	return nil
}

func (e *Executor) Logs(ctx context.Context, commandID string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(e.lastLog)), nil
}

func (e *Executor) Cleanup() {
	for _, dir := range e.workspaces {
		_ = os.RemoveAll(dir)
	}
	e.workspaces = make(map[string]string)
}

func (e *Executor) WorkspaceDir(commandID string) string {
	return e.workspaces[commandID]
}
