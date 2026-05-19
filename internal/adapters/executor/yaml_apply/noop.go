package yamlapply

import (
	"context"
	"fmt"

	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

type NoopManifestClient struct{}

func (NoopManifestClient) ServerDryRun(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesDryRunResult, error) {
	select {
	case <-ctx.Done():
		return deploymentusecase.KubernetesDryRunResult{}, ctx.Err()
	default:
	}
	if err := validateRequest(request); err != nil {
		return deploymentusecase.KubernetesDryRunResult{}, err
	}
	return deploymentusecase.KubernetesDryRunResult{
		Mode:      "noop",
		Message:   "server-side dry-run simulated by local no-op client",
		Resources: request.Plan.Resources,
	}, nil
}

func (NoopManifestClient) Apply(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesApplyResult, error) {
	select {
	case <-ctx.Done():
		return deploymentusecase.KubernetesApplyResult{}, ctx.Err()
	default:
	}
	if err := validateRequest(request); err != nil {
		return deploymentusecase.KubernetesApplyResult{}, err
	}
	return deploymentusecase.KubernetesApplyResult{
		Mode:      "noop",
		Message:   "apply simulated by local no-op client",
		Resources: request.Plan.Resources,
	}, nil
}

func (NoopManifestClient) WatchRollout(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.RolloutResult, error) {
	select {
	case <-ctx.Done():
		return deploymentusecase.RolloutResult{}, ctx.Err()
	default:
	}
	if err := validateRequest(request); err != nil {
		return deploymentusecase.RolloutResult{}, err
	}
	return deploymentusecase.RolloutResult{
		Mode:      "noop",
		Message:   "rollout verification simulated by local no-op client",
		Resources: request.Plan.Resources,
	}, nil
}

func (NoopManifestClient) Rollback(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesRollbackResult, error) {
	select {
	case <-ctx.Done():
		return deploymentusecase.KubernetesRollbackResult{}, ctx.Err()
	default:
	}
	if err := validateRequest(request); err != nil {
		return deploymentusecase.KubernetesRollbackResult{}, err
	}
	return deploymentusecase.KubernetesRollbackResult{
		Mode:      "noop",
		Message:   "rollback manifest restore simulated by local no-op client",
		Resources: request.Plan.Resources,
	}, nil
}

func validateRequest(request deploymentusecase.ManifestRequest) error {
	if request.Plan.DeploymentRunID == "" {
		return fmt.Errorf("deploymentRunId is required")
	}
	if len(request.Documents) == 0 {
		return fmt.Errorf("at least one manifest document is required")
	}
	return nil
}
