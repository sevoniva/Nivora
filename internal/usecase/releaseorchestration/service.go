package releaseorchestration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	domainevent "github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/ports/eventbus"
	"github.com/sevoniva/nivora/internal/ports/policy"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
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
}

type DeploymentService interface {
	Plan(ctx context.Context, input deploymentusecase.CreateRunInput) (deploymentusecase.CreateRunResult, error)
	CreateAndRun(ctx context.Context, input deploymentusecase.CreateRunInput) (deploymentusecase.CreateRunResult, error)
	Timeline(ctx context.Context, id string) ([]deploymentusecase.TimelineEntry, error)
}

type Service struct {
	store       Store
	artifacts   ArtifactService
	deployments DeploymentService
	policy      policy.Engine
	security    *securityusecase.Service
	eventBus    eventbus.EventBus
	now         func() time.Time
}

func (s *Service) WithSecurity(securityService *securityusecase.Service) *Service {
	s.security = securityService
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
	plan, err := s.buildPlan(ctx, input.Definition, releaseRecord)
	if err != nil {
		return PlanRecord{}, err
	}
	record := PlanRecord{Definition: input.Definition, Release: releaseRecord.Release, Plan: plan}
	if s.security != nil {
		securityRecord, err := s.security.Scan(ctx, securityusecase.ScanInput{
			SubjectType: "release",
			SubjectID:   releaseRecord.Release.ID,
			Reference:   strings.Join(plan.ArtifactSummary, ","),
			Policy:      securityusecase.DefaultPolicyConfig(),
			ActorID:     input.ActorID,
		})
		if err != nil {
			return PlanRecord{}, err
		}
		record.Security = securityRecord
		if securityRecord.Policy.Decision == "deny" {
			record.Plan.Warnings = append(record.Plan.Warnings, securityRecord.Policy.Reason)
		}
	}
	if err := s.store.SavePlan(ctx, record); err != nil {
		return PlanRecord{}, err
	}
	return s.recordPlanEventAndAudit(ctx, record, EventReleasePlanCreated, "Release plan created", input.ActorID, "Release plan created")
}

func (s *Service) Deploy(ctx context.Context, input DeployInput) (ExecutionRecord, error) {
	planRecord, err := s.Plan(ctx, PlanInput{Definition: input.Definition, ActorID: input.ActorID})
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
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionFailed, "Release execution failed", input.ActorID, planRecord.Security.Policy.Reason)
	}
	now := s.now()
	exec := ReleaseExecution{
		ID:              newID("rexec"),
		ReleaseID:       planRecord.Plan.ReleaseID,
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
	if deniedTarget := firstDeniedTarget(record.Plan.PolicyResults); deniedTarget != "" {
		record.Execution.Status = ExecutionFailed
		record.Execution.Reason = "release policy denied target " + deniedTarget
		finished := s.now()
		record.Execution.FinishedAt = &finished
		record.Execution.UpdatedAt = finished
		if err := s.store.SaveExecution(ctx, record); err != nil {
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
		return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionSucceeded, "Release execution succeeded", input.ActorID, "plan-only release execution succeeded")
	}
	if input.Definition.Spec.ApprovalRequired {
		record.Execution.Status = ExecutionWaitingApproval
		record.Execution.Reason = "approval is required; approval workflow is a Phase 2.7 placeholder"
		record.Execution.UpdatedAt = s.now()
		if err := s.store.SaveExecution(ctx, record); err != nil {
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
	return s.recordExecutionEventAndAudit(ctx, record, EventReleaseExecutionCanceled, "Release execution canceled", actorID, "Release execution canceled")
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
		result, runErr := s.deployments.CreateAndRun(ctx, deploymentusecase.CreateRunInput{Definition: target.Deployment})
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
	if record.Execution.Status == ExecutionPartiallySucceeded {
		eventType = EventReleaseExecutionPartiallySucceeded
		message = "Release execution partially succeeded"
	} else if record.Execution.Status == ExecutionFailed {
		eventType = EventReleaseExecutionFailed
		action = "Release execution failed"
		message = record.Execution.Reason
	}
	return s.recordExecutionEventAndAudit(ctx, record, eventType, action, actorID, message)
}

func (s *Service) resolveRelease(ctx context.Context, def Definition, actorID string) (artifactusecase.ReleaseRecord, error) {
	if def.Spec.ReleaseID != "" {
		return s.artifacts.GetRelease(ctx, def.Spec.ReleaseID)
	}
	return s.artifacts.CreateRelease(ctx, artifactusecase.CreateReleaseInput{Definition: def.Spec.Release, ActorID: actorID})
}

func (s *Service) buildPlan(ctx context.Context, def Definition, releaseRecord artifactusecase.ReleaseRecord) (ReleasePlan, error) {
	envID := newID("env")
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
	for _, target := range orderedTargets(def.Spec.Targets) {
		releaseTarget := environment.ReleaseTarget{
			ID:            newID("target"),
			EnvironmentID: envID,
			Name:          target.Name,
			TargetType:    target.Type,
			Context:       target.Deployment.Spec.Target.Context,
			Namespace:     target.Deployment.Spec.Target.Namespace,
			CreatedAt:     plan.CreatedAt,
			UpdatedAt:     plan.CreatedAt,
		}
		plan.Targets = append(plan.Targets, releaseTarget)
		plan.Ordering = append(plan.Ordering, target.Name)
		policyResult, err := s.policy.Evaluate(ctx, policy.Request{
			Subject: releaseRecord.Release.ID,
			Action:  "release.plan",
			Context: map[string]any{"target": target.Name, "targetType": target.Type},
		})
		if err != nil {
			return ReleasePlan{}, err
		}
		plan.PolicyResults = append(plan.PolicyResults, PolicyResult{Target: target.Name, Allowed: policyResult.Allowed, Reasons: policyResult.Reasons})
		if !policyResult.Allowed {
			plan.Warnings = append(plan.Warnings, "release policy denied target "+target.Name)
			continue
		}
		if target.Type == "noop" || target.Type == "webhook" {
			plan.DeploymentPlans = append(plan.DeploymentPlans, deploymentusecase.DeploymentPlan{
				DeploymentRunID: newID("drun"),
				TargetType:      target.Type,
				Actions:         []string{"record target placeholder"},
				Warnings:        []string{"target execution is a Phase 2.7 placeholder"},
				DiffSummary:     "placeholder target has no live diff",
			})
			continue
		}
		result, err := s.deployments.Plan(ctx, deploymentusecase.CreateRunInput{Definition: target.Deployment})
		if err != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("target %s plan failed: %v", target.Name, err))
			continue
		}
		plan.DeploymentPlans = append(plan.DeploymentPlans, result.Record.Plan)
	}
	return plan, nil
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

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}
