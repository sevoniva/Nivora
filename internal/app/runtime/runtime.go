package runtime

import (
	"context"

	ociartifact "github.com/sevoniva/nivora/internal/adapters/artifact/oci"
	"github.com/sevoniva/nivora/internal/adapters/cloud/aliyun"
	"github.com/sevoniva/nivora/internal/adapters/cloud/aws"
	cloudfake "github.com/sevoniva/nivora/internal/adapters/cloud/fake"
	"github.com/sevoniva/nivora/internal/adapters/cloud/tencent"
	"github.com/sevoniva/nivora/internal/adapters/eventbus/memory"
	argocdadapter "github.com/sevoniva/nivora/internal/adapters/executor/argocd"
	hostexecutor "github.com/sevoniva/nivora/internal/adapters/executor/host"
	shellexecutor "github.com/sevoniva/nivora/internal/adapters/executor/shell"
	yamlapply "github.com/sevoniva/nivora/internal/adapters/executor/yaml_apply"
	localgitops "github.com/sevoniva/nivora/internal/adapters/gitops/local"
	noopnotification "github.com/sevoniva/nivora/internal/adapters/notification/noop"
	builtinsecret "github.com/sevoniva/nivora/internal/adapters/secret/builtin"
	securitynoop "github.com/sevoniva/nivora/internal/adapters/security/noop"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	portcloud "github.com/sevoniva/nivora/internal/ports/cloud"
	"github.com/sevoniva/nivora/internal/ports/policy"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
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
	approvalService := NewApprovalService()
	return deploymentusecase.NewService(
		store,
		deploymentusecase.NewStaticManifestRenderer(),
		yamlapply.NoopManifestClient{},
		allowAllPolicyEngine{},
		bus,
	).WithHostExecutor(hostexecutor.NewNoop()).WithGitOps(localgitops.New(), argocdadapter.NoopProvider{AllowSync: true}).WithSecurity(NewSecurityService()).WithGovernance(approvalService)
}

func NewArtifactService() *artifactusecase.Service {
	return artifactusecase.NewService(artifactusecase.NewMemoryStore(), ociartifact.New(), memory.New())
}

func NewReleaseOrchestrationService() *releaseorchestration.Service {
	return NewReleaseOrchestrationServiceWith(NewArtifactService(), NewDeploymentService())
}

func NewReleaseOrchestrationServiceWith(artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) *releaseorchestration.Service {
	bus := memory.New()
	approvalService := NewApprovalService()
	return releaseorchestration.NewService(
		releaseorchestration.NewMemoryStore(),
		artifactService,
		deploymentService,
		allowAllPolicyEngine{},
		bus,
	).WithSecurity(NewSecurityService()).WithGovernance(approvalService)
}

func NewSecurityService() *securityusecase.Service {
	bus := memory.New()
	return securityusecase.NewService(securityusecase.NewMemoryStore(), securitynoop.New(), securitynoop.SignatureVerifier{}, bus)
}

func NewCredentialService() *credentialusecase.Service {
	return credentialusecase.NewService(credentialusecase.NewMemoryStore(), builtinsecret.New(), memory.New())
}

func NewAuthService() *authusecase.Service {
	return authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
}

func NewApprovalService() *approvalusecase.Service {
	return approvalusecase.NewService(approvalusecase.NewMemoryStore(), noopnotification.New(), memory.New())
}

func NewCloudService() *cloudusecase.Service {
	providers := map[string]portcloud.CloudProvider{
		domaincloud.ProviderAWS:     aws.New(),
		domaincloud.ProviderAliyun:  aliyun.New(),
		domaincloud.ProviderTencent: tencent.New(),
		domaincloud.ProviderGeneric: cloudfake.New(domaincloud.ProviderGeneric),
	}
	return cloudusecase.NewService(cloudusecase.NewMemoryStore(), providers, memory.New())
}

func NewPluginRegistry() *pluginusecase.Registry {
	return pluginusecase.NewDefaultRegistry()
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
