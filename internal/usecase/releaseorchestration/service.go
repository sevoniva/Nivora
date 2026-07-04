package releaseorchestration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	domainevent "github.com/sevoniva/nivora/internal/domain/event"
	domainpolicy "github.com/sevoniva/nivora/internal/domain/policy"
	domainrelease "github.com/sevoniva/nivora/internal/domain/release"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	"github.com/sevoniva/nivora/internal/ports/policy"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

const (
	EventReleasePlanCreated                 = "devops.release.plan.created"
	EventReleaseExecutionCreated            = "devops.release.execution.created"
	EventReleaseExecutionStarted            = "devops.release.execution.started"
	EventReleaseExecutionTargetStarted      = "devops.release.execution.target.started"
	EventReleaseExecutionTargetCompleted    = "devops.release.execution.target.completed"
	EventReleaseExecutionTargetFailed       = "devops.release.execution.target.failed"
	EventReleaseExecutionSucceeded          = "devops.release.execution.succeeded"
	EventReleaseExecutionPartiallySucceeded = "devops.release.execution.partially_succeeded"
	EventReleaseExecutionFailed             = "devops.release.execution.failed"
	EventReleaseExecutionCanceled           = "devops.release.execution.canceled"
)

type ArtifactService interface {
	CreateRelease(ctx context.Context, input artifactusecase.CreateReleaseInput) (artifactusecase.ReleaseRecord, error)
	GetRelease(ctx context.Context, id string) (artifactusecase.ReleaseRecord, error)
	UpdateReleaseStatus(ctx context.Context, id string, status domainrelease.ReleaseStatus, actorID string, reason string) (artifactusecase.ReleaseRecord, error)
}

type DeploymentService interface {
	Plan(ctx context.Context, input deploymentusecase.CreateRunInput) (deploymentusecase.CreateRunResult, error)
	CreateAndRun(ctx context.Context, input deploymentusecase.CreateRunInput) (deploymentusecase.CreateRunResult, error)
	Cancel(ctx context.Context, id string, actorID string) (deploymentusecase.RunRecord, error)
	Timeline(ctx context.Context, id string) ([]deploymentusecase.TimelineEntry, error)
}

type Service struct {
	store          Store
	artifacts      ArtifactService
	deployments    DeploymentService
	policy         policy.Engine
	security       *securityusecase.Service
	policies       SecurityPolicyCatalog
	governance     Governance
	releaseTargets ReleaseTargetCatalog
	eventBus       eventbus.EventBus
	now            func() time.Time
}

type Governance interface {
	RequestApproval(ctx context.Context, subjectType string, subjectID string, environmentID string, requestedBy string, reason string) (domainapproval.ApprovalRequest, error)
}

type SecurityPolicyCatalog interface {
	ResolveEnabledForScope(ctx context.Context, input policyusecase.ResolveInput) (domainpolicy.Policy, bool, error)
}

type ReleaseTargetCatalog interface {
	GetReleaseTarget(ctx context.Context, id string) (environment.ReleaseTarget, error)
}

func (s *Service) WithSecurity(securityService *securityusecase.Service) *Service {
	s.security = securityService
	return s
}

func (s *Service) WithPolicyCatalog(catalog SecurityPolicyCatalog) *Service {
	s.policies = catalog
	return s
}

func (s *Service) WithGovernance(governance Governance) *Service {
	s.governance = governance
	return s
}

func (s *Service) WithReleaseTargetCatalog(catalog ReleaseTargetCatalog) *Service {
	s.releaseTargets = catalog
	return s
}

func NewService(store Store, artifacts ArtifactService, deployments DeploymentService, policyEngine policy.Engine, bus eventbus.EventBus) *Service {
	return &Service{
		store:       store,
		artifacts:   artifacts,
		deployments: deployments,
		policy:      policyEngine,
		eventBus:    bus,
		now:         time.Now,
	}
}

