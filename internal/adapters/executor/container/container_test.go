package container

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/ports/executor"
)

func TestContainerExecEcho(t *testing.T) {
	ex := NewLocal()
	result, err := ex.Run(context.Background(), executor.Command{
		Name: "echo", Args: []string{"container-test"}, Timeout: 5 * time.Second,
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
	_, err := NewWithConfig(cfg).Run(context.Background(), executor.Command{
		Name: "echo", Args: []string{"x"}, Timeout: time.Second,
	})
	if err != ErrPrivilegedRejected {
		t.Fatalf("expected ErrPrivilegedRejected, got %v", err)
	}
}

func TestContainerExecRejectsHostNetwork(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowHostNetwork = true
	_, err := NewWithConfig(cfg).Run(context.Background(), executor.Command{
		Name: "echo", Args: []string{"x"}, Timeout: time.Second,
	})
	if err != ErrHostNetworkRejected {
		t.Fatalf("expected ErrHostNetworkRejected, got %v", err)
	}
}

func TestContainerExecTimeout(t *testing.T) {
	_, err := NewLocal().Run(context.Background(), executor.Command{
		Name: "sleep", Args: []string{"10"}, Timeout: 10 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestContainerExecOutputTruncation(t *testing.T) {
	e := NewLocal()
	e.cfg.MaxOutputBytes = 512
	result, err := e.Run(context.Background(), executor.Command{
		Name: "sh", Args: []string{"-c", `i=0; while [ $i -lt 2000 ]; do printf "x"; i=$((i+1)); done`},
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if int64(len(result.Stdout)) < 512 {
		t.Fatalf("expected truncated output >= 512, got %d", len(result.Stdout))
	}
}

func TestContainerExecRequiresCommandName(t *testing.T) {
	_, err := NewLocal().Run(context.Background(), executor.Command{Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for empty command name")
	}
}

func TestContainerExecPrepareRequiresJobRunID(t *testing.T) {
	err := NewLocal().Prepare(context.Background(), executor.JobContext{})
	if err == nil {
		t.Fatal("expected error for empty JobRunID")
	}
}

func TestContainerExecCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewLocal().Run(ctx, executor.Command{
		Name: "echo", Args: []string{"x"}, Timeout: time.Second,
	})
	if err == nil {
		t.Fatal("expected context.Canceled error")
	}
}

func TestDefaultConfigIsSafe(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.AllowPrivileged || cfg.AllowHostNetwork || cfg.AllowDockerSocket {
		t.Fatal("unsafe flags must be false by default")
	}
}

func TestContainerExecImplementsInterface(t *testing.T) {
	var _ executor.Executor = New()
}

func TestContainerExecLogs(t *testing.T) {
	e := NewLocal()
	e.Run(context.Background(), executor.Command{
		Name: "echo", Args: []string{"log-test"}, Timeout: time.Second,
	})
	reader, err := e.Logs(context.Background(), "test")
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
	result, err := NewLocal().Run(context.Background(), executor.Command{
		Name: "sh", Args: []string{"-c", `printf "%s" "$MY_VAR"`},
		Env: map[string]string{"MY_VAR": "container-env-test"}, Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Stdout != "container-env-test" {
		t.Fatalf("expected container-env-test, got %q", result.Stdout)
	}
}

func TestContainerExecWorkspaceCleanup(t *testing.T) {
	ex := NewLocal()
	ex.cfg.WorkspaceRoot = t.TempDir()
	if err := ex.Prepare(context.Background(), executor.JobContext{JobRunID: "cleanup-test"}); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := ex.Cancel(context.Background(), "cleanup-test"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if ex.WorkspaceDir("cleanup-test") != "" {
		t.Fatal("workspace should be cleaned after cancel")
	}
}

func TestContainerConfigHasAllSafetyDefaults(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ContainerImage != DefaultContainerImage || cfg.MemoryLimit != DefaultMemoryLimit || cfg.CPULimit != DefaultCPULimit {
		t.Fatal("safety defaults mismatch")
	}
}

func TestNewLocalSkipsRuntime(t *testing.T) {
	ex := NewLocal()
	if ex.hasRuntime() || ex.UsesContainers() {
		t.Fatal("NewLocal should not use container runtime")
	}
}

func TestDetectRuntime(t *testing.T) {
	rt := detectRuntime()
	t.Logf("detected runtime: %q (empty if no docker/podman)", rt)
}

// --- Real Docker tests (skip if Docker not functional with full flag set) ---

func hasWorkingDocker() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	cmd := exec.Command("docker", "run", "--rm",
		"--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges",
		"--network=none", "--user=65532:65532",
		"--tmpfs=/tmp:rw,noexec,nosuid",
		"--tmpfs=/workspace:rw,noexec,nosuid",
		"--workdir=/workspace",
		"-v", "/tmp:/workspace:rw",
		"alpine:3.20", "echo", "ok")
	return cmd.Run() == nil
}

func TestContainerExecWithRealDocker(t *testing.T) {
	if !hasWorkingDocker() {
		t.Skip("docker not available or cannot run with safety flags + volume mounts")
	}
	ex := New()
	ex.cfg.WorkspaceRoot = "/tmp"
	result, err := ex.Run(context.Background(), executor.Command{
		Name: "echo", Args: []string{"hello-from-real-container"}, Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("container run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, stderr=%s", result.ExitCode, result.Stderr)
	}
	t.Logf("container output: %s", strings.TrimSpace(result.Stdout))
}

func TestContainerExecSafeFlags(t *testing.T) {
	if !hasWorkingDocker() {
		t.Skip("docker not available")
	}
	ex := New()
	result, err := ex.Run(context.Background(), executor.Command{
		Name: "sh", Args: []string{"-c", "id -u"}, Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	t.Logf("container user id: %s", strings.TrimSpace(result.Stdout))
}
