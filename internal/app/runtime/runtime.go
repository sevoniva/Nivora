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
	postgresrepo "github.com/sevoniva/nivora/internal/adapters/repository/postgres"
	builtinsecret "github.com/sevoniva/nivora/internal/adapters/secret/builtin"
	securitynoop "github.com/sevoniva/nivora/internal/adapters/security/noop"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/db"
	portcloud "github.com/sevoniva/nivora/internal/ports/cloud"
	"github.com/sevoniva/nivora/internal/ports/policy"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
)

func NewPipelineService() *pipelineusecase.Service {
	store := pipelineusecase.NewMemoryStore()
	bus := memory.New()
	runner := pipelineusecase.NewLocalRunner("local-runner", shellexecutor.New())
	return pipelineusecase.NewService(store, runner, bus)
}

func NewPipelineServiceWithConfig(ctx context.Context, cfg config.Config) (*pipelineusecase.Service, func(), error) {
	bus := memory.New()
	runner := pipelineusecase.NewLocalRunner(cfg.Runner.Name, shellexecutor.New())
	if cfg.Database.RuntimeStore != "postgres" {
		return pipelineusecase.NewService(pipelineusecase.NewMemoryStore(), runner, bus), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return pipelineusecase.NewService(postgresrepo.NewPipelineStore(pool), runner, bus), pool.Close, nil
}

func NewDeploymentService() *deploymentusecase.Service {
	return NewDeploymentServiceWithStore(deploymentusecase.NewMemoryStore())
}

func NewDeploymentServiceWithConfig(ctx context.Context, cfg config.Config) (*deploymentusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewDeploymentService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return NewDeploymentServiceWithStore(postgresrepo.NewDeploymentStore(pool)), pool.Close, nil
}

func NewDeploymentServiceWithStore(store deploymentusecase.Store) *deploymentusecase.Service {
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
	return NewArtifactServiceWithStore(artifactusecase.NewMemoryStore())
}

func NewArtifactServiceWithConfig(ctx context.Context, cfg config.Config) (*artifactusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewArtifactService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return NewArtifactServiceWithStore(postgresrepo.NewReleaseStore(pool)), pool.Close, nil
}

func NewArtifactServiceWithStore(store artifactusecase.Store) *artifactusecase.Service {
	return artifactusecase.NewService(store, ociartifact.New(ociartifact.WithSecretProvider(builtinsecret.New())), memory.New())
}

func NewReleaseOrchestrationService() *releaseorchestration.Service {
	return NewReleaseOrchestrationServiceWith(NewArtifactService(), NewDeploymentService())
}

func NewReleaseOrchestrationServiceWith(artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) *releaseorchestration.Service {
	return NewReleaseOrchestrationServiceWithStore(releaseorchestration.NewMemoryStore(), artifactService, deploymentService)
}

func NewReleaseOrchestrationServiceWithConfig(ctx context.Context, cfg config.Config, artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) (*releaseorchestration.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewReleaseOrchestrationServiceWith(artifactService, deploymentService), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return NewReleaseOrchestrationServiceWithStore(postgresrepo.NewReleaseOrchestrationStore(pool), artifactService, deploymentService), pool.Close, nil
}

func NewReleaseOrchestrationServiceWithStore(store releaseorchestration.Store, artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) *releaseorchestration.Service {
	bus := memory.New()
	approvalService := NewApprovalService()
	return releaseorchestration.NewService(
		store,
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

func NewSecurityServiceWithConfig(ctx context.Context, cfg config.Config) (*securityusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewSecurityService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	bus := memory.New()
	return securityusecase.NewService(postgresrepo.NewSecurityStore(pool), securitynoop.New(), securitynoop.SignatureVerifier{}, bus), pool.Close, nil
}

func NewCredentialService() *credentialusecase.Service {
	return credentialusecase.NewService(credentialusecase.NewMemoryStore(), builtinsecret.New(), memory.New())
}

func NewCredentialServiceWithConfig(ctx context.Context, cfg config.Config) (*credentialusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewCredentialService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return credentialusecase.NewService(postgresrepo.NewCredentialStore(pool), builtinsecret.New(), memory.New()), pool.Close, nil
}

func NewAuthService() *authusecase.Service {
	return authusecase.NewService(authusecase.NewMemoryStore(), memory.New())
}

func NewAuthServiceWithConfig(ctx context.Context, cfg config.Config) (*authusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewAuthService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return authusecase.NewService(postgresrepo.NewAuthStore(pool), memory.New()), pool.Close, nil
}

func NewApprovalService() *approvalusecase.Service {
	return approvalusecase.NewService(approvalusecase.NewMemoryStore(), noopnotification.New(), memory.New())
}

func NewApprovalServiceWithConfig(ctx context.Context, cfg config.Config) (*approvalusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewApprovalService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return approvalusecase.NewService(postgresrepo.NewApprovalStore(pool), noopnotification.New(), memory.New()), pool.Close, nil
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

func NewCloudServiceWithConfig(ctx context.Context, cfg config.Config) (*cloudusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewCloudService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	providers := map[string]portcloud.CloudProvider{
		domaincloud.ProviderAWS:     aws.New(),
		domaincloud.ProviderAliyun:  aliyun.New(),
		domaincloud.ProviderTencent: tencent.New(),
		domaincloud.ProviderGeneric: cloudfake.New(domaincloud.ProviderGeneric),
	}
	return cloudusecase.NewService(postgresrepo.NewCloudStore(pool), providers, memory.New()), pool.Close, nil
}

func NewTenancyService() *tenancyusecase.Service {
	return tenancyusecase.NewService()
}

func NewComplianceService(pipelineService *pipelineusecase.Service, deploymentService *deploymentusecase.Service, artifactService *artifactusecase.Service, releaseService *releaseorchestration.Service, securityService *securityusecase.Service, approvalService *approvalusecase.Service) *complianceusecase.Service {
	return complianceusecase.NewService(pipelineService, deploymentService, artifactService, releaseService, securityService, approvalService)
}

func NewComplianceServiceWithConfig(ctx context.Context, cfg config.Config, pipelineService *pipelineusecase.Service, deploymentService *deploymentusecase.Service, artifactService *artifactusecase.Service, releaseService *releaseorchestration.Service, securityService *securityusecase.Service, approvalService *approvalusecase.Service) (*complianceusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewComplianceService(pipelineService, deploymentService, artifactService, releaseService, securityService, approvalService), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return complianceusecase.NewServiceWithStore(postgresrepo.NewComplianceStore(pool), pipelineService, deploymentService, artifactService, releaseService, securityService, approvalService), pool.Close, nil
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