func (s *Service) Plan(ctx context.Context, input PlanInput) (PlanRecord, error) {
	if err := input.Definition.Validate(); err != nil {
		return PlanRecord{}, err
	}
	releaseRecord, err := s.resolveRelease(ctx, input.Definition, input.ActorID)
	if err != nil {
		return PlanRecord{}, err
	}
	plan, err := s.buildPlan(ctx, input.Definition, releaseRecord, input.ProjectID)
	if err != nil {
		return PlanRecord{}, err
	}
	record := PlanRecord{Definition: input.Definition, Release: releaseRecord.Release, Plan: plan}
	if s.security != nil {
		scanInput, err := s.securityScanInput(ctx, input.Definition, releaseRecord.Release.ID, input.ProjectID, input.ActorID, plan)
		if err != nil {
			return PlanRecord{}, err
		}
		securityRecord, err := s.security.Scan(ctx, scanInput)
		if err != nil {
			return PlanRecord{}, err
		}
		record.Security = securityRecord
		if securityRecord.Policy.Decision == domainsecurity.GateDeny || securityRecord.Policy.Decision == domainsecurity.GateRequireApproval {
			record.Plan.Warnings = append(record.Plan.Warnings, securityRecord.Policy.Reason)
		}
	}
	if err := s.store.SavePlan(ctx, record); err != nil {
		return PlanRecord{}, err
	}
	if record, err = s.setPlanReleaseStatus(ctx, record, domainrelease.ReleaseStatusPlanning, input.ActorID, "release plan created"); err != nil {
		return PlanRecord{}, err
	}
	return s.recordPlanEventAndAudit(ctx, record, EventReleasePlanCreated, "Release plan created", input.ActorID, "Release plan created")
}

func (s *Service) Deploy(ctx context.Context, input DeployInput) (ExecutionRecord, error) {
	planRecord, err := s.Plan(ctx, PlanInput{Definition: input.Definition, ProjectID: input.ProjectID, ActorID: input.ActorID, CorrelationID: input.CorrelationID})
	if err != nil {
		return ExecutionRecord{}, err
	}
	if planRecord.Security.Policy.Decision == "deny" {
		now := s.now()
		record := ExecutionRecord{
			Definition: input.Definition,
			Release:    planRecord.Release,
			Plan:       planRecord.Plan,
			Security:   planRecord.Security,
			Execution: ReleaseExecution{
				ID:              newID("rexec"),
				ReleaseID:       planRecord.Plan.ReleaseID,
				CorrelationID:   input.CorrelationID,
				EnvironmentID:   planRecord.Plan.EnvironmentID,
				EnvironmentName: planRecord.Plan.EnvironmentName,
				Status:          ExecutionFailed,
				Reason:          planRecord.Security.Policy.Reason,
				CreatedAt:       now,
				UpdatedAt:       now,
				FinishedAt:      &now,
			},
		}
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusFailed, input.ActorID, planRecord.Security.Policy.Reason); err != nil {
			return ExecutionRecord{}, err
		}
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionFailed, "Release execution failed", input.ActorID, planRecord.Security.Policy.Reason)
	}
	now := s.now()
	exec := ReleaseExecution{
		ID:              newID("rexec"),
		ReleaseID:       planRecord.Plan.ReleaseID,
		CorrelationID:   input.CorrelationID,
		EnvironmentID:   planRecord.Plan.EnvironmentID,
		EnvironmentName: planRecord.Plan.EnvironmentName,
		Status:          ExecutionCreated,
		Targets:         initialTargetExecutions(planRecord.Plan.Targets),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	record := ExecutionRecord{
		Definition: input.Definition,
		Release:    planRecord.Release,
		Plan:       planRecord.Plan,
		Execution:  exec,
		Security:   planRecord.Security,
	}
	if err := s.store.SaveExecution(ctx, record); err != nil {
		return ExecutionRecord{}, err
	}
	if record, err = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionCreated, "Release execution created", input.ActorID, "Release execution created"); err != nil {
		return ExecutionRecord{}, err
	}
	if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusDeploying, input.ActorID, "release execution created"); err != nil {
		return ExecutionRecord{}, err
	}
	if deniedTarget := firstDeniedTarget(record.Plan.PolicyResults); deniedTarget != "" {
		record.Execution.Status = ExecutionFailed
		record.Execution.Reason = "release policy denied target " + deniedTarget
		finished := s.now()
		record.Execution.FinishedAt = &finished
		record.Execution.UpdatedAt = finished
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusFailed, input.ActorID, record.Execution.Reason); err != nil {
			return ExecutionRecord{}, err
		}
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionFailed, "Release execution failed", input.ActorID, record.Execution.Reason)
	}
	if record.Plan.Strategy == StrategyPlanOnly {
		record.Execution.Status = ExecutionSucceeded
		finished := s.now()
		record.Execution.FinishedAt = &finished
		record.Execution.UpdatedAt = finished
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusSucceeded, input.ActorID, "plan-only release execution succeeded"); err != nil {
			return ExecutionRecord{}, err
		}
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionSucceeded, "Release execution succeeded", input.ActorID, "plan-only release execution succeeded")
	}
	if input.Definition.Spec.ApprovalRequired || planRecord.Security.Policy.Decision == domainsecurity.GateRequireApproval {
		approvalReason := "release approval required"
		if planRecord.Security.Policy.Decision == domainsecurity.GateRequireApproval && planRecord.Security.Policy.Reason != "" {
			approvalReason = planRecord.Security.Policy.Reason
		}
		if s.governance != nil {
			approval, err := s.governance.RequestApproval(ctx, domainapproval.SubjectRelease, record.Execution.ID, record.Execution.EnvironmentID, input.ActorID, approvalReason)
			if err != nil {
				return ExecutionRecord{}, err
			}
			record.Approval = approval
		}
		record.Execution.Status = ExecutionWaitingApproval
		record.Execution.Reason = approvalReason
		record.Execution.UpdatedAt = s.now()
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusWaitingApproval, input.ActorID, record.Execution.Reason); err != nil {
			return ExecutionRecord{}, err
		}
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionCreated, "Release execution waiting for approval", input.ActorID, record.Execution.Reason)
	}
	return s.runSequential(ctx, record, input.ActorID)
}

