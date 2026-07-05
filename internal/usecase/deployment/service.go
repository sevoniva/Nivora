package deployment

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpolicy "github.com/sevoniva/nivora/internal/domain/policy"
	"github.com/sevoniva/nivora/internal/domain/release"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	portexecutor "github.com/sevoniva/nivora/internal/ports/executor"
	portgitops "github.com/sevoniva/nivora/internal/ports/gitops"
	"github.com/sevoniva/nivora/internal/ports/policy"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

const (
	EventDeploymentCreated             = "devops.deployment.created"
	EventDeploymentPlanning            = "devops.deployment.planning"
	EventDeploymentPrecheckStarted     = "devops.deployment.precheck.started"
	EventDeploymentPrecheckCompleted   = "devops.deployment.precheck.completed"
	EventDeploymentDryRunStarted       = "devops.deployment.dryrun.started"
	EventDeploymentDryRunCompleted     = "devops.deployment.dryrun.completed"
	EventDeploymentDryRunFailed        = "devops.deployment.dryrun.failed"
	EventDeploymentApplyStarted        = "devops.deployment.apply.started"
	EventDeploymentApplyCompleted      = "devops.deployment.apply.completed"
	EventDeploymentApplyFailed         = "devops.deployment.apply.failed"
	EventDeploymentVerifyStarted       = "devops.deployment.verify.started"
	EventDeploymentVerifyCompleted     = "devops.deployment.verify.completed"
	EventDeploymentVerifyFailed        = "devops.deployment.verify.failed"
	EventDeploymentSucceeded           = "devops.deployment.succeeded"
	EventDeploymentFailed              = "devops.deployment.failed"
	EventDeploymentCanceled            = "devops.deployment.canceled"
	EventDeploymentInventoryCreated    = "devops.deployment.inventory.created"
	EventDeploymentSnapshotCreated     = "devops.deployment.snapshot.created"
	EventDeploymentHealthStarted       = "devops.deployment.health.started"
	EventDeploymentHealthCompleted     = "devops.deployment.health.completed"
	EventDeploymentRollbackPlanCreated = "devops.deployment.rollback.plan.created"
	EventDeploymentRollbackStarted     = "devops.deployment.rollback.started"
	EventDeploymentRollbackSucceeded   = "devops.deployment.rollback.succeeded"
	EventDeploymentRollbackFailed      = "devops.deployment.rollback.failed"
	EventDeploymentDiffGenerated       = "devops.deployment.diff.generated"
	EventDeploymentResourceObserved    = "devops.deployment.resource.observed"
	EventDeploymentResourceDegraded    = "devops.deployment.resource.degraded"
	EventGitOpsPlanCreated             = "devops.gitops.plan.created"
	EventGitOpsDiffGenerated           = "devops.gitops.diff.generated"
	EventGitOpsWorkingTreeUpdated      = "devops.gitops.workingtree.updated"
	EventGitOpsCommitCreated           = "devops.gitops.commit.created"
	EventGitOpsPushCompleted           = "devops.gitops.push.completed"
	EventGitOpsPushSkipped             = "devops.gitops.push.skipped"
	EventGitOpsRollbackPlanned         = "devops.gitops.rollback.planned"
	EventGitOpsRollbackCompleted       = "devops.gitops.rollback.completed"
	EventArgoCDStatusReadStarted       = "devops.argocd.status.read.started"
	EventArgoCDStatusReadCompleted     = "devops.argocd.status.read.completed"
	EventArgoCDStatusReadFailed        = "devops.argocd.status.read.failed"
	EventArgoCDStatusRead              = "devops.argocd.status.read"
	EventArgoCDSyncRequested           = "devops.argocd.sync.requested"
	EventArgoCDSyncSkipped             = "devops.argocd.sync.skipped"
	EventArgoCDSyncStarted             = "devops.argocd.sync.started"
	EventArgoCDSyncCompleted           = "devops.argocd.sync.completed"
	EventArgoCDSyncFailed              = "devops.argocd.sync.failed"
	EventArgoCDHealthChanged           = "devops.argocd.health.changed"
	EventHostDeploymentPlanCreated     = "devops.host.deployment.plan.created"
	EventHostDeploymentStarted         = "devops.host.deployment.started"
	EventHostDeploymentHostStarted     = "devops.host.deployment.host.started"
	EventHostDeploymentHostCompleted   = "devops.host.deployment.host.completed"
	EventHostDeploymentHostFailed      = "devops.host.deployment.host.failed"
	EventHostDeploymentHealthCompleted = "devops.host.deployment.health.completed"
	EventHostRollbackPlanCreated       = "devops.host.rollback.plan.created"
	EventHostRollbackStarted           = "devops.host.rollback.started"
	EventHostRollbackCompleted         = "devops.host.rollback.completed"
	EventHostRollbackFailed            = "devops.host.rollback.failed"
)

type KubernetesManifestClient interface {
	ServerDryRun(ctx context.Context, request ManifestRequest) (KubernetesDryRunResult, error)
	Apply(ctx context.Context, request ManifestRequest) (KubernetesApplyResult, error)
	WatchRollout(ctx context.Context, request ManifestRequest) (RolloutResult, error)
	Rollback(ctx context.Context, request ManifestRequest) (KubernetesRollbackResult, error)
}

type ManifestClient = KubernetesManifestClient

type Service struct {
	store      Store
	renderer   ManifestRenderer
	client     ManifestClient
	policy     policy.Engine
	eventBus   eventbus.EventBus
	host       portexecutor.HostExecutor
	gitops     portgitops.WorkingTree
	argocd     portargocd.Provider
	security   *securityusecase.Service
	policies   SecurityPolicyCatalog
	repos      RepositoryCatalog
	governance Governance
	now        func() time.Time
}

type Governance interface {
	RequestApproval(ctx context.Context, subjectType string, subjectID string, environmentID string, requestedBy string, reason string) (domainapproval.ApprovalRequest, error)
	EvaluateChangeWindow(ctx context.Context, environmentID string) (domainapproval.ChangeWindowResult, error)
}

type SecurityPolicyCatalog interface {
	ResolveEnabledForScope(ctx context.Context, input policyusecase.ResolveInput) (domainpolicy.Policy, bool, error)
}

type RepositoryCatalog interface {
	GetRepository(ctx context.Context, id string) (domainapp.Repository, error)
}

func NewService(store Store, renderer ManifestRenderer, client ManifestClient, policyEngine policy.Engine, bus eventbus.EventBus) *Service {
	return &Service{
		store:    store,
		renderer: renderer,
		client:   client,
		policy:   policyEngine,
		eventBus: bus,
		now:      time.Now,
	}
}

func (s *Service) WithGitOps(workingTree portgitops.WorkingTree, provider portargocd.Provider) *Service {
	s.gitops = workingTree
	s.argocd = provider
	return s
}

func (s *Service) WithHostExecutor(executor portexecutor.HostExecutor) *Service {
	s.host = executor
	return s
}

func (s *Service) WithSecurity(securityService *securityusecase.Service) *Service {
	s.security = securityService
	return s
}

func (s *Service) WithPolicyCatalog(catalog SecurityPolicyCatalog) *Service {
	s.policies = catalog
	return s
}

func (s *Service) WithRepositoryCatalog(catalog RepositoryCatalog) *Service {
	s.repos = catalog
	return s
}

func (s *Service) WithGovernance(governance Governance) *Service {
	s.governance = governance
	return s
}

func (s *Service) CreateAndRun(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	input.Definition = normalizeDefinition(input.Definition)
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}
	resolvedDefinition, err := s.resolveGitOpsRepository(ctx, input.Definition, input.ProjectID)
	if err != nil {
		return CreateRunResult{}, err
	}
	input.Definition = resolvedDefinition
	if input.Definition.Spec.Options.Apply && (!input.AllowApply || !input.Confirm) {
		return CreateRunResult{}, fmt.Errorf("deployment apply requires explicit confirmation")
	}
	if input.Definition.Spec.Target.Type == "host" && input.Definition.Spec.Options.Apply {
		if !input.Confirm || !input.Definition.Spec.Host.AllowRemoteHostDeploy {
			return CreateRunResult{}, fmt.Errorf("remote host deployment requires confirm=true and host.allowRemoteHostDeploy=true")
		}
		if hostCredentialRef(input.Definition) == "" {
			return CreateRunResult{}, fmt.Errorf("remote host deployment requires host credentialRef")
		}
	}
	record := s.newRecord(input.Definition)
	record = applyProjectScope(record, input.ProjectID)
	record.Run.CorrelationID = input.CorrelationID
	if err := s.store.Save(ctx, record); err != nil {
		return CreateRunResult{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentCreated, "DeploymentRun created", input.ActorID, string(record.Run.Status), "DeploymentRun created"); err != nil {
		return CreateRunResult{}, err
	}
	if input.Definition.Spec.Target.Type == "argocd" {
		record, err := s.processGitOps(ctx, record, input.ActorID, input.AllowSync, input.Confirm)
		if err != nil {
			return CreateRunResult{}, err
		}
		return CreateRunResult{Record: record}, nil
	}
	if input.Definition.Spec.Target.Type == "host" {
		record, err := s.processHost(ctx, record, input.ActorID, input.Confirm)
		if err != nil {
			return CreateRunResult{}, err
		}
		return CreateRunResult{Record: record}, nil
	}
	record, err = s.process(ctx, record, input.ActorID)
	if err != nil {
		return CreateRunResult{}, err
	}
	return CreateRunResult{Record: record}, nil
}

func (s *Service) Plan(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	input.Definition = normalizeDefinition(input.Definition)
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}
	resolvedDefinition, err := s.resolveGitOpsRepository(ctx, input.Definition, input.ProjectID)
	if err != nil {
		return CreateRunResult{}, err
	}
	input.Definition = resolvedDefinition
	record := s.newRecord(input.Definition)
	record = applyProjectScope(record, input.ProjectID)
	record.Run.CorrelationID = input.CorrelationID
	if input.Definition.Spec.Target.Type == "argocd" {
		record.GitOpsPlan = s.buildGitOpsPlan(record.Run.ID, input.Definition, nil)
		record.Plan = deploymentPlanFromGitOps(record.GitOpsPlan, input.Definition)
		return CreateRunResult{Record: record}, nil
	}
	if input.Definition.Spec.Target.Type == "host" {
		record.Plan, record.HostPlan, record.RollbackPlan = s.buildHostPlan(record.Run.ID, input.Definition)
		record.Rollback = s.hostRollbackBaseline(record.Run.ID, input.Definition, record.HostPlan)
		return CreateRunResult{Record: record}, nil
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPlanning, s.now(), ""); err != nil {
		return CreateRunResult{}, err
	}
	documents, err := s.renderer.Render(ctx, input.Definition.Spec.Manifests, input.Definition.Spec.Target.Namespace)
	if err != nil {
		return CreateRunResult{}, err
	}
	record.Plan = s.buildPlan(record.Run.ID, input.Definition, documents)
	record.Plan = appendKubernetesSafetyWarnings(record.Plan, documents, input.Definition.Spec.Target.Namespace)
	record = s.attachResourceObservability(record, input.Definition, documents)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("rendered %d manifest document(s)", len(documents)), int64(len(record.Logs)+1)))
	return CreateRunResult{Record: record}, nil
}

