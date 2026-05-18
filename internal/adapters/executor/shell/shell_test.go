package shell

import (
	"context"
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
