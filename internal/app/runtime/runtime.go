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
	scmgeneric "github.com/sevoniva/nivora/internal/adapters/scm/generic"
	builtinsecret "github.com/sevoniva/nivora/internal/adapters/secret/builtin"
	securitynoop "github.com/sevoniva/nivora/internal/adapters/security/noop"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domaincloud "github.com/sevoniva/nivora/internal/domain/cloud"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/db"
	portartifact "github.com/sevoniva/nivora/internal/ports/artifact"
	portcloud "github.com/sevoniva/nivora/internal/ports/cloud"
	"github.com/sevoniva/nivora/internal/ports/policy"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	cloudusecase "github.com/sevoniva/nivora/internal/usecase/cloud"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	credentialusecase "github.com/sevoniva/nivora/internal/usecase/credential"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	repositoryusecase "github.com/sevoniva/nivora/internal/usecase/repository"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
	tenancyusecase "github.com/sevoniva/nivora/internal/usecase/tenancy"
	workflowusecase "github.com/sevoniva/nivora/internal/usecase/workflow"
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

func NewCatalogService() *catalogusecase.Service {
	return catalogusecase.NewService(catalogusecase.NewMemoryStore())
}

func NewCatalogServiceWithConfig(ctx context.Context, cfg config.Config) (*catalogusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewCatalogService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return catalogusecase.NewService(postgresrepo.NewCatalogStore(pool)), pool.Close, nil
}

func NewPipelineDefinitionCatalog() *pipelineusecase.DefinitionCatalog {
	return pipelineusecase.NewDefinitionCatalog(pipelineusecase.NewDefinitionMemoryStore())
}

