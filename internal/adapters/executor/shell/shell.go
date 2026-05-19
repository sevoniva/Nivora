package shell

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

const (
	DefaultMaxOutputBytes = 10 * 1024 * 1024
	MaxTimeoutSeconds     = 3600
	DefaultWorkspaceRoot  = ""
)

// Sensitive environment variable names that must never be passed to executor commands.
var blockedEnvVars = map[string]bool{
	"NIVORA_AUTH_TOKEN":          true,
	"NIVORA_DB_PASSWORD":         true,
	"DATABASE_URL":               true,
	"AWS_ACCESS_KEY_ID":          true,
	"AWS_SECRET_ACCESS_KEY":      true,
	"AWS_SESSION_TOKEN":          true,
	"ALICLOUD_ACCESS_KEY_ID":     true,
	"ALICLOUD_ACCESS_KEY_SECRET": true,
	"TENCENTCLOUD_SECRET_ID":     true,
	"TENCENTCLOUD_SECRET_KEY":    true,
	"KUBECONFIG":                 true,
	"KUBERNETES_SERVICE_HOST":    true,
	"GITHUB_TOKEN":               true,
	"GITLAB_TOKEN":               true,
	"DOCKER_HOST":                true,
	"VAULT_TOKEN":                true,
	"PASSWORD":                   true,
	"SECRET":                     true,
	"TOKEN":                      true,
	"CREDENTIAL":                 true,
	"PRIVATE_KEY":                true,
}

type Config struct {
	MaxOutputBytes     int64
	WorkspaceRoot      string
	AllowInheritEnv    bool
	CleanupWorkspace   bool
	AllowNetworkAccess bool
}

func DefaultConfig() Config {
	return Config{
		MaxOutputBytes:     DefaultMaxOutputBytes,
		CleanupWorkspace:   true,
		AllowNetworkAccess: true,
	}
}

type Executor struct {
	lastLog    []byte
	cfg        Config
	workspaces map[string]string
}

func New() *Executor {
	return NewWithConfig(DefaultConfig())
}

func NewWithConfig(cfg Config) *Executor {
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = DefaultMaxOutputBytes
	}
	return &Executor{
		cfg:        cfg,
		workspaces: make(map[string]string),
	}
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

	// Create isolated workspace directory.
	var workspace string
	if e.cfg.WorkspaceRoot != "" {
		dir, err := os.MkdirTemp(e.cfg.WorkspaceRoot, fmt.Sprintf("nivora-job-%s-*", job.JobRunID))
		if err != nil {
			return fmt.Errorf("create workspace: %w", err)
		}
		workspace = dir
		e.workspaces[job.JobRunID] = workspace
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

	// Process group for clean kill of children on timeout/cancel.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Working directory: prefer explicit workspace, then command dir.
	cmd.Dir = command.WorkingDir
	workspace, hasWorkspace := e.workspaces[command.ID]
	if hasWorkspace {
		cmd.Dir = workspace
	}

	// Environment handling.
	if len(command.Env) > 0 {
		// Explicit env: only the specified vars (plus filtered parent env if allowed).
		if e.cfg.AllowInheritEnv && !hasWorkspace {
			for _, ev := range os.Environ() {
				parts := strings.SplitN(ev, "=", 2)
				if len(parts) == 2 && !blockedEnvVars[strings.ToUpper(parts[0])] {
					cmd.Env = append(cmd.Env, ev)
				}
			}
		}
		cmd.Env = append(cmd.Env, "PATH=/usr/local/bin:/usr/bin:/bin")
		for k, v := range command.Env {
			if blockedEnvVars[strings.ToUpper(k)] {
				continue
			}
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	} else if !e.cfg.AllowInheritEnv {
		cmd.Env = []string{"PATH=/usr/local/bin:/usr/bin:/bin"}
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}

	// Kill process group if timeout/cancel to clean up orphaned children.
	if runCtx.Err() != nil && cmd.Process != nil {
		pgid, pgidErr := syscall.Getpgid(cmd.Process.Pid)
		if pgidErr == nil {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		}
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
	}

	// Clean up workspace.
	if e.cfg.CleanupWorkspace {
		workspace, ok := e.workspaces[commandID]
		if ok {
			_ = os.RemoveAll(workspace)
			delete(e.workspaces, commandID)
		}
	}
	return nil
}

func (e *Executor) Logs(ctx context.Context, commandID string) (io.ReadCloser, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return io.NopCloser(bytes.NewReader(e.lastLog)), nil
	}
}

// CleanupWorkspaces removes all created workspace directories.
func (e *Executor) CleanupWorkspaces() {
	for _, dir := range e.workspaces {
		_ = os.RemoveAll(dir)
	}
	e.workspaces = make(map[string]string)
}

// IsSensitiveEnvVar reports whether an environment variable name matches blocked patterns.
func IsSensitiveEnvVar(name string) bool {
	upper := strings.ToUpper(name)
	if blockedEnvVars[upper] {
		return true
	}
	for blocked := range blockedEnvVars {
		if strings.Contains(upper, blocked) {
			return true
		}
	}
	return false
}