func (s *Service) ApplyApprovalDecision(ctx context.Context, id string, approval domainapproval.ApprovalRequest, actorID string) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if record.Run.Status != domaindeployment.DeploymentRunWaitingApproval {
		return RunRecord{}, fmt.Errorf("deployment run is not waiting for approval")
	}
	if approval.SubjectID != "" && approval.SubjectID != id {
		return RunRecord{}, fmt.Errorf("approval subject does not match deployment run")
	}
	record.Approval = approval
	switch approval.Status {
	case domainapproval.StatusApproved:
		record.Definition.Spec.Options.ApprovalRequired = false
		record.Run.Reason = "approval approved"
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPrecheckCompleted, "Deployment approval approved", actorID, string(record.Run.Status), "Deployment approval approved; resuming deployment"); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
		return s.resumeKubernetesAfterApproval(ctx, record, actorID)
	case domainapproval.StatusRejected, domainapproval.StatusExpired:
		reason := "approval " + strings.ToLower(approval.Status)
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunFailed, s.now(), reason); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentFailed, "DeploymentRun failed", actorID, string(record.Run.Status), reason); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, record.Run.ID)
	case domainapproval.StatusCanceled:
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunCanceled, s.now(), "approval canceled"); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentCanceled, "DeploymentRun canceled", actorID, string(record.Run.Status), "approval canceled"); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, record.Run.ID)
	default:
		return RunRecord{}, fmt.Errorf("approval must be Approved, Rejected, Expired, or Canceled")
	}
}

