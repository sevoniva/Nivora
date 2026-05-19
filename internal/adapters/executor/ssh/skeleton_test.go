package ssh

import (
	"context"
	"errors"
	"strings"
	"testing"

	portexecutor "github.com/sevoniva/nivora/internal/ports/executor"
)

func TestExecutorRejectsUnguardedRemoteExecution(t *testing.T) {
	executor := NewWithRunner(fakeRunner{})
	_, err := executor.Execute(context.Background(), portexecutor.HostDeploymentRequest{
		HostID:     "host-1",
		HostName:   "host-1",
		Address:    "127.0.0.1",
		DeployPath: "/opt/app",
		ReleaseDir: "/opt/app/releases/run",
	})
	if err == nil {
		t.Fatal("expected guarded SSH execution rejection")
	}
}

func TestExecutorRunsUploadExecuteHealthAndRollbackThroughRunner(t *testing.T) {
	runner := &recordingRunner{}
	executor := NewWithRunner(runner)
	request := guardedRequest()
	if _, err := executor.Prepare(context.Background(), request); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if _, err := executor.Upload(context.Background(), request); err != nil {
		t.Fatalf("upload: %v", err)
	}
	if _, err := executor.Execute(context.Background(), request); err != nil {
		t.Fatalf("execute: %v", err)
	}
	request.HealthCheckType = "http"
	request.HealthCheck = "http://127.0.0.1:8080/healthz"
	if _, err := executor.HealthCheck(context.Background(), request); err != nil {
		t.Fatalf("health: %v", err)
	}
	if _, err := executor.Rollback(context.Background(), request); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if len(runner.commands) != 4 {
		t.Fatalf("commands = %#v", runner.commands)
	}
	if len(runner.uploads) != 1 || runner.uploads[0].Destination != "/opt/app/releases/run/artifact" {
		t.Fatalf("uploads = %#v", runner.uploads)
	}
	if !strings.Contains(runner.commands[1].Command, "systemctl restart 'demo'") {
		t.Fatalf("execute command = %s", runner.commands[1].Command)
	}
}

func TestExecutorRedactsRunnerErrors(t *testing.T) {
	executor := NewWithRunner(fakeRunner{err: errors.New("password token secret leaked")})
	_, err := executor.Execute(context.Background(), guardedRequest())
	if err == nil {
		t.Fatal("expected runner error")
	}
	if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "token") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("error was not redacted: %v", err)
	}
}

func guardedRequest() portexecutor.HostDeploymentRequest {
	return portexecutor.HostDeploymentRequest{
		DeploymentRunID: "drun-test",
		HostID:          "host-1",
		HostName:        "host-1",
		Address:         "127.0.0.1",
		Artifact:        "./dist/demo.tar.gz",
		DeployPath:      "/opt/app",
		ReleaseDir:      "/opt/app/releases/run",
		ServiceName:     "demo",
		Strategy:        "symlink",
		BatchIndex:      1,
		Apply:           true,
		Confirmed:       true,
		AllowRemote:     true,
		CredentialRef:   "cred-host",
		TimeoutSeconds:  5,
	}
}

type recordingRunner struct {
	commands []CommandRequest
	uploads  []UploadRequest
}

func (r *recordingRunner) Run(ctx context.Context, request CommandRequest) (portexecutor.HostDeploymentResult, error) {
	r.commands = append(r.commands, request)
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Succeeded", Message: "ok", Command: request.Command}, nil
}

func (r *recordingRunner) Upload(ctx context.Context, request UploadRequest) (portexecutor.HostDeploymentResult, error) {
	r.uploads = append(r.uploads, request)
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Succeeded", Message: "uploaded"}, nil
}

type fakeRunner struct {
	err error
}

func (r fakeRunner) Run(ctx context.Context, request CommandRequest) (portexecutor.HostDeploymentResult, error) {
	if r.err != nil {
		return portexecutor.HostDeploymentResult{}, r.err
	}
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Succeeded", Message: "ok"}, nil
}

func (r fakeRunner) Upload(ctx context.Context, request UploadRequest) (portexecutor.HostDeploymentResult, error) {
	if r.err != nil {
		return portexecutor.HostDeploymentResult{}, r.err
	}
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Succeeded", Message: "uploaded"}, nil
}