func (s *Service) GetPlan(ctx context.Context, releaseID string) (PlanRecord, error) {
	return s.store.GetLatestPlanForRelease(ctx, releaseID)
}

func (s *Service) ListExecutions(ctx context.Context, releaseID string) ([]ExecutionRecord, error) {
	return s.store.ListExecutions(ctx, releaseID)
}

func (s *Service) GetExecution(ctx context.Context, id string) (ExecutionRecord, error) {
	return s.store.GetExecution(ctx, id)
}

func (s *Service) Targets(ctx context.Context, id string) ([]TargetExecution, error) {
	record, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]TargetExecution(nil), record.Execution.Targets...), nil
}

func (s *Service) ApplyApprovalDecision(ctx context.Context, id string, approval domainapproval.ApprovalRequest, actorID string) (ExecutionRecord, error) {
	record, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return ExecutionRecord{}, err
	}
	if record.Execution.Status != ExecutionWaitingApproval {
		return ExecutionRecord{}, fmt.Errorf("release execution is not waiting for approval")
	}
	if approval.SubjectID != "" && approval.SubjectID != id {
		return ExecutionRecord{}, fmt.Errorf("approval subject does not match release execution")
	}
	record.Approval = approval
	switch approval.Status {
	case domainapproval.StatusApproved:
		record.Execution.Reason = "approval approved"
		record.Execution.UpdatedAt = s.now()
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionStarted, "Release approval approved", actorID, "Release approval approved; resuming execution"); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusDeploying, actorID, "approval approved; resuming release execution"); err != nil {
			return ExecutionRecord{}, err
		}
		return s.runSequential(ctx, record, actorID)
	case domainapproval.StatusRejected, domainapproval.StatusExpired:
		record.Execution.Status = ExecutionFailed
		record.Execution.Reason = "approval " + strings.ToLower(approval.Status)
		finished := s.now()
		record.Execution.FinishedAt = &finished
		record.Execution.UpdatedAt = finished
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusFailed, actorID, record.Execution.Reason); err != nil {
			return ExecutionRecord{}, err
		}
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionFailed, "Release execution failed", actorID, record.Execution.Reason)
	case domainapproval.StatusCanceled:
		record.Execution.Status = ExecutionCanceled
		record.Execution.Reason = "approval canceled"
		finished := s.now()
		record.Execution.FinishedAt = &finished
		record.Execution.UpdatedAt = finished
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusCanceled, actorID, record.Execution.Reason); err != nil {
			return ExecutionRecord{}, err
		}
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionCanceled, "Release execution canceled", actorID, record.Execution.Reason)
	default:
		return ExecutionRecord{}, fmt.Errorf("approval must be Approved, Rejected, Expired, or Canceled")
	}
}

func (s *Service) Timeline(ctx context.Context, id string) ([]TimelineEntry, error) {
	record, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	timeline := make([]TimelineEntry, 0, len(record.Events))
	for _, evt := range record.Events {
		entry := TimelineEntry{Type: evt.Type, Time: evt.Time, Subject: evt.Subject}
		if status, ok := evt.Data["status"].(string); ok {
			entry.Status = status
		}
		if message, ok := evt.Data["message"].(string); ok {
			entry.Message = message
		}
		timeline = append(timeline, entry)
	}
	for _, deployment := range record.Deployments {
		entries, err := s.deployments.Timeline(ctx, deployment.Run.ID)
		if err != nil {
			continue
		}
		for _, item := range entries {
			timeline = append(timeline, TimelineEntry{
				Type:    item.Type,
				Time:    item.Time,
				Subject: item.Subject,
				Status:  item.Status,
				Message: item.Message,
			})
		}
	}
	return timeline, nil
}

func (s *Service) Cancel(ctx context.Context, id string, actorID string) (ExecutionRecord, error) {
	record, err := s.store.GetExecution(ctx, id)
	if err != nil {
		return ExecutionRecord{}, err
	}
	if isTerminal(record.Execution.Status) {
		return record, ErrExecutionTerminal
	}
	return s.cancelRecord(ctx, record, actorID)
}