func (s *Service) resumeKubernetesAfterApproval(ctx context.Context, record RunRecord, actorID string) (RunRecord, error) {
	if record.Run.TargetType != "kubernetes-yaml" {
		return RunRecord{}, fmt.Errorf("approval resume currently supports kubernetes-yaml deployments")
	}
	documents, err := s.renderer.Render(ctx, record.Definition.Spec.Manifests, record.Definition.Spec.Target.Namespace)
	if err != nil {
		return s.fail(ctx, record, actorID, err.Error())
	}
	if record.Plan.DeploymentRunID == "" {
		record.Plan = s.buildPlan(record.Run.ID, record.Definition, documents)
		record = s.attachResourceObservability(record, record.Definition, documents)
	}
	var safetyAllowed bool
	if record, safetyAllowed, err = s.enforceKubernetesSafety(ctx, record, documents, actorID); err != nil || !safetyAllowed {
		return record, err
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunVerifying, s.now(), "approval approved"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if record, err = s.recordRuntimeEvent(ctx, record, EventDeploymentDryRunStarted, string(record.Run.Status), "Deployment dry-run started after approval"); err != nil {
		return RunRecord{}, err
	}
	request := ManifestRequest{Plan: record.Plan, Documents: documents, TimeoutSeconds: record.Plan.TimeoutSeconds}
	dryRunResult, err := s.client.ServerDryRun(ctx, request)
	if err != nil {
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentDryRunFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.DryRun = dryRunResult
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", dryRunResult.Message, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentDryRunCompleted, "Deployment dry-run completed", actorID, string(record.Run.Status), "Deployment dry-run completed after approval"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	record.Health = evaluateResourceHealth(record.Run.ID, record.Plan.Resources, s.now())
	record.Rollout = rolloutFromHealth(record.Health)
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentHealthCompleted, "Health evaluation completed", actorID, string(record.Run.Status), string(record.Health.Status)); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if !record.Plan.Apply {
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", "dry-run validation completed after approval", int64(len(record.Logs)+1)))
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "dry-run deployment run succeeded after approval"); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "DeploymentRun dry-run succeeded after approval"); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, record.Run.ID)
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunDeploying, s.now(), "approval approved"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentApplyStarted, "Deployment apply started", actorID, string(record.Run.Status), "Deployment apply started after approval"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	applyResult, err := s.client.Apply(ctx, request)
	if err != nil {
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentApplyFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Apply = applyResult
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", applyResult.Message, int64(len(record.Logs)+1)))
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "kubernetes YAML apply completed after approval"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "DeploymentRun apply succeeded after approval"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) Rollback(ctx context.Context, input RollbackInput) (RunRecord, error) {
	if !input.Confirm {
		return RunRecord{}, fmt.Errorf("deployment rollback requires explicit confirmation")
	}
	record, err := s.store.Get(ctx, input.DeploymentRunID)
	if err != nil {
		return RunRecord{}, err
	}
	if record.Run.TargetType == "host" {
		return s.rollbackHost(ctx, record, input)
	}
	if record.Run.TargetType != "kubernetes-yaml" {
		return RunRecord{}, fmt.Errorf("deployment rollback is only supported for kubernetes-yaml targets")
	}
	if len(record.RollbackPlan.Resources) == 0 {
		return RunRecord{}, fmt.Errorf("deployment rollback requires a rollback plan with resources")
	}
	def := normalizeDefinition(record.Definition)
	documents, err := s.renderer.Render(ctx, def.Spec.Manifests, def.Spec.Target.Namespace)
	if err != nil {
		return RunRecord{}, err
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunRollingBack, s.now(), "rollback requested"); err != nil {
		return RunRecord{}, err
	}
	if record.Rollback != nil {
		record.Rollback.Status = "running"
		record.Rollback.Reason = "manifest restore rollback requested"
		record.Rollback.UpdatedAt = s.now()
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentRollbackStarted, "Deployment rollback started", input.ActorID, string(record.Run.Status), "Guarded manifest restore rollback started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	request := ManifestRequest{Plan: record.Plan, Documents: documents, TimeoutSeconds: record.Plan.TimeoutSeconds}
	result, err := s.client.Rollback(ctx, request)
	if err != nil {
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentRollbackFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
		if record.Rollback != nil {
			record.Rollback.Status = "failed"
			record.Rollback.Reason = err.Error()
			record.Rollback.UpdatedAt = s.now()
		}
		return s.fail(ctx, record, input.ActorID, err.Error())
	}
	if record.Rollback != nil {
		record.Rollback.Status = "succeeded"
		record.Rollback.Reason = result.Message
		record.Rollback.UpdatedAt = s.now()
	}
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", result.Message, int64(len(record.Logs)+1)))
	record.Inventory.Applied = append([]ManifestResourceSummary(nil), result.Resources...)
	if len(record.Inventory.Applied) == 0 {
		record.Inventory.Applied = append([]ManifestResourceSummary(nil), record.Plan.Resources...)
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunRolledBack, s.now(), "manifest restore rollback completed"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentRollbackSucceeded, "Deployment rollback succeeded", input.ActorID, string(record.Run.Status), "Guarded manifest restore rollback completed"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) rollbackHost(ctx context.Context, record RunRecord, input RollbackInput) (RunRecord, error) {
	if len(record.HostPlan.Hosts) == 0 {
		return RunRecord{}, fmt.Errorf("host rollback requires a host deployment plan")
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunRollingBack, s.now(), "host rollback requested"); err != nil {
		return RunRecord{}, err
	}
	if record.Rollback != nil {
		record.Rollback.Status = "running"
		record.Rollback.Reason = "host symlink rollback requested"
		record.Rollback.UpdatedAt = s.now()
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostRollbackStarted, "Host rollback started", input.ActorID, string(record.Run.Status), "Guarded host symlink rollback started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	for _, step := range record.HostPlan.Hosts {
		request := portexecutor.HostDeploymentRequest{
			DeploymentRunID: record.Run.ID,
			HostID:          step.HostID,
			HostName:        step.HostName,
			Address:         step.Address,
			Artifact:        record.HostPlan.Artifact,
			DeployPath:      record.HostPlan.DeployPath,
			ReleaseDir:      step.ReleaseDir,
			ServiceName:     record.HostPlan.ServiceName,
			HealthCheck:     record.HostPlan.HealthCheck,
			RestartCommand:  record.HostPlan.RestartCommand,
			Strategy:        record.HostPlan.Strategy,
			BatchIndex:      step.BatchIndex,
			DryRun:          record.HostPlan.DryRun,
			Apply:           record.HostPlan.Apply,
			Confirmed:       input.Confirm,
			AllowRemote:     record.Definition.Spec.Host.AllowRemoteHostDeploy,
			CredentialRef:   hostCredentialRef(record.Definition),
			TimeoutSeconds:  step.TimeoutSeconds,
		}
		result := portexecutor.HostDeploymentResult{HostID: step.HostID, HostName: step.HostName, Status: "Succeeded", Message: "host rollback skipped by plan-only runtime"}
		if s.host != nil {
			var err error
			result, err = s.host.Rollback(ctx, request)
			if err != nil {
				record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "stderr", err.Error(), int64(len(record.Logs)+1)))
				if record.Rollback != nil {
					record.Rollback.Status = "failed"
					record.Rollback.Reason = err.Error()
					record.Rollback.UpdatedAt = s.now()
				}
				if saveErr := s.store.Save(ctx, record); saveErr != nil {
					return RunRecord{}, saveErr
				}
				if _, eventErr := s.recordRuntimeEvent(ctx, record, EventHostRollbackFailed, string(domaindeployment.DeploymentRunFailed), err.Error()); eventErr != nil {
					return RunRecord{}, eventErr
				}
				return s.fail(ctx, record, input.ActorID, err.Error())
			}
		}
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", result.Message, int64(len(record.Logs)+1)))
	}
	if record.Rollback != nil {
		record.Rollback.Status = "succeeded"
		record.Rollback.Reason = "host symlink rollback completed"
		record.Rollback.UpdatedAt = s.now()
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunRolledBack, s.now(), "host symlink rollback completed"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostRollbackCompleted, "Host rollback completed", input.ActorID, string(record.Run.Status), "Guarded host symlink rollback completed"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) process(ctx context.Context, record RunRecord, actorID string) (RunRecord, error) {
	if record.Definition.Spec.Target.Type == "argocd" {
		return s.processGitOps(ctx, record, actorID, false, false)
	}
	if record.Definition.Spec.Target.Type == "host" {
		return s.processHost(ctx, record, actorID, false)
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPlanning, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPlanning, "DeploymentRun planned", actorID, string(record.Run.Status), "DeploymentRun planning started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	documents, err := s.renderer.Render(ctx, record.Definition.Spec.Manifests, record.Definition.Spec.Target.Namespace)
	if err != nil {
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Plan = s.buildPlan(record.Run.ID, record.Definition, documents)
	record = s.attachResourceObservability(record, record.Definition, documents)
	var safetyAllowed bool
	if record, safetyAllowed, err = s.enforceKubernetesSafety(ctx, record, documents, actorID); err != nil || !safetyAllowed {
		return record, err
	}
	if s.security != nil {
		scanInput, err := s.securityScanInput(ctx, record, actorID)
		if err != nil {
			return s.fail(ctx, record, actorID, err.Error())
		}
		securityRecord, err := s.security.Scan(ctx, scanInput)
		if err != nil {
			return s.fail(ctx, record, actorID, err.Error())
		}
		record.Security = securityRecord
		if securityRecord.Policy.Decision == domainsecurity.GateDeny {
			record.Plan.Warnings = append(record.Plan.Warnings, securityRecord.Policy.Reason)
			return s.fail(ctx, record, actorID, securityRecord.Policy.Reason)
		}
		if securityRecord.Policy.Decision == domainsecurity.GateRequireApproval {
			record.Definition.Spec.Options.ApprovalRequired = true
			record.Plan.Warnings = append(record.Plan.Warnings, securityRecord.Policy.Reason)
		} else if securityRecord.Policy.Decision == domainsecurity.GateWarn {
			record.Plan.Warnings = append(record.Plan.Warnings, securityRecord.Policy.Reason)
		}
	}
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("rendered %d manifest document(s)", len(documents)), int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentInventoryCreated, "Resource inventory captured", actorID, string(record.Run.Status), "Resource inventory captured"); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSnapshotCreated, "Manifest snapshot created", actorID, string(record.Run.Status), "Manifest snapshot created"); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentDiffGenerated, "Deployment diff generated", actorID, string(record.Run.Status), record.Diff.Summary); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentRollbackPlanCreated, "Rollback plan created", actorID, string(record.Run.Status), "Non-destructive rollback plan created"); err != nil {
		return RunRecord{}, err
	}
	for _, resource := range record.Plan.Resources {
		if err := s.recordRuntimeEventOnly(ctx, record.Run.ID, EventDeploymentResourceObserved, string(record.Run.Status), resourceRef(resource)); err != nil {
			return RunRecord{}, err
		}
	}
	record, _ = s.store.Get(ctx, record.Run.ID)

	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPreChecking, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if record, err = s.recordRuntimeEvent(ctx, record, EventDeploymentPrecheckStarted, string(record.Run.Status), "Deployment policy pre-check started"); err != nil {
		return RunRecord{}, err
	}
	policyResult, err := s.policy.Evaluate(ctx, policy.Request{
		Subject: record.Run.ID,
		Action:  "deployment.dryrun",
		Context: map[string]any{"targetType": record.Run.TargetType, "apply": record.Plan.Apply},
	})
	if err != nil {
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Policy = policyResult
	if !policyResult.Allowed {
		reason := "deployment policy denied"
		if len(policyResult.Reasons) > 0 {
			reason = policyResult.Reasons[0]
		}
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentPrecheckCompleted, string(domaindeployment.DeploymentRunFailed), reason)
		return s.fail(ctx, record, actorID, reason)
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPrecheckCompleted, "Deployment policy pre-check completed", actorID, string(record.Run.Status), "Deployment policy pre-check allowed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if record.Definition.Spec.Options.ChangeWindowRequired && s.governance != nil {
		result, err := s.governance.EvaluateChangeWindow(ctx, record.Run.EnvironmentID)
		if err != nil {
			return s.fail(ctx, record, actorID, err.Error())
		}
		record.ChangeWindow = result
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if !result.Allowed {
			return s.fail(ctx, record, actorID, result.Reason)
		}
	}
	if record.Definition.Spec.Options.ApprovalRequired && s.governance != nil {
		approvalReason := "deployment approval required"
		if record.Security.Policy.Decision == domainsecurity.GateRequireApproval && record.Security.Policy.Reason != "" {
			approvalReason = record.Security.Policy.Reason
		}
		approval, err := s.governance.RequestApproval(ctx, domainapproval.SubjectDeployment, record.Run.ID, record.Run.EnvironmentID, actorID, approvalReason)
		if err != nil {
			return s.fail(ctx, record, actorID, err.Error())
		}
		record.Approval = approval
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunWaitingApproval, s.now(), "approval required"); err != nil {
			return RunRecord{}, err
		}
		record.Run.Reason = approvalReason
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		return record, nil
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunVerifying, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if record, err = s.recordRuntimeEvent(ctx, record, EventDeploymentDryRunStarted, string(record.Run.Status), "Deployment dry-run started"); err != nil {
		return RunRecord{}, err
	}
	request := ManifestRequest{Plan: record.Plan, Documents: documents, TimeoutSeconds: record.Plan.TimeoutSeconds}
	dryRunResult, err := s.client.ServerDryRun(ctx, request)
	if err != nil {
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentDryRunFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.DryRun = dryRunResult
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", dryRunResult.Message, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentDryRunCompleted, "Deployment dry-run completed", actorID, string(record.Run.Status), "Deployment dry-run completed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentHealthStarted, "Health evaluation started", actorID, string(record.Run.Status), "Deployment health evaluation started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	record.Health = evaluateResourceHealth(record.Run.ID, record.Plan.Resources, s.now())
	record.Rollout = rolloutFromHealth(record.Health)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("health evaluation completed: %s", record.Health.Status), int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentHealthCompleted, "Health evaluation completed", actorID, string(record.Run.Status), string(record.Health.Status)); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if !record.Plan.Apply {
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", "dry-run validation completed", int64(len(record.Logs)+1)))
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "dry-run deployment run succeeded"); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "DeploymentRun dry-run succeeded"); err != nil {
			return RunRecord{}, err
		}
		return s.store.Get(ctx, record.Run.ID)
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunDeploying, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentApplyStarted, "Deployment apply started", actorID, string(record.Run.Status), "Deployment apply started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	applyResult, err := s.client.Apply(ctx, request)
	if err != nil {
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentApplyFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Apply = applyResult
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", applyResult.Message, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentApplyCompleted, "Deployment apply completed", actorID, string(record.Run.Status), "Deployment apply completed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if record.Plan.Wait {
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunVerifying, s.now(), ""); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentVerifyStarted, "Deployment verification started", actorID, string(record.Run.Status), "Deployment rollout verification started"); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
		rolloutResult, err := s.client.WatchRollout(ctx, request)
		if err != nil {
			record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentVerifyFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
			return s.fail(ctx, record, actorID, err.Error())
		}
		record.Rollout = rolloutResult
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", rolloutResult.Message, int64(len(record.Logs)+1)))
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentVerifyCompleted, "Deployment verification completed", actorID, string(record.Run.Status), "Deployment rollout verification completed"); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "kubernetes YAML apply completed"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "DeploymentRun apply succeeded"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) processHost(ctx context.Context, record RunRecord, actorID string, confirm bool) (RunRecord, error) {
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPlanning, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	record.Plan, record.HostPlan, record.RollbackPlan = s.buildHostPlan(record.Run.ID, record.Definition)
	record.Rollback = s.hostRollbackBaseline(record.Run.ID, record.Definition, record.HostPlan)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("host deployment plan created for %d host(s)", len(record.HostPlan.Hosts)), int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostDeploymentPlanCreated, "Host deployment plan created", actorID, string(record.Run.Status), "Host deployment plan created"); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostRollbackPlanCreated, "Host rollback plan created", actorID, string(record.Run.Status), "Non-destructive host rollback plan created"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPreChecking, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPrecheckStarted, "Deployment policy pre-check started", actorID, string(record.Run.Status), "Host deployment policy pre-check started"); err != nil {
		return RunRecord{}, err
	}
	policyResult, err := s.policy.Evaluate(ctx, policy.Request{
		Subject: record.Run.ID,
		Action:  "deployment.host",
		Context: map[string]any{"targetType": record.Run.TargetType, "apply": record.HostPlan.Apply},
	})
	if err != nil {
		return s.fail(ctx, record, actorID, err.Error())
	}
	record.Policy = policyResult
	if !policyResult.Allowed {
		reason := "host deployment policy denied"
		if len(policyResult.Reasons) > 0 {
			reason = policyResult.Reasons[0]
		}
		record, _ = s.recordRuntimeEvent(ctx, record, EventDeploymentPrecheckCompleted, string(domaindeployment.DeploymentRunFailed), reason)
		return s.fail(ctx, record, actorID, reason)
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPrecheckCompleted, "Deployment policy pre-check completed", actorID, string(record.Run.Status), "Host deployment policy pre-check allowed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)

	nextStatus := domaindeployment.DeploymentRunVerifying
	if record.HostPlan.Apply {
		nextStatus = domaindeployment.DeploymentRunDeploying
	}
	if err := transitionDeploymentRun(&record.Run, nextStatus, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostDeploymentStarted, "Host deployment started", actorID, string(record.Run.Status), "Host deployment runtime started"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)

	currentBatch := 0
	hostFailures := 0
	for _, step := range record.HostPlan.Hosts {
		if step.BatchIndex != currentBatch {
			currentBatch = step.BatchIndex
			record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("host batch %d started", currentBatch), int64(len(record.Logs)+1)))
		}
		started := s.now()
		detail := HostDeploymentRunDetail{
			HostID:        step.HostID,
			HostName:      step.HostName,
			Address:       step.Address,
			BatchIndex:    step.BatchIndex,
			Status:        "Running",
			RollbackReady: record.HostPlan.Apply,
			StartedAt:     started,
		}
		record.HostDetails = append(record.HostDetails, detail)
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("host %s started", step.HostName), int64(len(record.Logs)+1)))
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostDeploymentHostStarted, "Host deployment host started", actorID, string(record.Run.Status), step.HostName); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)

		result, err := s.executeHostStep(ctx, record, step, confirm)
		finished := s.now()
		if err != nil {
			hostFailures++
			record.HostDetails[len(record.HostDetails)-1].Status = "Failed"
			record.HostDetails[len(record.HostDetails)-1].HealthStatus = "Degraded"
			record.HostDetails[len(record.HostDetails)-1].Message = err.Error()
			record.HostDetails[len(record.HostDetails)-1].FinishedAt = finished
			record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "stderr", err.Error(), int64(len(record.Logs)+1)))
			if saveErr := s.store.Save(ctx, record); saveErr != nil {
				return RunRecord{}, saveErr
			}
			record, _ = s.recordRuntimeEvent(ctx, record, EventHostDeploymentHostFailed, string(domaindeployment.DeploymentRunFailed), err.Error())
			if record.HostPlan.PauseOnFailure {
				return s.fail(ctx, record, actorID, err.Error())
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
			continue
		}
		record.HostDetails[len(record.HostDetails)-1].Status = result.Status
		record.HostDetails[len(record.HostDetails)-1].HealthStatus = "Healthy"
		record.HostDetails[len(record.HostDetails)-1].Message = result.Message
		record.HostDetails[len(record.HostDetails)-1].FinishedAt = finished
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", result.Message, int64(len(record.Logs)+1)))
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostDeploymentHostCompleted, "Host deployment host completed", actorID, string(record.Run.Status), result.Message); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
	}
	if hostFailures > 0 {
		return s.fail(ctx, record, actorID, fmt.Sprintf("%d host deployment(s) failed", hostFailures))
	}

	if record.HostPlan.Apply {
		if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunVerifying, s.now(), "host health checks completed"); err != nil {
			return RunRecord{}, err
		}
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventHostDeploymentHealthCompleted, "Host deployment health completed", actorID, string(record.Run.Status), "Host deployment health checks completed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "host deployment dry-run/noop completed"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "Host DeploymentRun succeeded"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) executeHostStep(ctx context.Context, record RunRecord, step HostDeploymentStep, confirm bool) (portexecutor.HostDeploymentResult, error) {
	request := portexecutor.HostDeploymentRequest{
		DeploymentRunID: record.Run.ID,
		HostID:          step.HostID,
		HostName:        step.HostName,
		Address:         step.Address,
		Artifact:        record.HostPlan.Artifact,
		DeployPath:      record.HostPlan.DeployPath,
		ReleaseDir:      step.ReleaseDir,
		ServiceName:     record.HostPlan.ServiceName,
		HealthCheck:     record.HostPlan.HealthCheck,
		RestartCommand:  record.HostPlan.RestartCommand,
		Strategy:        record.HostPlan.Strategy,
		BatchIndex:      step.BatchIndex,
		DryRun:          record.HostPlan.DryRun,
		Apply:           record.HostPlan.Apply,
		Confirmed:       confirm,
		AllowRemote:     record.Definition.Spec.Host.AllowRemoteHostDeploy,
		CredentialRef:   hostCredentialRef(record.Definition),
		TimeoutSeconds:  step.TimeoutSeconds,
	}
	if len(record.HostPlan.HealthChecks) > 0 {
		check := record.HostPlan.HealthChecks[0]
		request.HealthCheck = check.Target
		request.HealthCheckType = check.Type
		if check.Type == "command" {
			request.HealthCheck = check.Command
		}
		if request.TimeoutSeconds == 0 {
			request.TimeoutSeconds = check.TimeoutSeconds
		}
	}
	if s.host == nil {
		return portexecutor.HostDeploymentResult{HostID: step.HostID, HostName: step.HostName, Status: "Succeeded", Message: "host executor not configured; Phase 3.5 plan-only runtime completed"}, nil
	}
	if _, err := s.host.Prepare(ctx, request); err != nil {
		return portexecutor.HostDeploymentResult{}, err
	}
	if record.HostPlan.Apply {
		if _, err := s.host.Upload(ctx, request); err != nil {
			return portexecutor.HostDeploymentResult{}, err
		}
		if _, err := s.host.Execute(ctx, request); err != nil {
			return portexecutor.HostDeploymentResult{}, err
		}
	}
	return s.host.HealthCheck(ctx, request)
}

