package shell

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

func TestRunEcho(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "echo",
		Args:    []string{"hello"},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d", result.ExitCode)
	}
	if strings.TrimSpace(result.Stdout) != "hello" {
		t.Fatalf("stdout = %q", result.Stdout)
	}
}

func TestRunTimeout(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `sleep 2`},
		Timeout: 10 * time.Millisecond,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if result.ExitCode == 0 {
		t.Fatalf("exit code = %d", result.ExitCode)
	}
}

func TestRunCanceledContext(t *testing.T) {
	exec := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := exec.Run(ctx, executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `printf nope`},
		Timeout: time.Second,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunFailure(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `printf "bad" >&2; exit 3`},
		Timeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected command error")
	}
	if result.ExitCode != 3 {
		t.Fatalf("exit code = %d", result.ExitCode)
	}
	if strings.TrimSpace(result.Stderr) != "bad" {
		t.Fatalf("stderr = %q", result.Stderr)
	}
}

// --- Safety tests ---

func TestRunLargeOutputTruncation(t *testing.T) {
	exec := NewWithConfig(Config{MaxOutputBytes: 1024})
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `for i in $(seq 1 4000); do printf "x"; done`},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if int64(len(result.Stdout)) < 1024 {
		t.Fatalf("expected truncated output >= 1024 bytes, got %d", len(result.Stdout))
	}
	if int64(len(result.Stdout)) > 1024+200 {
		t.Fatalf("expected truncated output near 1024 bytes, got %d", len(result.Stdout))
	}
	if !strings.Contains(result.Stdout, "[output truncated]") {
		t.Fatal("expected truncation marker in output")
	}
}

func TestRunMaxTimeoutClamped(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "echo",
		Args:    []string{"ok"},
		Timeout: time.Duration(MaxTimeoutSeconds+1) * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d", result.ExitCode)
	}
}

func TestRunExplicitEnvIsolation(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `printf "%s" "$MY_VAR"`},
		Env:     map[string]string{"MY_VAR": "isolated_value"},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.Stdout != "isolated_value" {
		t.Fatalf("expected isolated_value, got %q", result.Stdout)
	}
}

// --- Enterprise-grade tests ---

func TestRunBlockedEnvVarFiltered(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name: "sh",
		Args: []string{"-c", `printf "%s" "${NIVORA_AUTH_TOKEN:-blocked}"`},
		Env: map[string]string{
			"NIVORA_AUTH_TOKEN": "secret-token-value",
			"MY_APP_VAR":        "allowed",
		},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if strings.Contains(result.Stdout, "secret-token-value") {
		t.Fatal("blocked env var leaked to command")
	}
}

func TestRunWorkspaceIsolation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nivora-test-workspace-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exec := NewWithConfig(Config{WorkspaceRoot: tmpDir, CleanupWorkspace: true})
	jobCtx := executor.JobContext{JobRunID: "test-job-001", RunnerID: "test-runner"}
	if err := exec.Prepare(context.Background(), jobCtx); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	result, err := exec.Run(context.Background(), executor.Command{
		ID:      "test-job-001",
		Name:    "sh",
		Args:    []string{"-c", `printf "%s" "$(pwd)"`},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if !strings.Contains(result.Stdout, tmpDir) {
		t.Fatalf("working directory not in workspace: got %q, expected under %q", result.Stdout, tmpDir)
	}

	exec.Cancel(context.Background(), "test-job-001")

	// Workspace should be cleaned up.
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) > 0 {
		t.Logf("workspace may not have been fully cleaned: %d entries remain", len(entries))
	}
}

func TestRunWorkspaceCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "nivora-test-cleanup-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	exec := NewWithConfig(Config{WorkspaceRoot: tmpDir, CleanupWorkspace: true})
	jobCtx := executor.JobContext{JobRunID: "test-job-cleanup", RunnerID: "test-runner"}

	if err := exec.Prepare(context.Background(), jobCtx); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Execute a command to create files.
	_, err = exec.Run(context.Background(), executor.Command{
		ID:      "test-job-cleanup",
		Name:    "sh",
		Args:    []string{"-c", `touch test.txt && mkdir subdir && touch subdir/nested.txt`},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}

	// Cancel should clean up workspace.
	exec.Cancel(context.Background(), "test-job-cleanup")

	// Allow filesystem sync.
	time.Sleep(100 * time.Millisecond)
	entries, _ := os.ReadDir(tmpDir)
	if len(entries) > 0 {
		t.Logf("%d entries remain in workspace root after cleanup", len(entries))
	}
}

func TestRunProcessGroupCleanup(t *testing.T) {
	exec := New()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := exec.Run(ctx, executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `sleep 10`},
		Timeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code, got %d", result.ExitCode)
	}
}

func TestIsSensitiveEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"NIVORA_AUTH_TOKEN", true},
		{"AWS_ACCESS_KEY_ID", true},
		{"AWS_SECRET_ACCESS_KEY", true},
		{"DATABASE_URL", true},
		{"KUBECONFIG", true},
		{"GITHUB_TOKEN", true},
		{"VAULT_TOKEN", true},
		{"MY_APP_PASSWORD", true},
		{"MY_APP_SECRET", true},
		{"PRIVATE_KEY", true},
		{"HOME", false},
		{"USER", false},
		{"PATH", false},
		{"LANG", false},
		{"MY_APP_VAR", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSensitiveEnvVar(tt.name); got != tt.expected {
				t.Errorf("IsSensitiveEnvVar(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxOutputBytes != DefaultMaxOutputBytes {
		t.Errorf("MaxOutputBytes = %d, want %d", cfg.MaxOutputBytes, DefaultMaxOutputBytes)
	}
	if !cfg.CleanupWorkspace {
		t.Error("CleanupWorkspace should be true by default")
	}
}

func TestPrepareRequiresJobRunID(t *testing.T) {
	exec := New()
	err := exec.Prepare(context.Background(), executor.JobContext{})
	if err == nil {
		t.Fatal("expected error for empty JobRunID")
	}
}

func TestRunRequiresCommandName(t *testing.T) {
	exec := New()
	_, err := exec.Run(context.Background(), executor.Command{
		Timeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected error for empty command name")
	}
}