func (s *Service) CancelExecutionsForRelease(ctx context.Context, releaseID string, actorID string) ([]ExecutionRecord, error) {
	records, err := s.store.ListExecutions(ctx, strings.TrimSpace(releaseID))
	if err != nil {
		return nil, err
	}
	canceled := make([]ExecutionRecord, 0, len(records))
	for _, record := range records {
		if isTerminal(record.Execution.Status) {
			continue
		}
		updated, err := s.cancelRecord(ctx, record, actorID)
		if err != nil {
			return nil, err
		}
		canceled = append(canceled, updated)
	}
	return canceled, nil
}

func (s *Service) cancelRecord(ctx context.Context, record ExecutionRecord, actorID string) (ExecutionRecord, error) {
	if s.deployments != nil {
		record = s.cancelDeploymentRuns(ctx, record, actorID)
	}
	record.Execution.Status = ExecutionCanceled
	record.Execution.Reason = "canceled by request"
	now := s.now()
	record.Execution.FinishedAt = &now
	record.Execution.UpdatedAt = now
	for i := range record.Execution.Targets {
		if !isTerminal(record.Execution.Targets[i].Status) {
			record.Execution.Targets[i].Status = ExecutionCanceled
		}
	}
	if err := s.store.SaveExecution(ctx, record); err != nil {
		return ExecutionRecord{}, err
	}
	var err error
	if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusCanceled, actorID, "release execution canceled"); err != nil {
		return ExecutionRecord{}, err
	}
	return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionCanceled, "Release execution canceled", actorID, "Release execution canceled")
}

func (s *Service) cancelDeploymentRuns(ctx context.Context, record ExecutionRecord, actorID string) ExecutionRecord {
	terminal := map[string]bool{}
	for _, deployment := range record.Deployments {
		if deployment.Run.ID != "" && isTerminal(mapDeploymentStatus(deployment.Run.Status)) {
			terminal[deployment.Run.ID] = true
		}
	}
	for _, runID := range record.Execution.DeploymentRunIDs {
		runID = strings.TrimSpace(runID)
		if runID == "" || terminal[runID] {
			continue
		}
		canceled, err := s.deployments.Cancel(ctx, runID, actorID)
		if err != nil {
			if errors.Is(err, deploymentusecase.ErrRunTerminal) {
				continue
			}
			record = appendTargetWarningForDeploymentRun(record, runID, "deployment cancel failed: "+err.Error())
			continue
		}
		record = replaceDeploymentRecord(record, canceled)
		record = markTargetStatusForDeploymentRun(record, runID, ExecutionCanceled)
	}
	return record
}

