package deployment

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	portgitops "github.com/sevoniva/nivora/internal/ports/gitops"
	"github.com/sevoniva/nivora/internal/ports/policy"
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
	EventDeploymentDiffGenerated       = "devops.deployment.diff.generated"
	EventDeploymentResourceObserved    = "devops.deployment.resource.observed"
	EventDeploymentResourceDegraded    = "devops.deployment.resource.degraded"
	EventGitOpsPlanCreated             = "devops.gitops.plan.created"
	EventGitOpsDiffGenerated           = "devops.gitops.diff.generated"
	EventGitOpsWorkingTreeUpdated      = "devops.gitops.workingtree.updated"
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
)

type KubernetesManifestClient interface {
	ServerDryRun(ctx context.Context, request ManifestRequest) (KubernetesDryRunResult, error)
	Apply(ctx context.Context, request ManifestRequest) (KubernetesApplyResult, error)
	WatchRollout(ctx context.Context, request ManifestRequest) (RolloutResult, error)
}

type ManifestClient = KubernetesManifestClient

type Service struct {
	store    Store
	renderer ManifestRenderer
	client   ManifestClient
	policy   policy.Engine
	eventBus eventbus.EventBus
	gitops   portgitops.WorkingTree
	argocd   portargocd.Provider
	now      func() time.Time
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

func (s *Service) CreateAndRun(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}
	if input.Definition.Spec.Options.Apply && !input.AllowApply {
		return CreateRunResult{}, fmt.Errorf("deployment apply requires explicit confirmation")
	}
	record := s.newRecord(input.Definition)
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
	record, err := s.process(ctx, record, input.ActorID)
	if err != nil {
		return CreateRunResult{}, err
	}
	return CreateRunResult{Record: record}, nil
}

func (s *Service) Plan(ctx context.Context, input CreateRunInput) (CreateRunResult, error) {
	if err := input.Definition.Validate(); err != nil {
		return CreateRunResult{}, err
	}
	record := s.newRecord(input.Definition)
	if input.Definition.Spec.Target.Type == "argocd" {
		record.GitOpsPlan = s.buildGitOpsPlan(record.Run.ID, input.Definition, nil)
		record.Plan = deploymentPlanFromGitOps(record.GitOpsPlan, input.Definition)
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
	record = s.attachResourceObservability(record, input.Definition, documents)
	record.Logs = append(record.Logs, s.logChunk(record.Run.ID, "system", fmt.Sprintf("rendered %d manifest document(s)", len(documents)), int64(len(record.Logs)+1)))
	return CreateRunResult{Record: record}, nil
}

func (s *Service) process(ctx context.Context, record RunRecord, actorID string) (RunRecord, error) {
	if record.Definition.Spec.Target.Type == "argocd" {
		return s.processGitOps(ctx, record, actorID, false, false)
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
		warnings = append(warnings, "apply requested; Phase 2.4 apply requires explicit confirmation and uses the configured manifest client")
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
		Executable:        false,
		Warnings:          []string{"rollback execution is not implemented in Phase 2.4; this plan is non-destructive"},
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
	if def.Spec.GitOps.Sync {
		warnings = append(warnings, "Argo CD sync requested; sync is disabled unless explicitly allowed and confirmed")
	}
	if def.Spec.GitOps.Prune {
		warnings = append(warnings, "Argo CD prune requested; prune is guarded and defaults to false")
	}
	if def.Spec.GitOps.Force {
		warnings = append(warnings, "Argo CD force sync is not supported in Phase 2.6")
	}
	return GitOpsChangePlan{
		DeploymentRunID:       runID,
		ApplicationName:       def.Spec.Target.ApplicationName,
		RepoURL:               def.Spec.Target.RepoURL,
		Path:                  def.Spec.Target.Path,
		Revision:              def.Spec.Target.Revision,
		Files:                 files,
		FileChanges:           changes,
		ArtifactChanges:       artifacts,
		ManifestValueChanges:  plannedImageChanges(def),
		CommitMessageProposal: fmt.Sprintf("chore: update %s release artifacts", def.Spec.Application),
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
		TargetContext:   def.Spec.Target.RepoURL,
		Namespace:       def.Spec.Target.Namespace,
		Artifacts:       append([]string(nil), plan.ArtifactChanges...),
		DryRun:          !def.Spec.GitOps.WriteToWorkingTree && !def.Spec.GitOps.Sync,
		Apply:           def.Spec.GitOps.WriteToWorkingTree,
		Actions:         actions,
		Warnings:        append([]string(nil), plan.Warnings...),
		DiffSummary:     fmt.Sprintf("GitOps plan for %s in %s; remote Git diff is not available in Phase 2.3", plan.ApplicationName, plan.Path),
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
			after = replaceContainerImage(after, artifact.Target.ImageName, artifact.Reference)
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
		Status:              "placeholder",
		TargetType:          def.Spec.Target.Type,
		TargetName:          def.Spec.Target.Name,
		ManifestSnapshotRef: "memory://" + runID + "/previous-manifests",
		ResourceRefs:        refs,
		Reason:              "rollback execution is not implemented in Phase 2.4",
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