func (s *Service) securityScanInput(ctx context.Context, record RunRecord, actorID string) (securityusecase.ScanInput, error) {
	projectID := strings.TrimSpace(record.Environment.ProjectID)
	environmentID := strings.TrimSpace(record.Definition.Spec.Environment)
	if environmentID == "" {
		environmentID = strings.TrimSpace(record.Run.EnvironmentID)
	}
	input := securityusecase.ScanInput{
		SubjectType:   domainsecurity.SubjectDeploymentPlan,
		SubjectID:     record.Run.ID,
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		Reference:     strings.Join(record.Plan.Artifacts, ","),
		Policy:        securityusecase.DefaultPolicyConfig(),
		ActorID:       actorID,
	}
	if s.policies == nil {
		return input, nil
	}
	policy, ok, err := s.policies.ResolveEnabledForScope(ctx, policyusecase.ResolveInput{
		ProjectID:     projectID,
		EnvironmentID: environmentID,
	})
	if err != nil || !ok {
		return input, err
	}
	securityusecase.ApplyPolicyDefinition(policy, &input)
	return input, nil
}

func (s *Service) fail(ctx context.Context, record RunRecord, actorID string, reason string) (RunRecord, error) {
	_ = transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunFailed, s.now(), reason)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", reason, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentFailed, "DeploymentRun failed", actorID, string(record.Run.Status), reason); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) processGitOps(ctx context.Context, record RunRecord, actorID string, allowSync bool, confirm bool) (RunRecord, error) {
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPlanning, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	record.GitOpsPlan = s.buildGitOpsPlan(record.Run.ID, record.Definition, nil)
	record.Plan = deploymentPlanFromGitOps(record.GitOpsPlan, record.Definition)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", "GitOps change plan created", int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsPlanCreated, "GitOps deployment planned", actorID, string(record.Run.Status), "GitOps change plan created"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunPreChecking, s.now(), ""); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentPrecheckCompleted, "Deployment policy pre-check completed", actorID, string(record.Run.Status), "GitOps policy pre-check allowed"); err != nil {
		return RunRecord{}, err
	}
	record, _ = s.store.Get(ctx, record.Run.ID)

	if record.Definition.Spec.GitOps.Rollback {
		result, err := s.rollbackGitOpsRevision(ctx, record.Definition, confirm)
		if err != nil {
			return s.fail(ctx, record, actorID, err.Error())
		}
		record.GitOpsRollback = result
		record.GitOpsPlan.RollbackRevision = record.Definition.Spec.GitOps.RollbackRevision
		record.GitOpsPlan.CommitRevision = result.Revision
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("GitOps rollback revision checked out: %s", result.Revision), int64(len(record.Logs)+1)))
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsRollbackPlanned, "GitOps rollback planned", actorID, string(record.Run.Status), "GitOps rollback by revision planned"); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsRollbackCompleted, "GitOps rollback revision checked out", actorID, string(record.Run.Status), result.Revision); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)
	}

	if record.Definition.Spec.GitOps.WriteToWorkingTree {
		updated, diff, err := s.applyGitOpsWorkingTree(ctx, record.GitOpsPlan, record.Definition)
		if err != nil {
			return s.fail(ctx, record, actorID, err.Error())
		}
		record.GitOpsPlan = updated
		record.GitOpsDiff = diff
		record.Plan.Warnings = append(record.Plan.Warnings, updated.Warnings...)
		record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", diff.Summary, int64(len(record.Logs)+1)))
		if err := s.store.Save(ctx, record); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsDiffGenerated, "GitOps diff generated", actorID, string(record.Run.Status), diff.Summary); err != nil {
			return RunRecord{}, err
		}
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsWorkingTreeUpdated, "GitOps working tree changed", actorID, string(record.Run.Status), "GitOps working tree updated locally"); err != nil {
			return RunRecord{}, err
		}
		record, _ = s.store.Get(ctx, record.Run.ID)

		if record.Definition.Spec.GitOps.Commit {
			commit, err := s.commitGitOpsChanges(ctx, record.GitOpsDiff, record.Definition)
			if err != nil {
				return s.fail(ctx, record, actorID, err.Error())
			}
			record.GitOpsCommit = commit
			record.GitOpsPlan.CommitRevision = commit.Revision
			record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("GitOps commit completed: %s", commit.Revision), int64(len(record.Logs)+1)))
			if err := s.store.Save(ctx, record); err != nil {
				return RunRecord{}, err
			}
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsCommitCreated, "GitOps commit created", actorID, string(record.Run.Status), commit.Revision); err != nil {
				return RunRecord{}, err
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
		}
		if record.Definition.Spec.GitOps.Push {
			if !record.Definition.Spec.GitOps.AllowPush || !confirm {
				return s.fail(ctx, record, actorID, "gitops push requires gitops.allowPush=true and confirmation")
			}
			push, err := s.gitops.Push(ctx, record.Definition.Spec.GitOps.WorkingTree, record.Definition.Spec.GitOps.Remote, record.Definition.Spec.GitOps.Branch, true)
			if err != nil {
				return s.fail(ctx, record, actorID, err.Error())
			}
			record.GitOpsPush = push
			record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("GitOps push completed: %s", push.Revision), int64(len(record.Logs)+1)))
			if err := s.store.Save(ctx, record); err != nil {
				return RunRecord{}, err
			}
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsPushCompleted, "GitOps push completed", actorID, string(record.Run.Status), push.Revision); err != nil {
				return RunRecord{}, err
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
		} else if record.Definition.Spec.GitOps.AllowPush {
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventGitOpsPushSkipped, "GitOps push skipped", actorID, string(record.Run.Status), "push=false; remote Git push was not requested"); err != nil {
				return RunRecord{}, err
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
		}
	}

	if s.argocd != nil && (record.Definition.Spec.GitOps.StatusRead || record.Definition.Spec.GitOps.Sync) {
		if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDStatusReadStarted, "Argo CD status read started", actorID, string(record.Run.Status), "Argo CD application status read started"); err != nil {
			return RunRecord{}, err
		}
		status, err := s.argocd.GetApplicationStatus(ctx, record.Definition.Spec.Target.ApplicationName)
		if err != nil {
			record, _ = s.recordRuntimeEvent(ctx, record, EventArgoCDStatusReadFailed, string(record.Run.Status), err.Error())
			if record.Definition.Spec.GitOps.RequireStatus || record.Definition.Spec.GitOps.Sync {
				return s.fail(ctx, record, actorID, err.Error())
			}
			record.Plan.Warnings = append(record.Plan.Warnings, "Argo CD status read failed: "+err.Error())
			if err := s.store.Save(ctx, record); err != nil {
				return RunRecord{}, err
			}
		} else {
			resources, _ := s.argocd.GetApplicationResources(ctx, record.Definition.Spec.Target.ApplicationName)
			record.ArgoCD = status
			record.ArgoCDResources = resources
			record.GitOpsPlan.Status = status
			record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", status.Message, int64(len(record.Logs)+1)))
			if err := s.store.Save(ctx, record); err != nil {
				return RunRecord{}, err
			}
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDStatusReadCompleted, "Argo CD status read completed", actorID, string(record.Run.Status), status.Message); err != nil {
				return RunRecord{}, err
			}
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDStatusRead, "Argo CD status read", actorID, string(record.Run.Status), status.Message); err != nil {
				return RunRecord{}, err
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
		}
	}

	if record.Definition.Spec.GitOps.Sync {
		if s.argocd == nil {
			return s.fail(ctx, record, actorID, "argocd provider is not configured")
		}
		if record.Definition.Spec.GitOps.Force {
			return s.fail(ctx, record, actorID, "argocd force sync is not supported in Phase 2.6")
		}
		request := portargocd.SyncRequest{
			ApplicationName: record.Definition.Spec.Target.ApplicationName,
			Revision:        record.Definition.Spec.Target.Revision,
			Prune:           record.Definition.Spec.GitOps.Prune,
			Force:           record.Definition.Spec.GitOps.Force,
			Wait:            record.Definition.Spec.GitOps.Wait,
			TimeoutSeconds:  record.Definition.Spec.GitOps.TimeoutSeconds,
			AllowSync:       allowSync && record.Definition.Spec.GitOps.AllowSync,
			Confirmed:       confirm,
		}
		if !request.AllowSync || !request.Confirmed {
			result := portargocd.SyncResult{ApplicationName: request.ApplicationName, Requested: false, Message: "Argo CD sync skipped; sync requires gitops.allowSync=true plus API/CLI allow-sync and confirmation"}
			record.ArgoCDSync = result
			record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", result.Message, int64(len(record.Logs)+1)))
			if err := s.store.Save(ctx, record); err != nil {
				return RunRecord{}, err
			}
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDSyncSkipped, "Argo CD sync skipped", actorID, string(record.Run.Status), result.Message); err != nil {
				return RunRecord{}, err
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
		} else {
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDSyncRequested, "Argo CD sync requested", actorID, string(record.Run.Status), "Guarded Argo CD sync requested"); err != nil {
				return RunRecord{}, err
			}
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDSyncStarted, "Argo CD sync started", actorID, string(record.Run.Status), "Guarded Argo CD sync started"); err != nil {
				return RunRecord{}, err
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
			result, err := s.argocd.SyncApplication(ctx, portargocd.SyncRequest{
				ApplicationName: request.ApplicationName,
				Revision:        request.Revision,
				Prune:           request.Prune,
				Force:           request.Force,
				Wait:            request.Wait,
				TimeoutSeconds:  request.TimeoutSeconds,
				AllowSync:       request.AllowSync,
				Confirmed:       request.Confirmed,
			})
			if err != nil {
				record, _ = s.recordRuntimeEvent(ctx, record, EventArgoCDSyncFailed, string(record.Run.Status), err.Error())
				return s.fail(ctx, record, actorID, err.Error())
			}
			record.ArgoCDSync = result
			record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", result.Message, int64(len(record.Logs)+1)))
			if err := s.store.Save(ctx, record); err != nil {
				return RunRecord{}, err
			}
			if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDSyncCompleted, "Argo CD sync completed", actorID, string(record.Run.Status), result.Message); err != nil {
				return RunRecord{}, err
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
			if record.Definition.Spec.GitOps.Wait {
				watch, err := s.argocd.WatchApplicationStatus(ctx, request.ApplicationName, request.TimeoutSeconds)
				if err != nil {
					record, _ = s.recordRuntimeEvent(ctx, record, EventArgoCDSyncFailed, string(record.Run.Status), err.Error())
					return s.fail(ctx, record, actorID, err.Error())
				}
				record.ArgoCDWatch = watch
				if len(watch) > 0 {
					last := watch[len(watch)-1]
					record.ArgoCD = last
					record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("Argo CD watch completed: sync=%s health=%s", last.SyncStatus, last.HealthStatus), int64(len(record.Logs)+1)))
					if err := s.store.Save(ctx, record); err != nil {
						return RunRecord{}, err
					}
					if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDHealthChanged, "Argo CD health changed", actorID, string(record.Run.Status), last.HealthStatus); err != nil {
						return RunRecord{}, err
					}
					if last.SyncStatus == "OutOfSync" || last.HealthStatus == "Degraded" {
						return s.fail(ctx, record, actorID, fmt.Sprintf("Argo CD watch ended unhealthy: sync=%s health=%s", last.SyncStatus, last.HealthStatus))
					}
				}
			}
			record, _ = s.store.Get(ctx, record.Run.ID)
		}
	} else if err := s.recordEventAndAudit(ctx, record.Run.ID, EventArgoCDSyncSkipped, "Argo CD sync skipped", actorID, string(record.Run.Status), "sync=false; Argo CD sync was not requested"); err != nil {
		return RunRecord{}, err
	}

	record, _ = s.store.Get(ctx, record.Run.ID)
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunVerifying, s.now(), "GitOps plan verification completed"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunSucceeded, s.now(), "gitops deployment plan completed"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, record.Run.ID, EventDeploymentSucceeded, "DeploymentRun succeeded", actorID, string(record.Run.Status), "GitOps DeploymentRun succeeded"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, record.Run.ID)
}