func (s *Service) runSequential(ctx context.Context, record ExecutionRecord, actorID string) (ExecutionRecord, error) {
	started := s.now()
	record.Execution.Status = ExecutionRunning
	record.Execution.StartedAt = &started
	record.Execution.UpdatedAt = started
	if err := s.store.SaveExecution(ctx, record); err != nil {
		return ExecutionRecord{}, err
	}
	var err error
	if record, err = s.setExecutionReleaseStatus(ctx, record, domainrelease.ReleaseStatusDeploying, actorID, "sequential release execution started"); err != nil {
		return ExecutionRecord{}, err
	}
	if record, err = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionStarted, "Release execution started", actorID, "Sequential release execution started"); err != nil {
		return ExecutionRecord{}, err
	}
	successes := 0
	failures := 0
	targets := orderedTargets(record.Definition.Spec.Targets)
	for _, target := range targets {
		if target.Type == "noop" || target.Type == "webhook" {
			record = s.markTargetStarted(record, target.Name)
			if err := s.store.SaveExecution(ctx, record); err != nil {
				return ExecutionRecord{}, err
			}
			record, _ = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionTargetStarted, "Target execution started", actorID, target.Name)
			record = s.markTargetCompleted(record, target.Name, "", ExecutionSucceeded, nil)
			successes++
			if err := s.store.SaveExecution(ctx, record); err != nil {
				return ExecutionRecord{}, err
			}
			record, _ = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionTargetCompleted, "Target execution completed", actorID, target.Name+" completed")
			continue
		}
		record = s.markTargetStarted(record, target.Name)
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		record, _ = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionTargetStarted, "Target execution started", actorID, target.Name)
		result, runErr := s.deployments.CreateAndRun(ctx, deploymentusecase.CreateRunInput{Definition: target.Deployment, ProjectID: firstTargetProjectID(record, target.Name), CorrelationID: record.Execution.CorrelationID})
		status := mapDeploymentStatus(result.Record.Run.Status)
		if runErr != nil {
			status = ExecutionFailed
			record = s.markTargetCompleted(record, target.Name, "", status, []string{runErr.Error()})
			failures++
			record, _ = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionTargetFailed, "Target execution failed", actorID, runErr.Error())
		} else {
			record.Deployments = append(record.Deployments, result.Record)
			record.Execution.DeploymentRunIDs = append(record.Execution.DeploymentRunIDs, result.Record.Run.ID)
			if status == ExecutionSucceeded {
				successes++
				record, _ = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionTargetCompleted, "Target execution completed", actorID, target.Name+" completed")
			} else {
				failures++
				record, _ = s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionTargetFailed, "Target execution failed", actorID, result.Record.Run.Reason)
			}
			record = s.markTargetCompleted(record, target.Name, result.Record.Run.ID, status, result.Record.Plan.Warnings)
		}
		if err := s.store.SaveExecution(ctx, record); err != nil {
			return ExecutionRecord{}, err
		}
		if failures > 0 && !record.Definition.Spec.ContinueOnFailure {
			break
		}
	}
	record.Execution.Status = aggregateStatus(successes, failures, len(targets), record.Definition.Spec.ContinueOnFailure)
	if failures > 0 {
		record.Execution.Reason = "one or more release targets failed"
	}
	finished := s.now()
	record.Execution.FinishedAt = &finished
	record.Execution.UpdatedAt = finished
	if err := s.store.SaveExecution(ctx, record); err != nil {
		return ExecutionRecord{}, err
	}
	eventType := EventReleaseExecutionSucceeded
	action := "Release execution completed"
	message := "Release execution succeeded"
	releaseStatus := domainrelease.ReleaseStatusSucceeded
	if record.Execution.Status == ExecutionPartiallySucceeded {
		eventType = EventReleaseExecutionPartiallySucceeded
		message = "Release execution partially succeeded"
		releaseStatus = domainrelease.ReleaseStatusFailed
	} else if record.Execution.Status == ExecutionFailed {
		eventType = EventReleaseExecutionFailed
		action = "Release execution failed"
		message = record.Execution.Reason
		releaseStatus = domainrelease.ReleaseStatusFailed
	}
	if record, err = s.setExecutionReleaseStatus(ctx, record, releaseStatus, actorID, message); err != nil {
		return ExecutionRecord{}, err
	}
	return s.recordExecutionEventAndAudit(ctx, record, eventType, action, actorID, message)
}

func (s *Service) resolveRelease(ctx context.Context, def Definition, actorID string) (artifactusecase.ReleaseRecord, error) {
	if def.Spec.ReleaseID != "" {
		return s.artifacts.GetRelease(ctx, def.Spec.ReleaseID)
	}
	return s.artifacts.CreateRelease(ctx, artifactusecase.CreateReleaseInput{Definition: def.Spec.Release, ActorID: actorID})
}

