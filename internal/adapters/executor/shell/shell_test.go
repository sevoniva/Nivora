package shell

import (
	"context"
	"errors"
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
		Env:     map[string]string{"PATH": "/bin:/usr/bin", "MY_VAR": "isolated_value"},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run command: %v", err)
	}
	if result.Stdout != "isolated_value" {
		t.Fatalf("expected isolated_value, got %q", result.Stdout)
	}
}
