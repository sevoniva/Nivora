package ssh

import (
	"context"
	"fmt"

	portexecutor "github.com/sevoniva/nivora/internal/ports/executor"
)

type Executor struct{}

func New() Executor {
	return Executor{}
}

func (Executor) Prepare(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return blocked(ctx, request)
}

func (Executor) Upload(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return blocked(ctx, request)
}

func (Executor) Execute(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return blocked(ctx, request)
}

func (Executor) HealthCheck(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return blocked(ctx, request)
}

func (Executor) Rollback(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return blocked(ctx, request)
}

func blocked(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	select {
	case <-ctx.Done():
		return portexecutor.HostDeploymentResult{}, ctx.Err()
	default:
	}
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Rejected", Message: "remote SSH host deployment is disabled in Phase 3.5"}, fmt.Errorf("remote SSH host deployment is disabled in Phase 3.5")
}