func (s *Service) buildPlan(ctx context.Context, def Definition, releaseRecord artifactusecase.ReleaseRecord, projectID string) (ReleasePlan, error) {
	envID := newID("env")
	projectID = strings.TrimSpace(projectID)
	plan := ReleasePlan{
		ID:              newID("rplan"),
		ReleaseID:       releaseRecord.Release.ID,
		EnvironmentID:   envID,
		EnvironmentName: def.Spec.Environment,
		Strategy:        def.Spec.Strategy,
		Concurrency:     def.Spec.Concurrency,
		CreatedAt:       s.now(),
	}
	if plan.Strategy == "" {
		plan.Strategy = StrategySequential
	}
	if plan.Strategy == StrategyParallel {
		plan.Warnings = append(plan.Warnings, "parallel execution is planned as future work; Phase 2.7 uses sequential execution")
	}
	for _, binding := range releaseRecord.Bindings {
		ref := binding.Reference
		if binding.DigestReference != "" {
			ref = binding.DigestReference
		}
		plan.ArtifactSummary = append(plan.ArtifactSummary, ref)
	}
	environmentResolvedFromCatalog := false
	for _, target := range orderedTargets(def.Spec.Targets) {
		releaseTarget, resolvedTarget, err := s.resolveReleaseTarget(ctx, target, projectID, plan.EnvironmentID, plan.CreatedAt)
		if err != nil {
			return ReleasePlan{}, err
		}
		if target.TargetID != "" && releaseTarget.EnvironmentID != "" {
			if !environmentResolvedFromCatalog {
				previousEnvironmentID := plan.EnvironmentID
				plan.EnvironmentID = releaseTarget.EnvironmentID
				environmentResolvedFromCatalog = true
				for i := range plan.Targets {
					if plan.Targets[i].EnvironmentID == previousEnvironmentID {
						plan.Targets[i].EnvironmentID = plan.EnvironmentID
					}
				}
			} else if plan.EnvironmentID != releaseTarget.EnvironmentID {
				return ReleasePlan{}, fmt.Errorf("release catalog targets must use one environment: %q and %q", plan.EnvironmentID, releaseTarget.EnvironmentID)
			}
		}
		if projectID != "" && releaseTarget.ProjectID != "" && releaseTarget.ProjectID != projectID {
			return ReleasePlan{}, fmt.Errorf("release target %q is outside project %q", releaseTarget.ID, projectID)
		}
		if releaseTarget.ProjectID == "" {
			releaseTarget.ProjectID = projectID
		}
		plan.Targets = append(plan.Targets, releaseTarget)
		plan.Ordering = append(plan.Ordering, resolvedTarget.Name)
		policyResult, err := s.policy.Evaluate(ctx, policy.Request{
			Subject: releaseRecord.Release.ID,
			Action:  "release.plan",
			Context: map[string]any{"target": resolvedTarget.Name, "targetType": resolvedTarget.Type},
		})
		if err != nil {
			return ReleasePlan{}, err
		}
		plan.PolicyResults = append(plan.PolicyResults, PolicyResult{Target: resolvedTarget.Name, Allowed: policyResult.Allowed, Reasons: policyResult.Reasons})
		if !policyResult.Allowed {
			plan.Warnings = append(plan.Warnings, "release policy denied target "+resolvedTarget.Name)
			continue
		}
		if resolvedTarget.Type == "noop" || resolvedTarget.Type == "webhook" {
			plan.DeploymentPlans = append(plan.DeploymentPlans, deploymentusecase.DeploymentPlan{
				DeploymentRunID: newID("drun"),
				TargetType:      resolvedTarget.Type,
				Actions:         []string{"record target placeholder"},
				Warnings:        []string{"target execution is a Phase 2.7 placeholder"},
				DiffSummary:     "placeholder target has no live diff",
			})
			continue
		}
		if resolvedTarget.Deployment.Kind == "" {
			return ReleasePlan{}, fmt.Errorf("release catalog target %q type %q requires an inline deployment spec", resolvedTarget.Name, resolvedTarget.Type)
		}
		fillDeploymentTargetFromCatalog(&resolvedTarget.Deployment, releaseTarget)
		if err := resolvedTarget.Deployment.Validate(); err != nil {
			return ReleasePlan{}, fmt.Errorf("release orchestration target %q deployment invalid: %w", resolvedTarget.Name, err)
		}
		result, err := s.deployments.Plan(ctx, deploymentusecase.CreateRunInput{Definition: resolvedTarget.Deployment, ProjectID: releaseTarget.ProjectID})
		if err != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("target %s plan failed: %v", resolvedTarget.Name, err))
			continue
		}
		plan.DeploymentPlans = append(plan.DeploymentPlans, result.Record.Plan)
	}
	return plan, nil
}