func (s *Service) Get(ctx context.Context, id string) (RunRecord, error) { return s.store.Get(ctx, id) }
func (s *Service) List(ctx context.Context) ([]RunRecord, error)         { return s.store.List(ctx) }
func (s *Service) ListFiltered(ctx context.Context, scopeType, scopeID string) ([]RunRecord, error) {
	return s.store.ListFiltered(ctx, scopeType, scopeID)
}
func (s *Service) Logs(ctx context.Context, id string) ([]event.LogChunk, error) {
	return s.store.Logs(ctx, id)
}
func (s *Service) Events(ctx context.Context, id string) ([]event.Event, error) {
	return s.store.Events(ctx, id)
}

func (s *Service) Resources(ctx context.Context, id string) ([]ManifestResourceSummary, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]ManifestResourceSummary(nil), record.Plan.Resources...), nil
}

func (s *Service) Health(ctx context.Context, id string) (HealthEvaluation, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return HealthEvaluation{}, err
	}
	return record.Health, nil
}

func (s *Service) Diff(ctx context.Context, id string) (DeploymentDiff, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return DeploymentDiff{}, err
	}
	return record.Diff, nil
}

func (s *Service) Snapshot(ctx context.Context, id string) (ManifestSnapshot, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return ManifestSnapshot{}, err
	}
	return record.Snapshot, nil
}

func (s *Service) RollbackPlan(ctx context.Context, id string) (RollbackPlan, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RollbackPlan{}, err
	}
	return record.RollbackPlan, nil
}

func (s *Service) ArgoCDStatus(ctx context.Context, applicationName string) (portargocd.ApplicationStatus, error) {
	if s.argocd == nil {
		return portargocd.ApplicationStatus{}, fmt.Errorf("argocd provider is not configured")
	}
	return s.argocd.GetApplicationStatus(ctx, applicationName)
}

func (s *Service) ArgoCDResources(ctx context.Context, applicationName string) ([]portargocd.ResourceStatus, error) {
	if s.argocd == nil {
		return nil, fmt.Errorf("argocd provider is not configured")
	}
	return s.argocd.GetApplicationResources(ctx, applicationName)
}

func (s *Service) SyncArgoCDApplication(ctx context.Context, request portargocd.SyncRequest) (portargocd.SyncResult, error) {
	if s.argocd == nil {
		return portargocd.SyncResult{}, fmt.Errorf("argocd provider is not configured")
	}
	if request.Force {
		return portargocd.SyncResult{}, fmt.Errorf("argocd force sync is not supported in Phase 2.6")
	}
	if !request.AllowSync || !request.Confirmed {
		return portargocd.SyncResult{}, fmt.Errorf("argocd sync requires allowSync=true and confirmed=true")
	}
	return s.argocd.SyncApplication(ctx, request)
}

func (s *Service) SyncDeployment(ctx context.Context, id string, actorID string, allowSync bool, confirm bool) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if record.Definition.Spec.Target.Type != "argocd" {
		return RunRecord{}, fmt.Errorf("deployment %s is not an argocd target", id)
	}
	record.Definition.Spec.GitOps.Sync = true
	record.Definition.Spec.GitOps.AllowSync = record.Definition.Spec.GitOps.AllowSync || allowSync
	record.Run.Status = domaindeployment.DeploymentRunCreated
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	return s.processGitOps(ctx, record, actorID, allowSync, confirm)
}

func (s *Service) CreateHostGroup(ctx context.Context, group HostGroup) (HostGroup, error) {
	if strings.TrimSpace(group.Name) == "" {
		return HostGroup{}, fmt.Errorf("host group name is required")
	}
	if len(group.Hosts) == 0 {
		return HostGroup{}, fmt.Errorf("host group requires at least one host")
	}
	now := s.now()
	if group.ID == "" {
		group.ID = newID("hostgrp")
	}
	if group.CreatedAt.IsZero() {
		group.CreatedAt = now
	}
	group.UpdatedAt = now
	for i := range group.Hosts {
		if group.Hosts[i].ID == "" {
			group.Hosts[i].ID = newID("host")
		}
		if group.Hosts[i].Name == "" {
			return HostGroup{}, fmt.Errorf("host %d name is required", i)
		}
		if group.Hosts[i].EnvironmentID == "" {
			group.Hosts[i].EnvironmentID = group.EnvironmentID
		}
		if group.Hosts[i].CredentialRef == "" {
			group.Hosts[i].CredentialRef = group.CredentialRef
		}
	}
	if err := s.store.SaveHostGroup(ctx, group); err != nil {
		return HostGroup{}, err
	}
	return s.store.GetHostGroup(ctx, group.ID)
}

func (s *Service) GetHostGroup(ctx context.Context, id string) (HostGroup, error) {
	return s.store.GetHostGroup(ctx, id)
}

func (s *Service) ListHostGroups(ctx context.Context) ([]HostGroup, error) {
	return s.store.ListHostGroups(ctx)
}

func (s *Service) Hosts(ctx context.Context, id string) ([]HostDeploymentRunDetail, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]HostDeploymentRunDetail(nil), record.HostDetails...), nil
}

func (s *Service) Timeline(ctx context.Context, id string) ([]TimelineEntry, error) {
	events, err := s.Events(ctx, id)
	if err != nil {
		return nil, err
	}
	timeline := make([]TimelineEntry, 0, len(events))
	for _, evt := range events {
		entry := TimelineEntry{Type: evt.Type, Time: evt.Time, Subject: evt.Subject}
		if status, ok := evt.Data["status"].(string); ok {
			entry.Status = status
		}
		if message, ok := evt.Data["message"].(string); ok {
			entry.Message = message
		}
		timeline = append(timeline, entry)
	}
	return timeline, nil
}

