package container

import (
	"context"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

func TestContainerExecEcho(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "echo",
		Args:    []string{"container-test"},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d", result.ExitCode)
	}
}

func TestContainerExecRejectsPrivileged(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowPrivileged = true
	exec := NewWithConfig(cfg)
	_, err := exec.Run(context.Background(), executor.Command{
		Name:    "echo",
		Args:    []string{"should-not-run"},
		Timeout: time.Second,
	})
	if err != ErrPrivilegedRejected {
		t.Fatalf("expected ErrPrivilegedRejected, got %v", err)
	}
}

func TestContainerExecRejectsHostNetwork(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowHostNetwork = true
	exec := NewWithConfig(cfg)
	_, err := exec.Run(context.Background(), executor.Command{
		Name:    "echo",
		Args:    []string{"should-not-run"},
		Timeout: time.Second,
	})
	if err != ErrHostNetworkRejected {
		t.Fatalf("expected ErrHostNetworkRejected, got %v", err)
	}
}

func TestContainerExecTimeout(t *testing.T) {
	exec := New()
	_, err := exec.Run(context.Background(), executor.Command{
		Name:    "sleep",
		Args:    []string{"10"},
		Timeout: 10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestContainerExecOutputTruncation(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MaxOutputBytes = 512
	exec := NewWithConfig(cfg)
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `for i in $(seq 1 2000); do printf "x"; done`},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if int64(len(result.Stdout)) < 512 {
		t.Fatalf("expected truncated output >= 512, got %d", len(result.Stdout))
	}
	if int64(len(result.Stdout)) > 512+100 {
		t.Fatalf("expected truncated output near 512, got %d", len(result.Stdout))
	}
}

func TestContainerExecRequiresCommandName(t *testing.T) {
	exec := New()
	_, err := exec.Run(context.Background(), executor.Command{Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for empty command name")
	}
}

func TestContainerExecPrepareRequiresJobRunID(t *testing.T) {
	exec := New()
	err := exec.Prepare(context.Background(), executor.JobContext{})
	if err == nil {
		t.Fatal("expected error for empty JobRunID")
	}
}

func TestContainerExecCanceledContext(t *testing.T) {
	exec := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := exec.Run(ctx, executor.Command{
		Name:    "echo",
		Args:    []string{"should-not-run"},
		Timeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected context.Canceled error")
	}
}

func TestDefaultConfigIsSafe(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.AllowPrivileged {
		t.Fatal("AllowPrivileged must be false in default config")
	}
	if cfg.AllowHostNetwork {
		t.Fatal("AllowHostNetwork must be false in default config")
	}
	if cfg.AllowDockerSocket {
		t.Fatal("AllowDockerSocket must be false in default config")
	}
	if cfg.ContainerImage != DefaultContainerImage {
		t.Fatalf("expected %s, got %s", DefaultContainerImage, cfg.ContainerImage)
	}
	if cfg.MaxOutputBytes != DefaultMaxOutputBytes {
		t.Fatalf("expected %d, got %d", DefaultMaxOutputBytes, cfg.MaxOutputBytes)
	}
}

func TestContainerExecImplementsInterface(t *testing.T) {
	var _ executor.Executor = New()
}

func TestContainerExecLogs(t *testing.T) {
	exec := New()
	_, err := exec.Run(context.Background(), executor.Command{
		Name:    "echo",
		Args:    []string{"log-test"},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	reader, err := exec.Logs(context.Background(), "test")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	defer reader.Close()
	buf := make([]byte, 256)
	n, _ := reader.Read(buf)
	if n <= 0 {
		t.Fatal("expected log content")
	}
}

func TestContainerExecEnvIsolation(t *testing.T) {
	exec := New()
	result, err := exec.Run(context.Background(), executor.Command{
		Name:    "sh",
		Args:    []string{"-c", `printf "%s" "$MY_VAR"`},
		Env:     map[string]string{"MY_VAR": "container-env-test"},
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Stdout != "container-env-test" {
		t.Fatalf("expected container-env-test, got %q", result.Stdout)
	}
}
