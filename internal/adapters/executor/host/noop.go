package host

import (
	"context"

	portexecutor "github.com/sevoniva/nivora/internal/ports/executor"
)

type NoopExecutor struct{}

func NewNoop() NoopExecutor {
	return NoopExecutor{}
}

func (NoopExecutor) Prepare(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return result(ctx, request, "prepared host deployment plan")
}

func (NoopExecutor) Upload(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return result(ctx, request, "upload skipped by noop host executor")
}

func (NoopExecutor) Execute(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return result(ctx, request, "execution skipped by noop host executor")
}

func (NoopExecutor) HealthCheck(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return result(ctx, request, "health check completed by noop host executor")
}

func (NoopExecutor) Rollback(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return result(ctx, request, "rollback skipped by noop host executor")
}

func result(ctx context.Context, request portexecutor.HostDeploymentRequest, message string) (portexecutor.HostDeploymentResult, error) {
	select {
	case <-ctx.Done():
		return portexecutor.HostDeploymentResult{}, ctx.Err()
	default:
	}
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Succeeded", Message: message}, nil
}