func NewPipelineDefinitionCatalogWithConfig(ctx context.Context, cfg config.Config) (*pipelineusecase.DefinitionCatalog, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewPipelineDefinitionCatalog(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return pipelineusecase.NewDefinitionCatalog(postgresrepo.NewPipelineDefinitionStore(pool)), pool.Close, nil
}

func NewRepositoryService() *repositoryusecase.Service {
	return repositoryusecase.NewService(repositoryusecase.NewMemoryStore(), scmgeneric.New())
}

func NewRepositoryServiceWithConfig(ctx context.Context, cfg config.Config) (*repositoryusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewRepositoryService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return repositoryusecase.NewService(postgresrepo.NewRepositoryStore(pool), scmgeneric.New()), pool.Close, nil
}

func NewWorkflowService() *workflowusecase.Service {
	return workflowusecase.NewService(workflowusecase.NewMemoryStore())
}

func NewWorkflowServiceWithConfig(ctx context.Context, cfg config.Config) (*workflowusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewWorkflowService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return workflowusecase.NewService(postgresrepo.NewWorkflowStore(pool)), pool.Close, nil
}

func NewDeploymentService() *deploymentusecase.Service {
	return NewDeploymentServiceWithStore(deploymentusecase.NewMemoryStore())
}

func NewDeploymentServiceWithConfig(ctx context.Context, cfg config.Config) (*deploymentusecase.Service, func(), error) {
	securityService, closeSecurity, err := NewSecurityServiceWithConfig(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	approvalService, closeApproval, err := NewApprovalServiceWithConfig(ctx, cfg)
	if err != nil {
		closeSecurity()
		return nil, nil, err
	}
	service, closeStore, err := NewDeploymentServiceWithConfigDependencies(ctx, cfg, securityService, approvalService)
	if err != nil {
		closeApproval()
		closeSecurity()
		return nil, nil, err
	}
	return service, func() {
		closeStore()
		closeApproval()
		closeSecurity()
	}, nil
}

func NewDeploymentServiceWithConfigDependencies(ctx context.Context, cfg config.Config, securityService *securityusecase.Service, approvalService *approvalusecase.Service) (*deploymentusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewDeploymentServiceWithStoreAndGovernance(deploymentusecase.NewMemoryStore(), securityService, approvalService), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return NewDeploymentServiceWithStoreAndGovernance(postgresrepo.NewDeploymentStore(pool), securityService, approvalService), pool.Close, nil
}

func NewDeploymentServiceWithStore(store deploymentusecase.Store) *deploymentusecase.Service {
	return NewDeploymentServiceWithStoreAndGovernance(store, nil, nil)
}

func NewDeploymentServiceWithStoreAndGovernance(store deploymentusecase.Store, securityService *securityusecase.Service, approvalService *approvalusecase.Service) *deploymentusecase.Service {
	bus := memory.New()
	if securityService == nil {
		securityService = NewSecurityService()
	}
	if approvalService == nil {
		approvalService = NewApprovalService()
	}
	return deploymentusecase.NewService(
		store,
		deploymentusecase.NewStaticManifestRenderer(),
		yamlapply.NoopManifestClient{},
		allowAllPolicyEngine{},
		bus,
	).WithHostExecutor(hostexecutor.NewNoop()).WithGitOps(localgitops.New(), argocdadapter.NoopProvider{AllowSync: true}).WithSecurity(securityService).WithGovernance(approvalService)
}

func NewArtifactService() *artifactusecase.Service {
	return NewArtifactServiceWithSecretProvider(newRuntimeSecretProvider())
}

func NewArtifactServiceWithConfig(ctx context.Context, cfg config.Config) (*artifactusecase.Service, func(), error) {
	return NewArtifactServiceWithConfigAndSecretProvider(ctx, cfg, newRuntimeSecretProvider())
}

func NewArtifactServiceWithConfigAndSecretProvider(ctx context.Context, cfg config.Config, secrets portsecret.Provider) (*artifactusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewArtifactServiceWithSecretProvider(secrets), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return NewArtifactServiceWithStoreAndSecretProvider(postgresrepo.NewReleaseStore(pool), secrets), pool.Close, nil
}

func NewArtifactServiceWithStore(store artifactusecase.Store) *artifactusecase.Service {
	return NewArtifactServiceWithStoreAndSecretProvider(store, newRuntimeSecretProvider())
}

func NewArtifactServiceWithSecretProvider(secrets portsecret.Provider) *artifactusecase.Service {
	return NewArtifactServiceWithStoreAndSecretProvider(artifactusecase.NewMemoryStore(), secrets)
}

func NewArtifactServiceWithStoreAndSecretProvider(store artifactusecase.Store, secrets portsecret.Provider) *artifactusecase.Service {
	secrets = ensureRuntimeSecretProvider(secrets)
	return artifactusecase.NewService(store, ociartifact.New(ociartifact.WithSecretProvider(secrets)), memory.New())
}

func NewArtifactRegistryService() *artifactusecase.RegistryService {
	return NewArtifactRegistryServiceWithSecretProvider(newRuntimeSecretProvider())
}

func NewArtifactRegistryServiceWithConfig(ctx context.Context, cfg config.Config) (*artifactusecase.RegistryService, func(), error) {
	return NewArtifactRegistryServiceWithConfigAndSecretProvider(ctx, cfg, newRuntimeSecretProvider())
}

func NewArtifactRegistryServiceWithConfigAndSecretProvider(ctx context.Context, cfg config.Config, secrets portsecret.Provider) (*artifactusecase.RegistryService, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewArtifactRegistryServiceWithSecretProvider(secrets), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return artifactusecase.NewRegistryServiceWithProviderFactory(postgresrepo.NewArtifactRegistryStore(pool), artifactRegistryProviderFactoryWithSecretProvider(secrets)), pool.Close, nil
}

func NewArtifactRegistryServiceWithSecretProvider(secrets portsecret.Provider) *artifactusecase.RegistryService {
	return artifactusecase.NewRegistryServiceWithProviderFactory(artifactusecase.NewRegistryMemoryStore(), artifactRegistryProviderFactoryWithSecretProvider(secrets))
}

func artifactRegistryProviderFactory() artifactusecase.RegistryProviderFactory {
	return artifactRegistryProviderFactoryWithSecretProvider(newRuntimeSecretProvider())
}

func artifactRegistryProviderFactoryWithSecretProvider(secrets portsecret.Provider) artifactusecase.RegistryProviderFactory {
	secrets = ensureRuntimeSecretProvider(secrets)
	return func(registry domainartifact.ArtifactRegistry) portartifact.ArtifactProvider {
		return ociartifact.New(
			ociartifact.WithConfig(ociartifact.Config{
				Name:          registry.Name,
				Endpoint:      registry.Endpoint,
				Insecure:      registry.Insecure,
				CredentialRef: portartifact.CredentialRef{ID: registry.CredentialRef},
			}),
			ociartifact.WithSecretProvider(secrets),
		)
	}
}

func NewReleaseOrchestrationService() *releaseorchestration.Service {
	return NewReleaseOrchestrationServiceWith(NewArtifactService(), NewDeploymentService())
}

func NewReleaseOrchestrationServiceWith(artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) *releaseorchestration.Service {
	return NewReleaseOrchestrationServiceWithStore(releaseorchestration.NewMemoryStore(), artifactService, deploymentService)
}

func NewReleaseOrchestrationServiceWithConfig(ctx context.Context, cfg config.Config, artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) (*releaseorchestration.Service, func(), error) {
	securityService, closeSecurity, err := NewSecurityServiceWithConfig(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	approvalService, closeApproval, err := NewApprovalServiceWithConfig(ctx, cfg)
	if err != nil {
		closeSecurity()
		return nil, nil, err
	}
	service, closeStore, err := NewReleaseOrchestrationServiceWithConfigDependencies(ctx, cfg, artifactService, deploymentService, securityService, approvalService)
	if err != nil {
		closeApproval()
		closeSecurity()
		return nil, nil, err
	}
	return service, func() {
		closeStore()
		closeApproval()
		closeSecurity()
	}, nil
}

func NewReleaseOrchestrationServiceWithConfigDependencies(ctx context.Context, cfg config.Config, artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service, securityService *securityusecase.Service, approvalService *approvalusecase.Service) (*releaseorchestration.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewReleaseOrchestrationServiceWithStoreAndGovernance(releaseorchestration.NewMemoryStore(), artifactService, deploymentService, securityService, approvalService), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return NewReleaseOrchestrationServiceWithStoreAndGovernance(postgresrepo.NewReleaseOrchestrationStore(pool), artifactService, deploymentService, securityService, approvalService), pool.Close, nil
}

func NewReleaseOrchestrationServiceWithStore(store releaseorchestration.Store, artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service) *releaseorchestration.Service {
	return NewReleaseOrchestrationServiceWithStoreAndGovernance(store, artifactService, deploymentService, nil, nil)
}

func NewReleaseOrchestrationServiceWithStoreAndGovernance(store releaseorchestration.Store, artifactService *artifactusecase.Service, deploymentService *deploymentusecase.Service, securityService *securityusecase.Service, approvalService *approvalusecase.Service) *releaseorchestration.Service {
	bus := memory.New()
	if securityService == nil {
		securityService = NewSecurityService()
	}
	if approvalService == nil {
		approvalService = NewApprovalService()
	}
	return releaseorchestration.NewService(
		store,
		artifactService,
		deploymentService,
		allowAllPolicyEngine{},
		bus,
	).WithSecurity(securityService).WithGovernance(approvalService)
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

func NewPolicyCatalogService() *policyusecase.Service {
	return policyusecase.NewService(policyusecase.NewMemoryStore())
}

func NewPolicyCatalogServiceWithConfig(ctx context.Context, cfg config.Config) (*policyusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewPolicyCatalogService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return policyusecase.NewService(postgresrepo.NewPolicyStore(pool)), pool.Close, nil
}

func NewCredentialService() *credentialusecase.Service {
	return NewCredentialServiceWithSecretProvider(newRuntimeSecretProvider())
}

func NewCredentialServiceWithConfig(ctx context.Context, cfg config.Config) (*credentialusecase.Service, func(), error) {
	return NewCredentialServiceWithConfigAndSecretProvider(ctx, cfg, newRuntimeSecretProvider())
}

func NewCredentialServiceWithConfigAndSecretProvider(ctx context.Context, cfg config.Config, secrets portsecret.Provider) (*credentialusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewCredentialServiceWithSecretProvider(secrets), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return credentialusecase.NewService(postgresrepo.NewCredentialStore(pool), ensureRuntimeSecretProvider(secrets), memory.New()), pool.Close, nil
}

func NewCredentialServiceWithSecretProvider(secrets portsecret.Provider) *credentialusecase.Service {
	return credentialusecase.NewService(credentialusecase.NewMemoryStore(), ensureRuntimeSecretProvider(secrets), memory.New())
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

func NewTenancyServiceWithConfig(ctx context.Context, cfg config.Config) (*tenancyusecase.Service, func(), error) {
	if cfg.Database.RuntimeStore != "postgres" {
		return NewTenancyService(), func() {}, nil
	}
	pool, err := db.Open(ctx, cfg.Database.URL)
	if err != nil {
		return nil, nil, err
	}
	return tenancyusecase.NewServiceWithStore(postgresrepo.NewTenancyStore(pool)), pool.Close, nil
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

func NewSecretProvider() portsecret.Provider {
	return builtinsecret.New()
}

func newRuntimeSecretProvider() portsecret.Provider {
	return NewSecretProvider()
}

func ensureRuntimeSecretProvider(secrets portsecret.Provider) portsecret.Provider {
	if secrets != nil {
		return secrets
	}
	return newRuntimeSecretProvider()
}

func (allowAllPolicyEngine) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	select {
	case <-ctx.Done():
		return policy.Result{}, ctx.Err()
	default:
		return policy.Result{Allowed: true, Reasons: []string{"Phase 2.1 allow-all policy placeholder"}}, nil
	}
}
