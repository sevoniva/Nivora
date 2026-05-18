package runtime

import (
	"context"

	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	shellexecutor "github.com/sevoniva/nivora/internal/adapters/executor/shell"
	yamlapply "github.com/sevoniva/nivora/internal/adapters/executor/yaml_apply"
	"github.com/sevoniva/nivora/internal/ports/policy"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func NewPipelineService() *pipelineusecase.Service {
	store := pipelineusecase.NewMemoryStore()
	bus := memory.New()
	runner := pipelineusecase.NewLocalRunner("local-runner", shellexecutor.New())
	return pipelineusecase.NewService(store, runner, bus)
}

func NewDeploymentService() *deploymentusecase.Service {
	store := deploymentusecase.NewMemoryStore()
	bus := memory.New()
	return deploymentusecase.NewService(
		store,
		deploymentusecase.NewStaticManifestRenderer(),
		yamlapply.NoopManifestClient{},
		allowAllPolicyEngine{},
		bus,
	)
}

type allowAllPolicyEngine struct{}

func (allowAllPolicyEngine) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	select {
	case <-ctx.Done():
		return policy.Result{}, ctx.Err()
	default:
		return policy.Result{Allowed: true, Reasons: []string{"Phase 2.1 allow-all policy placeholder"}}, nil
	}
}