func (s *Service) Cancel(ctx context.Context, id string, actorID string) (RunRecord, error) {
	record, err := s.store.Get(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if isTerminalDeploymentStatus(record.Run.Status) {
		return record, ErrRunTerminal
	}
	if err := transitionDeploymentRun(&record.Run, domaindeployment.DeploymentRunCanceled, s.now(), "canceled by request"); err != nil {
		return RunRecord{}, err
	}
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	if err := s.recordEventAndAudit(ctx, id, EventDeploymentCanceled, "DeploymentRun canceled", actorID, string(record.Run.Status), "DeploymentRun canceled"); err != nil {
		return RunRecord{}, err
	}
	return s.store.Get(ctx, id)
}

func (s *Service) newRecord(def Definition) RunRecord {
	now := s.now()
	releaseID := newID("rel")
	runID := newID("drun")
	envID := newID("env")
	targetID := newID("target")
	artifactRefs := make([]string, 0, len(def.Spec.Artifacts))
	artifacts := make([]release.ReleaseArtifact, 0, len(def.Spec.Artifacts))
	for _, artifact := range def.Spec.Artifacts {
		artifactRefs = append(artifactRefs, artifact.Reference)
		artifacts = append(artifacts, release.ReleaseArtifact{
			ID:        newID("artifact"),
			ReleaseID: releaseID,
			Name:      artifact.Name,
			Type:      artifact.Type,
			Role:      "deployment",
			Required:  true,
			Reference: artifact.Reference,
			Digest:    artifact.Digest,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return RunRecord{
		Definition: def,
		Release: release.Release{
			ID:            releaseID,
			Name:          def.Metadata.Name,
			ApplicationID: def.Spec.Application,
			Version:       def.Metadata.Name,
			Status:        "Created",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Artifacts: artifacts,
		Environment: environment.Environment{
			ID:        envID,
			Name:      def.Spec.Environment,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Target: environment.ReleaseTarget{
			ID:            targetID,
			EnvironmentID: envID,
			Name:          def.Spec.Target.Name,
			TargetType:    def.Spec.Target.Type,
			Context:       def.Spec.Target.Context,
			Namespace:     def.Spec.Target.Namespace,
			Enabled:       true,
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		Run: domaindeployment.DeploymentRun{
			ID:                  runID,
			ReleaseID:           releaseID,
			ApplicationID:       def.Spec.Application,
			EnvironmentID:       envID,
			ReleaseTargetID:     targetID,
			TargetType:          def.Spec.Target.Type,
			Status:              domaindeployment.DeploymentRunCreated,
			ArtifactReferences:  artifactRefs,
			ManifestSnapshotRef: "memory://" + runID + "/manifests",
			CreatedAt:           now,
			UpdatedAt:           now,
		},
	}
}

func applyProjectScope(record RunRecord, projectID string) RunRecord {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return record
	}
	record.Environment.ProjectID = projectID
	record.Target.ProjectID = projectID
	return record
}

func (s *Service) buildPlan(runID string, def Definition, docs []ManifestDocument) DeploymentPlan {
	artifacts := make([]string, 0, len(def.Spec.Artifacts))
	artifactDetails := make([]domainartifact.Inspection, 0, len(def.Spec.Artifacts))
	for _, artifact := range def.Spec.Artifacts {
		reference := artifact.Reference
		if artifact.Digest != "" && !containsDigest(reference) {
			reference = reference + "@" + artifact.Digest
		}
		inspection, err := domainartifact.InspectReference(reference, domainartifact.ArtifactType(artifact.Type))
		if err != nil {
			warnings := []string{"live cluster diff is not implemented in Phase 2.4", fmt.Sprintf("artifact %q could not be inspected: %v", artifact.Name, err)}
			return DeploymentPlan{
				DeploymentRunID: runID,
				TargetType:      def.Spec.Target.Type,
				TargetContext:   def.Spec.Target.Context,
				Namespace:       def.Spec.Target.Namespace,
				Artifacts:       []string{artifact.Reference},
				DryRun:          def.Spec.Options.dryRunOnly(),
				Apply:           def.Spec.Options.Apply,
				Wait:            def.Spec.Options.Wait,
				TimeoutSeconds:  def.Spec.Options.TimeoutSeconds,
				Actions:         []string{"render manifests", "validate manifests", "policy pre-check", "server-side dry-run"},
				Warnings:        warnings,
				DiffSummary:     "artifact inspection failed",
			}
		}
		artifacts = append(artifacts, inspection.Reference.Normalized)
		artifactDetails = append(artifactDetails, inspection)
	}
	resources := make([]ManifestResourceSummary, 0, len(docs))
	for _, doc := range docs {
		resources = append(resources, doc.Resource)
	}
	manifestImages := ExtractManifestImages(docs)
	warnings := []string{"live cluster diff is not implemented in Phase 2.4"}
	for _, detail := range artifactDetails {
		for _, warning := range detail.Warnings {
			warnings = append(warnings, fmt.Sprintf("artifact %s: %s", detail.Reference.Normalized, warning.Message))
		}
	}
	warnings = append(warnings, verifyManifestImages(def.Spec.Artifacts, artifactDetails, manifestImages)...)
	if def.Spec.Options.Apply {
		warnings = append(warnings, "apply requested; Phase 6.0 apply requires explicit confirmation and uses the configured manifest client")
		if def.Spec.Target.Context == "" && def.Spec.Target.ClusterName == "" && def.Spec.Target.ClusterURL == "" {
			warnings = append(warnings, "apply target should specify an explicit context, clusterName, or clusterURL before using a real Kubernetes adapter")
		}
	}
	actions := []string{"render manifests", "validate manifests", "policy pre-check", "server-side dry-run"}
	if def.Spec.Options.Apply {
		actions = append(actions, "apply manifests")
	}
	if def.Spec.Options.Apply && def.Spec.Options.Wait {
		actions = append(actions, "rollout verification")
	}
	return DeploymentPlan{
		DeploymentRunID: runID,
		TargetType:      def.Spec.Target.Type,
		TargetContext:   def.Spec.Target.Context,
		Namespace:       def.Spec.Target.Namespace,
		ManifestCount:   len(docs),
		Resources:       resources,
		Artifacts:       artifacts,
		ArtifactDetails: artifactDetails,
		ManifestImages:  manifestImages,
		DryRun:          def.Spec.Options.dryRunOnly(),
		Apply:           def.Spec.Options.Apply,
		Wait:            def.Spec.Options.Wait,
		TimeoutSeconds:  def.Spec.Options.TimeoutSeconds,
		Actions:         actions,
		Warnings:        warnings,
		DiffSummary:     fmt.Sprintf("desired state contains %d manifest resource(s); live diff is not available in Phase 2.4", len(docs)),
	}
}

func (s *Service) enforceKubernetesSafety(ctx context.Context, record RunRecord, docs []ManifestDocument, actorID string) (RunRecord, bool, error) {
	result := DefaultK8sSafetyPolicy().ValidateManifests(ctx, docs, record.Definition.Spec.Target.Namespace)
	record.Plan = appendKubernetesSafetyResult(record.Plan, result)
	if result.Allowed {
		return record, true, nil
	}
	reason := "kubernetes safety policy denied"
	if message := firstFailedKubernetesSafetyMessage(result); message != "" {
		reason += ": " + message
	}
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", reason, int64(len(record.Logs)+1)))
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, false, err
	}
	failed, err := s.fail(ctx, record, actorID, reason)
	return failed, false, err
}

func appendKubernetesSafetyWarnings(plan DeploymentPlan, docs []ManifestDocument, namespace string) DeploymentPlan {
	result := DefaultK8sSafetyPolicy().ValidateManifests(context.Background(), docs, namespace)
	return appendKubernetesSafetyResult(plan, result)
}

func appendKubernetesSafetyResult(plan DeploymentPlan, result K8sSafetyResult) DeploymentPlan {
	if result.Allowed {
		return plan
	}
	message := "kubernetes safety policy denied"
	if detail := firstFailedKubernetesSafetyMessage(result); detail != "" {
		message += ": " + detail
	}
	if !containsString(plan.Warnings, message) {
		plan.Warnings = append(plan.Warnings, message)
	}
	return plan
}

func firstFailedKubernetesSafetyMessage(result K8sSafetyResult) string {
	for _, check := range result.Checks {
		if !check.Passed {
			return check.Message
		}
	}
	if len(result.Warnings) > 0 {
		return result.Warnings[0]
	}
	return ""
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func (s *Service) buildHostPlan(runID string, def Definition) (DeploymentPlan, HostDeploymentPlan, RollbackPlan) {
	def = normalizeDefinition(def)
	artifact := def.Spec.Artifacts[0].Reference
	deployPath := strings.TrimRight(def.Spec.Host.DeployPath, "/")
	if deployPath == "" {
		deployPath = "/opt/nivora/apps/" + def.Spec.Application
	}
	strategy := def.Spec.Host.Strategy
	if strategy == "" {
		strategy = "symlink"
	}
	batchSize := def.Spec.Host.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}
	pauseOnFailure := true
	if strings.EqualFold(def.Spec.Host.Metadata["pauseOnFailure"], "false") {
		pauseOnFailure = false
	}
	healthChecks := normalizeHostHealthChecks(def.Spec.Host)
	timeoutSeconds := def.Spec.Options.TimeoutSeconds
	actions := []string{"validate artifact reference", "validate host targets", "prepare versioned release directory", "plan current/previous/next symlink switch", "plan health checks", "create rollback baseline"}
	if def.Spec.Host.ServiceName != "" || def.Spec.Host.RestartCommand != "" {
		actions = append(actions, "plan guarded service restart")
	}
	if def.Spec.Options.Apply {
		actions = append(actions, "upload artifact", "switch symlink", "restart service when configured", "run health checks")
	}
	warnings := []string{"remote SSH execution is disabled by default unless explicitly confirmed and credentialed", "host rollback is guarded and uses symlink restore without deleting release directories by default"}
	if !def.Spec.Options.Apply {
		warnings = append(warnings, "host deployment is plan/dry-run only")
	}
	steps := make([]HostDeploymentStep, 0, len(def.Spec.Host.Hosts))
	resources := make([]ManifestResourceSummary, 0, len(def.Spec.Host.Hosts))
	for _, host := range def.Spec.Host.Hosts {
		hostID := host.ID
		if hostID == "" {
			hostID = host.Name
		}
		releaseDir := deployPath + "/releases/" + runID
		batchIndex := len(steps)/batchSize + 1
		steps = append(steps, HostDeploymentStep{
			HostID:          hostID,
			HostName:        host.Name,
			Address:         host.Address,
			BatchIndex:      batchIndex,
			ReleaseDir:      releaseDir,
			CurrentSymlink:  deployPath + "/current",
			PreviousSymlink: deployPath + "/previous",
			NextSymlink:     deployPath + "/next",
			TimeoutSeconds:  timeoutSeconds,
			Actions:         append([]string(nil), actions...),
		})
		resources = append(resources, ManifestResourceSummary{
			APIVersion:  "nivora.io/v1alpha1",
			Kind:        "Host",
			Name:        host.Name,
			Namespace:   def.Spec.Environment,
			Labels:      cloneStringMap(host.Labels),
			DesiredHash: stableHostHash(host, artifact, deployPath),
			Status:      "Planned",
			Health:      ResourceHealthUnknown,
			CreatedAt:   s.now(),
			UpdatedAt:   s.now(),
		})
	}
	rollbackPlan := RollbackPlan{
		DeploymentRunID:   runID,
		CurrentSnapshotID: "memory://" + runID + "/host-release-plan",
		TargetType:        def.Spec.Target.Type,
		TargetName:        def.Spec.Target.Name,
		Resources:         resources,
		Strategy:          "symlink-restore",
		Executable:        false,
		Warnings:          []string{"host rollback is guarded by explicit confirmation and does not delete release directories by default"},
		CreatedAt:         s.now(),
	}
	hostPlan := HostDeploymentPlan{
		DeploymentRunID: runID,
		GroupName:       def.Spec.Target.Name,
		EnvironmentID:   def.Spec.Environment,
		Artifact:        artifact,
		DeployPath:      deployPath,
		ServiceName:     def.Spec.Host.ServiceName,
		HealthCheck:     def.Spec.Host.HealthCheck,
		HealthChecks:    healthChecks,
		RestartCommand:  def.Spec.Host.RestartCommand,
		Strategy:        strategy,
		BatchSize:       batchSize,
		PauseOnFailure:  pauseOnFailure,
		DryRun:          !def.Spec.Options.Apply,
		Apply:           def.Spec.Options.Apply,
		Hosts:           steps,
		Actions:         actions,
		Warnings:        warnings,
		RollbackPlan:    rollbackPlan,
	}
	plan := DeploymentPlan{
		DeploymentRunID: runID,
		TargetType:      def.Spec.Target.Type,
		TargetContext:   def.Spec.Target.Name,
		ManifestCount:   0,
		Resources:       resources,
		Artifacts:       []string{artifact},
		DryRun:          !def.Spec.Options.Apply,
		Apply:           def.Spec.Options.Apply,
		Wait:            def.Spec.Options.Wait,
		TimeoutSeconds:  def.Spec.Options.TimeoutSeconds,
		Actions:         actions,
		Warnings:        warnings,
		DiffSummary:     fmt.Sprintf("host deployment plan contains %d host(s); live host diff is not implemented in Phase 3.5", len(steps)),
	}
	return plan, hostPlan, rollbackPlan
}

func (s *Service) attachResourceObservability(record RunRecord, def Definition, docs []ManifestDocument) RunRecord {
	now := s.now()
	record.Inventory = ResourceInventory{
		DeploymentRunID: record.Run.ID,
		Desired:         append([]ManifestResourceSummary(nil), record.Plan.Resources...),
		Applied:         append([]ManifestResourceSummary(nil), record.Plan.Resources...),
		Warnings:        []string{"live Kubernetes resource observation is not required in Phase 2.4"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	record.Snapshot = buildManifestSnapshot(record.Run.ID, docs, now)
	record.Run.ManifestSnapshotRef = record.Snapshot.StorageRef
	record.Diff = buildDeploymentDiff(record.Run.ID, record.Plan.Resources)
	record.Rollback = s.rollbackBaseline(record.Run.ID, def, record.Plan.Resources)
	record.RollbackPlan = RollbackPlan{
		DeploymentRunID:   record.Run.ID,
		CurrentSnapshotID: record.Snapshot.ID,
		TargetType:        def.Spec.Target.Type,
		TargetName:        def.Spec.Target.Name,
		Resources:         append([]ManifestResourceSummary(nil), record.Plan.Resources...),
		Strategy:          "manifest-restore",
		Executable:        def.Spec.Options.Apply,
		Warnings:          []string{"rollback execution requires explicit confirmation and restores manifests without prune/delete by default"},
		CreatedAt:         now,
	}
	record.Health = evaluateResourceHealth(record.Run.ID, record.Plan.Resources, now)
	return record
}

func buildManifestSnapshot(runID string, docs []ManifestDocument, now time.Time) ManifestSnapshot {
	var content strings.Builder
	for _, doc := range docs {
		content.WriteString("---\n")
		content.WriteString(doc.Content)
		if !strings.HasSuffix(doc.Content, "\n") {
			content.WriteString("\n")
		}
	}
	sum := sha256.Sum256([]byte(content.String()))
	inline := content.String()
	if len(inline) > 64*1024 {
		inline = ""
	}
	return ManifestSnapshot{
		ID:              newID("snapshot"),
		DeploymentRunID: runID,
		ContentHash:     "sha256:" + hex.EncodeToString(sum[:]),
		DocumentCount:   len(docs),
		ResourceCount:   len(docs),
		StorageRef:      "memory://" + runID + "/manifest-snapshot",
		InlineContent:   inline,
		CreatedAt:       now,
	}
}

func buildDeploymentDiff(runID string, resources []ManifestResourceSummary) DeploymentDiff {
	refs := make([]string, 0, len(resources))
	for _, resource := range resources {
		refs = append(refs, resourceRef(resource))
	}
	return DeploymentDiff{
		DeploymentRunID:  runID,
		AddedResources:   append([]string(nil), refs...),
		UnknownLiveState: append([]string(nil), refs...),
		Summary:          fmt.Sprintf("desired state contains %d resource(s); live state is unknown in Phase 2.4", len(resources)),
		Warnings:         []string{"live Kubernetes diff is not implemented in Phase 2.4"},
	}
}

func evaluateResourceHealth(runID string, resources []ManifestResourceSummary, now time.Time) HealthEvaluation {
	eval := HealthEvaluation{DeploymentRunID: runID, Status: ResourceHealthUnknown, EvaluatedAt: now}
	for _, resource := range resources {
		health := defaultHealth(resource.Kind)
		if health == ResourceHealthHealthy {
			eval.Healthy++
		} else if health == ResourceHealthDegraded {
			eval.Degraded++
		}
		summary := ResourceHealthSummary{
			Resource:       resource,
			DesiredExists:  true,
			LiveExists:     false,
			Health:         health,
			DiffSummary:    "live state not observed",
			LastObservedAt: now,
			Warnings:       []string{"live Kubernetes health is not observed in the default Phase 2.4 runtime"},
		}
		eval.Resources = append(eval.Resources, summary)
		eval.ResourcesChecked++
	}
	if eval.ResourcesChecked == 0 {
		eval.Warnings = append(eval.Warnings, "no resources to evaluate")
		return eval
	}
	eval.Status = ResourceHealthProgressing
	if eval.Healthy == eval.ResourcesChecked {
		eval.Status = ResourceHealthHealthy
	}
	if eval.Degraded > 0 {
		eval.Status = ResourceHealthDegraded
	}
	return eval
}

func defaultHealth(kind string) ResourceHealth {
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "Job", "CronJob":
		return ResourceHealthProgressing
	case "Service", "ConfigMap", "Secret", "Ingress", "Namespace", "ServiceAccount", "Role", "RoleBinding", "ClusterRole", "ClusterRoleBinding":
		return ResourceHealthHealthy
	default:
		return ResourceHealthUnsupported
	}
}

func rolloutFromHealth(health HealthEvaluation) RolloutResult {
	return RolloutResult{
		Mode:      "local-health",
		Message:   fmt.Sprintf("health evaluation completed with status %s", health.Status),
		Warnings:  append([]string(nil), health.Warnings...),
		Resources: healthResources(health.Resources),
	}
}

func healthResources(items []ResourceHealthSummary) []ManifestResourceSummary {
	resources := make([]ManifestResourceSummary, 0, len(items))
	for _, item := range items {
		resource := item.Resource
		resource.Health = item.Health
		resources = append(resources, resource)
	}
	return resources
}

func resourceRef(resource ManifestResourceSummary) string {
	if resource.Namespace == "" {
		return fmt.Sprintf("%s/%s", resource.Kind, resource.Name)
	}
	return fmt.Sprintf("%s/%s/%s", resource.Kind, resource.Namespace, resource.Name)
}

func (s *Service) resolveGitOpsRepository(ctx context.Context, def Definition, projectID string) (Definition, error) {
	if def.Spec.Target.Type != "argocd" || strings.TrimSpace(def.Spec.Target.RepositoryID) == "" {
		return def, nil
	}
	if s.repos == nil {
		return Definition{}, fmt.Errorf("%w: repository catalog is not configured for target.repositoryId", ErrInvalidInput)
	}
	repositoryID := strings.TrimSpace(def.Spec.Target.RepositoryID)
	repository, err := s.repos.GetRepository(ctx, repositoryID)
	if err != nil {
		return Definition{}, fmt.Errorf("%w: repository %q could not be resolved: %v", ErrInvalidInput, repositoryID, err)
	}
	if projectID = strings.TrimSpace(projectID); projectID != "" && repository.ProjectID != projectID {
		return Definition{}, fmt.Errorf("%w: repository %q is not accessible for project %q", ErrInvalidInput, repositoryID, projectID)
	}
	if !repository.Enabled {
		return Definition{}, fmt.Errorf("%w: repository %q is disabled", ErrInvalidInput, repositoryID)
	}
	if strings.TrimSpace(repository.URL) == "" {
		return Definition{}, fmt.Errorf("%w: repository %q has no URL", ErrInvalidInput, repositoryID)
	}
	if def.Spec.Target.RepoURL == "" {
		def.Spec.Target.RepoURL = repository.URL
	}
	if def.Spec.Target.Revision == "" && repository.DefaultBranch != "" {
		def.Spec.Target.Revision = repository.DefaultBranch
	}
	def.Spec.Target.RepositoryID = repository.ID
	def.Spec.Target.RepositoryName = repository.Name
	def.Spec.Target.RepositoryProvider = repository.Provider
	return def, nil
}

func (s *Service) buildGitOpsPlan(runID string, def Definition, changes []portgitops.FileChange) GitOpsChangePlan {
	files := append([]string(nil), def.Spec.GitOps.Files...)
	if len(files) == 0 && def.Spec.Target.Path != "" {
		files = append(files, strings.Trim(def.Spec.Target.Path, "/")+"/deployment.yaml")
	}
	artifacts := make([]string, 0, len(def.Spec.Artifacts))
	for _, artifact := range def.Spec.Artifacts {
		artifacts = append(artifacts, artifact.Reference)
	}
	warnings := []string{"GitOps plan-only mode is the safe Phase 2.3 default"}
	if def.Spec.GitOps.WriteToWorkingTree && def.Spec.GitOps.WorkingTree == "" {
		warnings = append(warnings, "writeToWorkingTree=true requires gitops.workingTree")
	}
	if def.Spec.GitOps.Commit {
		warnings = append(warnings, "GitOps commit requested; local commit only, push remains disabled unless explicitly allowed")
	}
	if def.Spec.GitOps.Push {
		warnings = append(warnings, "GitOps push requested; push requires allowPush=true and confirmation")
	}
	if def.Spec.GitOps.Rollback {
		warnings = append(warnings, "GitOps rollback by revision requested; checkout requires confirmation and does not sync by default")
	}
	if def.Spec.GitOps.Sync {
		warnings = append(warnings, "Argo CD sync requested; sync is disabled unless explicitly allowed and confirmed")
	}
	if def.Spec.GitOps.Prune {
		warnings = append(warnings, "Argo CD prune requested; prune is guarded and defaults to false")
	}
	if def.Spec.GitOps.Force {
		warnings = append(warnings, "Argo CD force sync is not supported in Phase 2.6")
	}
	if def.Spec.Target.RepositoryID != "" {
		warnings = append(warnings, "GitOps repository was resolved from the catalog by repositoryId; provider credentials remain behind CredentialRef metadata")
	}
	return GitOpsChangePlan{
		DeploymentRunID:       runID,
		ApplicationName:       def.Spec.Target.ApplicationName,
		RepositoryID:          def.Spec.Target.RepositoryID,
		RepositoryName:        def.Spec.Target.RepositoryName,
		RepositoryProvider:    def.Spec.Target.RepositoryProvider,
		RepoURL:               def.Spec.Target.RepoURL,
		Path:                  def.Spec.Target.Path,
		Revision:              def.Spec.Target.Revision,
		Files:                 files,
		FileChanges:           changes,
		ArtifactChanges:       artifacts,
		ManifestValueChanges:  plannedImageChanges(def),
		CommitMessageProposal: fmt.Sprintf("chore: update %s release artifacts", def.Spec.Application),
		RollbackRevision:      def.Spec.GitOps.RollbackRevision,
		DryRun:                !def.Spec.GitOps.WriteToWorkingTree,
		Warnings:              warnings,
		SyncRequested:         def.Spec.GitOps.Sync,
	}
}

func deploymentPlanFromGitOps(plan GitOpsChangePlan, def Definition) DeploymentPlan {
	actions := []string{"build GitOps change plan", "policy pre-check"}
	if def.Spec.GitOps.WriteToWorkingTree {
		actions = append(actions, "update local working tree", "generate diff")
	}
	if def.Spec.GitOps.Commit {
		actions = append(actions, "commit local GitOps working tree changes")
	}
	if def.Spec.GitOps.Push {
		actions = append(actions, "push GitOps commit if explicitly confirmed")
	}
	if def.Spec.GitOps.Rollback {
		actions = append(actions, "checkout requested GitOps rollback revision")
	}
	if def.Spec.GitOps.StatusRead {
		actions = append(actions, "read Argo CD application status")
	}
	if def.Spec.GitOps.Sync {
		actions = append(actions, "request Argo CD sync if explicitly allowed")
	}
	if def.Spec.GitOps.Wait {
		actions = append(actions, "watch Argo CD sync and health status")
	}
	return DeploymentPlan{
		DeploymentRunID: plan.DeploymentRunID,
		TargetType:      def.Spec.Target.Type,
		TargetContext:   plan.RepoURL,
		Namespace:       def.Spec.Target.Namespace,
		Artifacts:       append([]string(nil), plan.ArtifactChanges...),
		DryRun:          !def.Spec.GitOps.WriteToWorkingTree && !def.Spec.GitOps.Sync,
		Apply:           def.Spec.GitOps.WriteToWorkingTree,
		Actions:         actions,
		Warnings:        append([]string(nil), plan.Warnings...),
		DiffSummary:     fmt.Sprintf("GitOps plan for %s in %s; remote Git push and sync remain guarded in Phase 6.1", plan.ApplicationName, plan.Path),
	}
}

func plannedImageChanges(def Definition) []string {
	changes := make([]string, 0, len(def.Spec.Artifacts))
	for _, artifact := range def.Spec.Artifacts {
		changes = append(changes, fmt.Sprintf("set %s image to %s", artifact.Name, artifact.Reference))
	}
	return changes
}

func (s *Service) applyGitOpsWorkingTree(ctx context.Context, plan GitOpsChangePlan, def Definition) (GitOpsChangePlan, GitOpsDiff, error) {
	if s.gitops == nil {
		return plan, GitOpsDiff{}, fmt.Errorf("gitops working tree adapter is not configured")
	}
	if def.Spec.GitOps.WorkingTree == "" {
		return plan, GitOpsDiff{}, fmt.Errorf("gitops.workingTree is required when writeToWorkingTree=true")
	}
	if len(plan.Files) == 0 {
		return plan, GitOpsDiff{}, fmt.Errorf("gitops plan has no files to update")
	}
	var changes []portgitops.FileChange
	for _, file := range plan.Files {
		before, err := s.gitops.ReadFile(ctx, def.Spec.GitOps.WorkingTree, file)
		if err != nil {
			return plan, GitOpsDiff{}, err
		}
		after := before
		for _, artifact := range def.Spec.Artifacts {
			if artifact.Target.ImageName == "" {
				continue
			}
			after = replaceContainerImage(after, artifact.Target.ImageName, deploymentArtifactReference(artifact))
		}
		diff, err := s.gitops.Diff(ctx, def.Spec.GitOps.WorkingTree, file, before, after)
		if err != nil {
			return plan, GitOpsDiff{}, err
		}
		change := portgitops.FileChange{Path: file, Before: before, After: after, Diff: diff, Changed: before != after, Operation: "update-image"}
		if before == after {
			change.Warning = "no matching image field changed"
		} else if err := s.gitops.WriteFile(ctx, def.Spec.GitOps.WorkingTree, file, after); err != nil {
			return plan, GitOpsDiff{}, err
		}
		changes = append(changes, change)
	}
	plan.FileChanges = changes
	plan.DryRun = false
	return plan, GitOpsDiff{Summary: fmt.Sprintf("generated local GitOps diff for %d file(s)", len(changes)), Files: changes}, nil
}

func (s *Service) commitGitOpsChanges(ctx context.Context, diff GitOpsDiff, def Definition) (portgitops.CommitResult, error) {
	if s.gitops == nil {
		return portgitops.CommitResult{}, fmt.Errorf("gitops working tree adapter is not configured")
	}
	if def.Spec.GitOps.WorkingTree == "" {
		return portgitops.CommitResult{}, fmt.Errorf("gitops.workingTree is required for gitops commit")
	}
	files := changedGitOpsFiles(diff)
	if len(files) == 0 {
		return portgitops.CommitResult{}, fmt.Errorf("gitops commit requires changed files")
	}
	message := strings.TrimSpace(def.Spec.GitOps.CommitMessage)
	if message == "" {
		message = fmt.Sprintf("chore: update %s release artifacts", def.Spec.Application)
	}
	return s.gitops.Commit(ctx, def.Spec.GitOps.WorkingTree, message, files)
}

func (s *Service) rollbackGitOpsRevision(ctx context.Context, def Definition, confirm bool) (portgitops.CommitResult, error) {
	if s.gitops == nil {
		return portgitops.CommitResult{}, fmt.Errorf("gitops working tree adapter is not configured")
	}
	if def.Spec.GitOps.WorkingTree == "" {
		return portgitops.CommitResult{}, fmt.Errorf("gitops.workingTree is required for gitops rollback")
	}
	return s.gitops.CheckoutRevision(ctx, def.Spec.GitOps.WorkingTree, def.Spec.GitOps.RollbackRevision, confirm)
}

func changedGitOpsFiles(diff GitOpsDiff) []string {
	files := make([]string, 0, len(diff.Files))
	for _, change := range diff.Files {
		if change.Changed {
			files = append(files, change.Path)
		}
	}
	return files
}

func deploymentArtifactReference(artifact Artifact) string {
	if artifact.Target.Substitute && artifact.Digest != "" && !containsDigest(artifact.Reference) {
		return artifact.Reference + "@" + artifact.Digest
	}
	return artifact.Reference
}

func replaceContainerImage(content string, containerName string, reference string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if !strings.Contains(line, "name: "+containerName) {
			continue
		}
		for j := i + 1; j < len(lines) && j <= i+8; j++ {
			trimmed := strings.TrimSpace(lines[j])
			if strings.HasPrefix(trimmed, "image: ") {
				prefix := lines[j][:strings.Index(lines[j], "image: ")]
				lines[j] = prefix + "image: " + reference
				return strings.Join(lines, "\n")
			}
		}
	}
	return content
}

func verifyManifestImages(specArtifacts []Artifact, artifactDetails []domainartifact.Inspection, images []ManifestImage) []string {
	if len(images) == 0 {
		return nil
	}
	expectedByContainer := make(map[string]domainartifact.Inspection)
	expectedByName := make(map[string]domainartifact.Inspection)
	for i, artifact := range specArtifacts {
		if i >= len(artifactDetails) {
			continue
		}
		detail := artifactDetails[i]
		if artifact.Target.ImageName != "" {
			expectedByContainer[artifact.Target.ImageName] = detail
		}
		expectedByName[detail.Reference.Name] = detail
	}
	var warnings []string
	for _, image := range images {
		inspection, err := domainartifact.InspectReference(image.Image, domainartifact.ArtifactTypeImage)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("manifest image %q could not be inspected: %v", image.Image, err))
			continue
		}
		for _, warning := range inspection.Warnings {
			warnings = append(warnings, fmt.Sprintf("manifest image %s: %s", image.Image, warning.Message))
		}
		expected, ok := expectedByContainer[image.Container]
		if !ok {
			expected, ok = expectedByName[inspection.Reference.Name]
		}
		if !ok {
			warnings = append(warnings, fmt.Sprintf("manifest image %s is not bound to a release artifact", image.Image))
			continue
		}
		if expected.Reference.Normalized != inspection.Reference.Normalized {
			warnings = append(warnings, fmt.Sprintf("manifest image %s differs from bound artifact %s", image.Image, expected.Reference.Normalized))
		}
	}
	return warnings
}

func containsDigest(reference string) bool {
	return strings.Contains(reference, "@sha256:")
}

func normalizeDefinition(def Definition) Definition {
	if def.Spec.Target.Type != "host" {
		return def
	}
	if def.Spec.Host.Strategy == "" {
		def.Spec.Host.Strategy = "symlink"
	}
	if def.Spec.Artifact.Reference != "" && len(def.Spec.Artifacts) == 0 {
		def.Spec.Artifacts = []Artifact{def.Spec.Artifact}
	}
	if len(def.Spec.Host.Hosts) == 0 {
		def.Spec.Host.Hosts = []Host{{
			ID:            "local-noop-host",
			Name:          def.Spec.Target.Name,
			EnvironmentID: def.Spec.Environment,
			CredentialRef: def.Spec.Host.CredentialRef,
		}}
	}
	for i := range def.Spec.Host.Hosts {
		if def.Spec.Host.Hosts[i].ID == "" {
			def.Spec.Host.Hosts[i].ID = def.Spec.Host.Hosts[i].Name
		}
		if def.Spec.Host.Hosts[i].EnvironmentID == "" {
			def.Spec.Host.Hosts[i].EnvironmentID = def.Spec.Environment
		}
		if def.Spec.Host.Hosts[i].CredentialRef == "" {
			def.Spec.Host.Hosts[i].CredentialRef = def.Spec.Host.CredentialRef
		}
	}
	return def
}

func hostCredentialRef(def Definition) string {
	if def.Spec.Host.CredentialRef != "" {
		return def.Spec.Host.CredentialRef
	}
	for _, host := range def.Spec.Host.Hosts {
		if host.CredentialRef != "" {
			return host.CredentialRef
		}
	}
	return def.Spec.Target.CredentialsRef
}

func normalizeHostHealthChecks(spec HostSpec) []HostHealthCheck {
	checks := make([]HostHealthCheck, 0, len(spec.HealthChecks)+1)
	for _, check := range spec.HealthChecks {
		if check.TimeoutSeconds == 0 {
			check.TimeoutSeconds = 30
		}
		checks = append(checks, check)
	}
	if len(checks) == 0 && spec.HealthCheck != "" {
		checkType := "command"
		target := spec.HealthCheck
		if strings.HasPrefix(spec.HealthCheck, "http://") || strings.HasPrefix(spec.HealthCheck, "https://") {
			checkType = "http"
		} else if strings.Contains(spec.HealthCheck, ":") && !strings.Contains(spec.HealthCheck, " ") {
			checkType = "tcp"
		}
		check := HostHealthCheck{Type: checkType, TimeoutSeconds: 30}
		if checkType == "command" {
			check.Command = target
		} else {
			check.Target = target
		}
		checks = append(checks, check)
	}
	return checks
}

func stableHostHash(host Host, artifact string, deployPath string) string {
	sum := sha256.Sum256([]byte(host.ID + "\n" + host.Name + "\n" + host.Address + "\n" + artifact + "\n" + deployPath))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func (s *Service) rollbackBaseline(runID string, def Definition, resources []ManifestResourceSummary) *domaindeployment.RollbackRecord {
	refs := make([]string, 0, len(resources))
	for _, resource := range resources {
		refs = append(refs, fmt.Sprintf("%s/%s/%s", resource.Kind, resource.Namespace, resource.Name))
	}
	now := s.now()
	return &domaindeployment.RollbackRecord{
		ID:                  newID("rollback"),
		DeploymentRunID:     runID,
		Strategy:            "manifest-snapshot",
		Status:              "planned",
		TargetType:          def.Spec.Target.Type,
		TargetName:          def.Spec.Target.Name,
		ManifestSnapshotRef: "memory://" + runID + "/previous-manifests",
		ResourceRefs:        refs,
		Reason:              "manifest restore rollback requires explicit confirmation",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func (s *Service) hostRollbackBaseline(runID string, def Definition, plan HostDeploymentPlan) *domaindeployment.RollbackRecord {
	refs := make([]string, 0, len(plan.Hosts))
	for _, host := range plan.Hosts {
		refs = append(refs, fmt.Sprintf("Host/%s/%s", def.Spec.Environment, host.HostName))
	}
	now := s.now()
	return &domaindeployment.RollbackRecord{
		ID:                  newID("rollback"),
		DeploymentRunID:     runID,
		Strategy:            "symlink-restore",
		Status:              "placeholder",
		TargetType:          def.Spec.Target.Type,
		TargetName:          def.Spec.Target.Name,
		ManifestSnapshotRef: "memory://" + runID + "/host-release-plan",
		ResourceRefs:        refs,
		Reason:              "host rollback uses guarded symlink restore and optional service restart",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func (s *Service) logChunk(runID string, stream string, content string, sequence int64) event.LogChunk {
	return event.LogChunk{
		ID:              newID("log"),
		DeploymentRunID: runID,
		Stream:          stream,
		Sequence:        sequence,
		Content:         content,
		CreatedAt:       s.now(),
	}
}

func validateManifestRequest(request ManifestRequest) error {
	if request.Plan.DeploymentRunID == "" {
		return fmt.Errorf("deploymentRunId is required")
	}
	if len(request.Documents) == 0 {
		return fmt.Errorf("at least one manifest document is required")
	}
	return nil
}

func (s *Service) recordEventAndAudit(ctx context.Context, runID string, eventType string, action string, actorID string, status string, message string) error {
	if err := s.recordEvent(ctx, runID, eventType, status, message); err != nil {
		return err
	}
	return s.store.AppendAudit(ctx, runID, audit.AuditLog{
		ID:        newID("audit"),
		ActorID:   actorID,
		Action:    action,
		Subject:   runID,
		CreatedAt: s.now(),
	})
}

func (s *Service) recordEvent(ctx context.Context, runID string, eventType string, status string, message string) error {
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              newID("evt"),
		Type:            eventType,
		Source:          "nivora/deployment",
		Subject:         runID,
		Time:            s.now(),
		DataContentType: "application/json",
		Data: map[string]any{
			"status":  status,
			"message": message,
		},
	}
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return err
	}
	return s.store.AppendEvent(ctx, runID, evt)
}

func (s *Service) recordRuntimeEventOnly(ctx context.Context, runID string, eventType string, status string, message string) error {
	return s.recordEvent(ctx, runID, eventType, status, message)
}

func (s *Service) recordRuntimeEvent(ctx context.Context, record RunRecord, eventType string, status string, message string) (RunRecord, error) {
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              newID("evt"),
		Type:            eventType,
		Source:          "nivora/deployment",
		Subject:         record.Run.ID,
		Time:            s.now(),
		DataContentType: "application/json",
		Data: map[string]any{
			"status":  status,
			"message": message,
		},
	}
	if err := s.eventBus.Publish(ctx, evt); err != nil {
		return RunRecord{}, err
	}
	record.Events = append(record.Events, evt)
	if err := s.store.Save(ctx, record); err != nil {
		return RunRecord{}, err
	}
	return record, nil
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