func (s *Service) resolveReleaseTarget(ctx context.Context, target TargetSpec, projectID string, environmentID string, now time.Time) (environment.ReleaseTarget, TargetSpec, error) {
	if strings.TrimSpace(target.TargetID) != "" {
		if s.releaseTargets == nil {
			return environment.ReleaseTarget{}, TargetSpec{}, fmt.Errorf("release target catalog is not configured")
		}
		catalogTarget, err := s.releaseTargets.GetReleaseTarget(ctx, strings.TrimSpace(target.TargetID))
		if err != nil {
			return environment.ReleaseTarget{}, TargetSpec{}, err
		}
		if !catalogTarget.Enabled {
			return environment.ReleaseTarget{}, TargetSpec{}, fmt.Errorf("release target %q is disabled", catalogTarget.ID)
		}
		if projectID != "" && catalogTarget.ProjectID != "" && catalogTarget.ProjectID != projectID {
			return environment.ReleaseTarget{}, TargetSpec{}, fmt.Errorf("release target %q is outside project %q", catalogTarget.ID, projectID)
		}
		resolved := target
		if resolved.Name == "" {
			resolved.Name = catalogTarget.Name
		}
		if resolved.Type == "" {
			resolved.Type = catalogTarget.TargetType
		}
		catalogTarget.Labels = mergeStringMaps(catalogTarget.Labels, target.Labels)
		if catalogTarget.ProjectID == "" {
			catalogTarget.ProjectID = projectID
		}
		if catalogTarget.EnvironmentID == "" {
			catalogTarget.EnvironmentID = environmentID
		}
		return catalogTarget, resolved, nil
	}
	releaseTarget := environment.ReleaseTarget{
		ID:            newID("target"),
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		Name:          target.Name,
		TargetType:    target.Type,
		Context:       target.Deployment.Spec.Target.Context,
		Namespace:     target.Deployment.Spec.Target.Namespace,
		Labels:        cloneStringMap(target.Labels),
		Enabled:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return releaseTarget, target, nil
}

func fillDeploymentTargetFromCatalog(def *deploymentusecase.Definition, target environment.ReleaseTarget) {
	if def.Spec.Target.Type == "" {
		def.Spec.Target.Type = target.TargetType
	}
	if def.Spec.Target.Name == "" {
		def.Spec.Target.Name = target.Name
	}
	if def.Spec.Target.Context == "" {
		def.Spec.Target.Context = target.Context
	}
	if def.Spec.Target.Namespace == "" {
		def.Spec.Target.Namespace = target.Namespace
	}
}

func (s *Service) securityScanInput(ctx context.Context, def Definition, releaseID string, projectID string, actorID string, plan ReleasePlan) (securityusecase.ScanInput, error) {
	projectID = strings.TrimSpace(projectID)
	environmentID := strings.TrimSpace(def.Spec.Environment)
	if environmentID == "" {
		environmentID = strings.TrimSpace(plan.EnvironmentName)
	}
	input := securityusecase.ScanInput{
		SubjectType:   domainsecurity.SubjectRelease,
		SubjectID:     releaseID,
		ProjectID:     projectID,
		EnvironmentID: environmentID,
		Reference:     strings.Join(plan.ArtifactSummary, ","),
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

func (s *Service) setPlanReleaseStatus(ctx context.Context, record PlanRecord, status domainrelease.ReleaseStatus, actorID string, reason string) (PlanRecord, error) {
	updated, err := s.updateReleaseStatus(ctx, record.Release.ID, status, actorID, reason)
	if err != nil {
		return PlanRecord{}, err
	}
	record.Release = mergeReleaseStatus(record.Release, updated.Release)
	if err := s.store.SavePlan(ctx, record); err != nil {
		return PlanRecord{}, err
	}
	return record, nil
}

func (s *Service) setExecutionReleaseStatus(ctx context.Context, record ExecutionRecord, status domainrelease.ReleaseStatus, actorID string, reason string) (ExecutionRecord, error) {
	updated, err := s.updateReleaseStatus(ctx, record.Execution.ReleaseID, status, actorID, reason)
	if err != nil {
		return ExecutionRecord{}, err
	}
	record.Release = mergeReleaseStatus(record.Release, updated.Release)
	if err := s.store.SaveExecution(ctx, record); err != nil {
		return ExecutionRecord{}, err
	}
	return record, nil
}

func (s *Service) updateReleaseStatus(ctx context.Context, id string, status domainrelease.ReleaseStatus, actorID string, reason string) (artifactusecase.ReleaseRecord, error) {
	if s.artifacts == nil {
		return artifactusecase.ReleaseRecord{}, fmt.Errorf("release artifact service is not configured")
	}
	updated, err := s.artifacts.UpdateReleaseStatus(ctx, id, status, actorID, reason)
	if errors.Is(err, artifactusecase.ErrReleaseNotFound) && id != "" {
		return artifactusecase.ReleaseRecord{Release: domainrelease.Release{ID: id, Status: string(status), UpdatedAt: s.now()}}, nil
	}
	return updated, err
}

func mergeReleaseStatus(current domainrelease.Release, updated domainrelease.Release) domainrelease.Release {
	if updated.ID == "" {
		return current
	}
	if current.ID == "" {
		return updated
	}
	current.Status = updated.Status
	current.UpdatedAt = updated.UpdatedAt
	if updated.Metadata != nil {
		if current.Metadata == nil {
			current.Metadata = map[string]string{}
		}
		for key, value := range updated.Metadata {
			current.Metadata[key] = value
		}
	}
	return current
}

func (s *Service) recordPlanEventAndAudit(ctx context.Context, record PlanRecord, eventType string, action string, actorID string, message string) (PlanRecord, error) {
	evt := s.event(record.Plan.ID, eventType, string(ExecutionPlanning), message)
	record.Events = append(record.Events, evt)
	record.Audits = append(record.Audits, audit.AuditLog{ID: newID("audit"), ActorID: actorID, Action: action, Subject: record.Plan.ID, CreatedAt: s.now()})
	if err := s.store.SavePlan(ctx, record); err != nil {
		return PlanRecord{}, err
	}
	if s.eventBus != nil {
		_ = s.eventBus.Publish(ctx, evt)
	}
	return record, nil
}

func (s *Service) recordExecutionEventAndAudit(ctx context.Context, record ExecutionRecord, eventType string, action string, actorID string, message string) (ExecutionRecord, error) {
	evt := s.event(record.Execution.ID, eventType, string(record.Execution.Status), message)
	record.Events = append(record.Events, evt)
	record.Audits = append(record.Audits, audit.AuditLog{ID: newID("audit"), ActorID: actorID, Action: action, Subject: record.Execution.ID, CreatedAt: s.now()})
	if err := s.store.SaveExecution(ctx, record); err != nil {
		return ExecutionRecord{}, err
	}
	if s.eventBus != nil {
		_ = s.eventBus.Publish(ctx, evt)
	}
	return record, nil
}

func (s *Service) event(subject string, eventType string, status string, message string) domainevent.Event {
	return domainevent.Event{
		ID:              newID("evt"),
		SpecVersion:     "1.0",
		Type:            eventType,
		Source:          "nivora.release-orchestration",
		Subject:         subject,
		Time:            s.now(),
		DataContentType: "application/json",
		Data: map[string]any{
			"status":  status,
			"message": message,
		},
	}
}

func initialTargetExecutions(targets []environment.ReleaseTarget) []TargetExecution {
	executions := make([]TargetExecution, 0, len(targets))
	for i, target := range targets {
		executions = append(executions, TargetExecution{
			TargetID:   target.ID,
			TargetName: target.Name,
			TargetType: target.TargetType,
			Status:     ExecutionCreated,
			Order:      i + 1,
		})
	}
	return executions
}

func firstTargetProjectID(record ExecutionRecord, targetName string) string {
	for _, target := range record.Plan.Targets {
		if target.Name == targetName {
			return target.ProjectID
		}
	}
	return ""
}

func (s *Service) markTargetStarted(record ExecutionRecord, name string) ExecutionRecord {
	for i := range record.Execution.Targets {
		if record.Execution.Targets[i].TargetName == name {
			record.Execution.Targets[i].Status = ExecutionRunning
		}
	}
	return record
}

func (s *Service) markTargetCompleted(record ExecutionRecord, name string, runID string, status ExecutionStatus, warnings []string) ExecutionRecord {
	for i := range record.Execution.Targets {
		if record.Execution.Targets[i].TargetName == name {
			record.Execution.Targets[i].DeploymentRunID = runID
			record.Execution.Targets[i].Status = status
			record.Execution.Targets[i].Warnings = append(record.Execution.Targets[i].Warnings, warnings...)
		}
	}
	return record
}

func replaceDeploymentRecord(record ExecutionRecord, deployment deploymentusecase.RunRecord) ExecutionRecord {
	for i := range record.Deployments {
		if record.Deployments[i].Run.ID == deployment.Run.ID {
			record.Deployments[i] = deployment
			return record
		}
	}
	record.Deployments = append(record.Deployments, deployment)
	return record
}

func markTargetStatusForDeploymentRun(record ExecutionRecord, runID string, status ExecutionStatus) ExecutionRecord {
	for i := range record.Execution.Targets {
		if record.Execution.Targets[i].DeploymentRunID == runID && !isTerminal(record.Execution.Targets[i].Status) {
			record.Execution.Targets[i].Status = status
		}
	}
	return record
}

func appendTargetWarningForDeploymentRun(record ExecutionRecord, runID string, warning string) ExecutionRecord {
	for i := range record.Execution.Targets {
		if record.Execution.Targets[i].DeploymentRunID == runID {
			record.Execution.Targets[i].Warnings = append(record.Execution.Targets[i].Warnings, warning)
		}
	}
	return record
}

func mapDeploymentStatus(status domaindeployment.DeploymentRunStatus) ExecutionStatus {
	switch status {
	case domaindeployment.DeploymentRunSucceeded:
		return ExecutionSucceeded
	case domaindeployment.DeploymentRunFailed:
		return ExecutionFailed
	case domaindeployment.DeploymentRunCanceled:
		return ExecutionCanceled
	default:
		return ExecutionRunning
	}
}

func aggregateStatus(successes int, failures int, total int, continueOnFailure bool) ExecutionStatus {
	if total == 0 {
		return ExecutionFailed
	}
	if failures == 0 && successes == total {
		return ExecutionSucceeded
	}
	if failures > 0 && successes > 0 && continueOnFailure {
		return ExecutionPartiallySucceeded
	}
	return ExecutionFailed
}

func firstDeniedTarget(results []PolicyResult) string {
	for _, result := range results {
		if !result.Allowed {
			return result.Target
		}
	}
	return ""
}

func isTerminal(status ExecutionStatus) bool {
	switch status {
	case ExecutionSucceeded, ExecutionPartiallySucceeded, ExecutionFailed, ExecutionCanceled, ExecutionRolledBack:
		return true
	default:
		return false
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func mergeStringMaps(base map[string]string, override map[string]string) map[string]string {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(override))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range override {
		out[key] = value
	}
	return out
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
