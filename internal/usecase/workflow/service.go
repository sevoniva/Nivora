package workflow

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

var (
	ErrNotFound        = errors.New("workflow record not found")
	ErrRunTerminal     = errors.New("workflow run is already terminal")
	ErrRunNotRetryable = errors.New("workflow run is not retryable")
)

const (
	EventWorkflowValidated      = "devops.workflow.validated"
	EventWorkflowPlanCreated    = "devops.workflow.plan.created"
	EventWorkflowRunRequested   = "devops.workflow.run.requested"
	EventWorkflowMatrixExpanded = "devops.workflow.matrix.expanded"
	EventWorkflowRunCanceled    = "devops.workflow.run.canceled"
	EventWorkflowRunRetried     = "devops.workflow.run.retried"
	EventWorkflowRunReconciled  = "devops.workflow.run.reconciled"
)

type PipelineRunCreator interface {
	CreateQueued(ctx context.Context, input pipelineusecase.CreateRunInput) (pipelineusecase.CreateRunResult, error)
}

type PipelineMetadataRecorder interface {
	RecordArtifact(ctx context.Context, artifact pipelineusecase.PipelineArtifact) (pipelineusecase.PipelineArtifact, error)
	RecordCacheEntry(ctx context.Context, entry pipelineusecase.PipelineCacheEntry) (pipelineusecase.PipelineCacheEntry, error)
}

type PipelineRunReader interface {
	Get(ctx context.Context, id string) (pipelineusecase.RunRecord, error)
}

type PipelineRunCanceler interface {
	Cancel(ctx context.Context, id string, actorID string) (pipelineusecase.RunRecord, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{store: store, now: time.Now}
}

func (s *Service) Validate(ctx context.Context, input PlanInput) (Plan, error) {
	record, err := s.buildPlanRecord(ctx, input)
	if err != nil {
		return Plan{}, err
	}
	if err := s.recordWorkflowEventAndAudit(ctx, record.WorkflowID, EventWorkflowValidated, "workflow validated", "", map[string]any{
		"workflowId":           record.WorkflowID,
		"repositoryId":         record.RepositoryID,
		"repositorySnapshotId": record.RepositorySnapshotID,
		"path":                 record.Path,
		"ref":                  record.Ref,
		"jobCount":             len(record.Plan.Jobs),
		"stepCount":            len(record.Plan.Steps),
		"conversionReady":      record.Plan.ConversionReady,
	}); err != nil {
		return Plan{}, err
	}
	if len(record.Plan.MatrixExpansions) > 0 {
		if err := s.recordWorkflowEventAndAudit(ctx, record.WorkflowID, EventWorkflowMatrixExpanded, "workflow matrix expanded", "", map[string]any{
			"workflowId":     record.WorkflowID,
			"repositoryId":   record.RepositoryID,
			"expansionCount": len(record.Plan.MatrixExpansions),
		}); err != nil {
			return Plan{}, err
		}
	}
	return record.Plan, nil
}

func (s *Service) Plan(ctx context.Context, input PlanInput) (PlanRecord, error) {
	record, err := s.buildPlanRecord(ctx, input)
	if err != nil {
		return PlanRecord{}, err
	}
	if err := s.store.SavePlan(ctx, record); err != nil {
		return PlanRecord{}, err
	}
	if err := s.recordWorkflowEventAndAudit(ctx, record.ID, EventWorkflowPlanCreated, "workflow plan created", "", map[string]any{
		"workflowId":           record.WorkflowID,
		"workflowPlanId":       record.ID,
		"repositoryId":         record.RepositoryID,
		"repositorySnapshotId": record.RepositorySnapshotID,
		"path":                 record.Path,
		"ref":                  record.Ref,
		"jobCount":             len(record.Plan.Jobs),
		"stepCount":            len(record.Plan.Steps),
		"conversionReady":      record.Plan.ConversionReady,
	}); err != nil {
		return PlanRecord{}, err
	}
	if len(record.Plan.MatrixExpansions) > 0 {
		if err := s.recordWorkflowEventAndAudit(ctx, record.ID, EventWorkflowMatrixExpanded, "workflow matrix expanded", "", map[string]any{
			"workflowId":           record.WorkflowID,
			"workflowPlanId":       record.ID,
			"repositoryId":         record.RepositoryID,
			"repositorySnapshotId": record.RepositorySnapshotID,
			"expansionCount":       len(record.Plan.MatrixExpansions),
		}); err != nil {
			return PlanRecord{}, err
		}
	}
	return record, nil
}

func (s *Service) buildPlanRecord(ctx context.Context, input PlanInput) (PlanRecord, error) {
	if err := ctx.Err(); err != nil {
		return PlanRecord{}, err
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return PlanRecord{}, fmt.Errorf("%w: workflow content is required", ErrInvalid)
	}
	def, err := ParseDefinition([]byte(content))
	if err != nil {
		return PlanRecord{}, err
	}
	plan, err := PlanDefinition(def, input.Options)
	if err != nil {
		return PlanRecord{}, err
	}
	now := s.now().UTC()
	hash := contentHash(content)
	record := PlanRecord{
		ID:                   defaultID("wplan"),
		WorkflowID:           plan.WorkflowID,
		RepositoryID:         strings.TrimSpace(input.RepositoryID),
		RepositorySnapshotID: strings.TrimSpace(input.RepositorySnapshotID),
		Path:                 strings.TrimSpace(input.Path),
		Ref:                  strings.TrimSpace(input.Ref),
		Name:                 plan.Name,
		ContentHash:          hash,
		Plan:                 plan,
		CreatedAt:            now,
	}
	record.Plan.PlanID = record.ID
	record.Plan.RepositoryID = record.RepositoryID
	record.Plan.RepositorySnapshotID = record.RepositorySnapshotID
	record.Plan.SourcePath = record.Path
	record.Plan.Ref = record.Ref
	record.Plan.ContentHash = record.ContentHash
	record.Plan.CreatedAt = now
	return record, nil
}

func (s *Service) GetPlan(ctx context.Context, id string) (PlanRecord, error) {
	return s.store.GetPlan(ctx, strings.TrimSpace(id))
}

func (s *Service) GetLatestPlan(ctx context.Context, workflowID string) (PlanRecord, error) {
	return s.store.GetLatestPlan(ctx, strings.TrimSpace(workflowID))
}

func (s *Service) ListPlans(ctx context.Context, filter PlanListFilter) ([]PlanRecord, error) {
	filter.RepositoryID = strings.TrimSpace(filter.RepositoryID)
	filter.WorkflowID = strings.TrimSpace(filter.WorkflowID)
	return s.store.ListPlans(ctx, filter)
}

func (s *Service) ListWorkflows(ctx context.Context, filter PlanListFilter) ([]WorkflowSummary, error) {
	filter.WorkflowID = strings.TrimSpace(filter.WorkflowID)
	filter.RepositoryID = strings.TrimSpace(filter.RepositoryID)
	filter.Limit = 100
	filter.Offset = 0
	plans, err := s.store.ListPlans(ctx, filter)
	if err != nil {
		return nil, err
	}
	summaries := map[string]WorkflowSummary{}
	order := []string{}
	for _, plan := range plans {
		key := plan.WorkflowID
		if key == "" {
			continue
		}
		summary, ok := summaries[key]
		if !ok {
			order = append(order, key)
			summary = WorkflowSummary{
				WorkflowID:   key,
				Name:         plan.Name,
				RepositoryID: plan.RepositoryID,
				LatestPlanID: plan.ID,
				ContentHash:  plan.ContentHash,
				Ref:          plan.Ref,
				UpdatedAt:    plan.CreatedAt,
			}
		}
		summary.PlanCount++
		if plan.CreatedAt.After(summary.UpdatedAt) {
			summary.Name = plan.Name
			summary.RepositoryID = plan.RepositoryID
			summary.LatestPlanID = plan.ID
			summary.ContentHash = plan.ContentHash
			summary.Ref = plan.Ref
			summary.UpdatedAt = plan.CreatedAt
		}
		summaries[key] = summary
	}
	out := make([]WorkflowSummary, 0, len(order))
	for _, key := range order {
		out = append(out, summaries[key])
	}
	return out, nil
}

func (s *Service) GetWorkflow(ctx context.Context, workflowID string) (WorkflowSummary, error) {
	workflows, err := s.ListWorkflows(ctx, PlanListFilter{WorkflowID: strings.TrimSpace(workflowID)})
	if err != nil {
		return WorkflowSummary{}, err
	}
	for _, summary := range workflows {
		if summary.WorkflowID == strings.TrimSpace(workflowID) {
			return summary, nil
		}
	}
	return WorkflowSummary{}, ErrNotFound
}

func (s *Service) Run(ctx context.Context, input RunInput, pipelines PipelineRunCreator) (RunResult, error) {
	if pipelines == nil {
		return RunResult{}, fmt.Errorf("%w: pipeline runtime is required", ErrInvalid)
	}
	if !input.Confirm || !input.AllowPipelineRun {
		return RunResult{}, fmt.Errorf("%w: workflow run requires confirm=true and allowPipelineRun=true", ErrInvalid)
	}
	planRecord, err := s.planRecordForRun(ctx, input)
	if err != nil {
		return RunResult{}, err
	}
	conversion, err := ToPipelineDefinitionFromPlan(planRecord.Plan)
	if err != nil {
		return RunResult{}, err
	}
	workflowRunID := defaultID("wrun")
	pipelineResult, err := pipelines.CreateQueued(ctx, pipelineusecase.CreateRunInput{
		Definition:    conversion.Definition,
		ProjectID:     strings.TrimSpace(input.ProjectID),
		EnvironmentID: strings.TrimSpace(input.EnvironmentID),
		ActorID:       strings.TrimSpace(input.ActorID),
		CorrelationID: strings.TrimSpace(input.CorrelationID),
		Workflow: pipelineusecase.WorkflowRunMetadata{
			WorkflowID:           planRecord.WorkflowID,
			WorkflowPlanID:       planRecord.ID,
			WorkflowRunID:        workflowRunID,
			RepositoryID:         firstNonEmpty(strings.TrimSpace(input.RepositoryID), planRecord.RepositoryID),
			RepositorySnapshotID: firstNonEmpty(strings.TrimSpace(input.RepositorySnapshotID), planRecord.RepositorySnapshotID, planRecord.Plan.RepositorySnapshotID),
			SourcePath:           planRecord.Path,
			Ref:                  firstNonEmpty(strings.TrimSpace(input.Ref), planRecord.Ref),
		},
	})
	if err != nil {
		return RunResult{}, err
	}
	if recorder, ok := pipelines.(PipelineMetadataRecorder); ok {
		metadataWarnings, err := s.recordPipelinePlanMetadata(ctx, planRecord, pipelineResult.Record.Run.ID, recorder)
		if err != nil {
			return RunResult{}, err
		}
		conversion.Warnings = append(conversion.Warnings, metadataWarnings...)
	}
	now := s.now().UTC()
	record := RunRecord{
		ID:                   workflowRunID,
		WorkflowID:           planRecord.WorkflowID,
		WorkflowPlanID:       planRecord.ID,
		RepositoryID:         firstNonEmpty(strings.TrimSpace(input.RepositoryID), planRecord.RepositoryID),
		RepositorySnapshotID: firstNonEmpty(strings.TrimSpace(input.RepositorySnapshotID), planRecord.RepositorySnapshotID, planRecord.Plan.RepositorySnapshotID),
		PipelineRunID:        pipelineResult.Record.Run.ID,
		PipelineID:           pipelineResult.Record.Pipeline.ID,
		ProjectID:            strings.TrimSpace(input.ProjectID),
		EnvironmentID:        strings.TrimSpace(input.EnvironmentID),
		Ref:                  firstNonEmpty(strings.TrimSpace(input.Ref), planRecord.Ref),
		Status:               RunQueued,
		Warnings:             append([]string(nil), conversion.Warnings...),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := s.store.SaveRun(ctx, record); err != nil {
		return RunResult{}, err
	}
	if err := s.recordWorkflowEventAndAudit(ctx, record.ID, EventWorkflowRunRequested, "workflow run requested", strings.TrimSpace(input.ActorID), map[string]any{
		"workflowId":           record.WorkflowID,
		"workflowPlanId":       record.WorkflowPlanID,
		"workflowRunId":        record.ID,
		"repositoryId":         record.RepositoryID,
		"repositorySnapshotId": record.RepositorySnapshotID,
		"pipelineRunId":        record.PipelineRunID,
		"projectId":            record.ProjectID,
		"environmentId":        record.EnvironmentID,
		"status":               string(record.Status),
	}); err != nil {
		return RunResult{}, err
	}
	return RunResult{
		WorkflowRun: record,
		PipelineRun: pipelineResult.Record,
		Conversion:  conversion,
		Plan:        planRecord.Plan,
		Warnings:    append([]string(nil), record.Warnings...),
	}, nil
}

func (s *Service) recordPipelinePlanMetadata(ctx context.Context, planRecord PlanRecord, pipelineRunID string, recorder PipelineMetadataRecorder) ([]string, error) {
	if strings.TrimSpace(pipelineRunID) == "" || recorder == nil {
		return nil, nil
	}
	warnings := []string{}
	baseMetadata := map[string]string{
		"workflowId":     planRecord.WorkflowID,
		"workflowPlanId": planRecord.ID,
		"source":         "workflow-plan",
	}
	for _, artifact := range planRecord.Plan.ArtifactOutputs {
		metadata := mergeStringMetadata(baseMetadata, artifact.Metadata)
		if artifact.Path != "" {
			metadata["path"] = artifact.Path
		}
		_, err := recorder.RecordArtifact(ctx, pipelineusecase.PipelineArtifact{
			PipelineRunID: pipelineRunID,
			Name:          artifact.Name,
			Type:          firstNonEmpty(artifact.Type, "workflow-artifact"),
			ContentHash:   artifact.ContentHash,
			StorageRef:    artifact.StorageRef,
			RetentionDays: artifact.RetentionDays,
			Metadata:      metadata,
		})
		if err != nil {
			return nil, err
		}
	}
	for _, cache := range planRecord.Plan.CacheHints {
		metadata := mergeStringMetadata(baseMetadata, cache.Metadata)
		if len(cache.Path) > 0 {
			metadata["paths"] = strings.Join(cache.Path, ",")
		}
		_, err := recorder.RecordCacheEntry(ctx, pipelineusecase.PipelineCacheEntry{
			PipelineRunID: pipelineRunID,
			Key:           cache.Key,
			RestoreKeys:   append([]string(nil), cache.RestoreKeys...),
			Scope:         cache.Scope,
			Metadata:      metadata,
		})
		if err != nil {
			return nil, err
		}
	}
	if len(planRecord.Plan.ArtifactOutputs) > 0 || len(planRecord.Plan.CacheHints) > 0 {
		warnings = append(warnings, "workflow artifact and cache declarations were recorded as PipelineRun metadata only; no artifact or cache blob was read or uploaded")
	}
	return warnings, nil
}

func mergeStringMetadata(values ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, metadata := range values {
		for key, value := range metadata {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if key != "" && value != "" {
				out[key] = value
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Service) GetRun(ctx context.Context, id string) (RunRecord, error) {
	return s.store.GetRun(ctx, strings.TrimSpace(id))
}

func (s *Service) ListRuns(ctx context.Context, filter RunListFilter) ([]RunRecord, error) {
	filter.RepositoryID = strings.TrimSpace(filter.RepositoryID)
	filter.WorkflowID = strings.TrimSpace(filter.WorkflowID)
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	return s.store.ListRuns(ctx, filter)
}

func (s *Service) RefreshRunStatus(ctx context.Context, id string, pipelines PipelineRunReader) (RunRecord, error) {
	record, err := s.GetRun(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	return s.refreshRunRecordStatus(ctx, record, pipelines)
}

func (s *Service) RefreshRuns(ctx context.Context, filter RunListFilter, pipelines PipelineRunReader) ([]RunRecord, error) {
	filter.RepositoryID = strings.TrimSpace(filter.RepositoryID)
	filter.WorkflowID = strings.TrimSpace(filter.WorkflowID)
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	requestedStatus := filter.Status
	filter.Status = ""
	records, err := s.store.ListRuns(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]RunRecord, 0, len(records))
	for _, record := range records {
		refreshed, err := s.refreshRunRecordStatus(ctx, record, pipelines)
		if err != nil {
			return nil, err
		}
		if requestedStatus != "" && refreshed.Status != requestedStatus {
			continue
		}
		out = append(out, refreshed)
	}
	return out, nil
}

func (s *Service) ReconcileRuns(ctx context.Context, filter RunListFilter, pipelines PipelineRunReader) (ReconcileResult, error) {
	filter.RepositoryID = strings.TrimSpace(filter.RepositoryID)
	filter.WorkflowID = strings.TrimSpace(filter.WorkflowID)
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	requestedStatus := filter.Status
	filter.Status = ""
	records, err := s.store.ListRuns(ctx, filter)
	if err != nil {
		return ReconcileResult{}, err
	}
	result := ReconcileResult{WorkflowRuns: []RunRecord{}}
	if pipelines == nil {
		result.Warnings = append(result.Warnings, "workflow reconciliation could not read linked PipelineRun state")
	}
	for _, record := range records {
		if requestedStatus != "" && record.Status != requestedStatus {
			continue
		}
		if isTerminalRunStatus(record.Status) {
			continue
		}
		result.Scanned++
		refreshed := record
		if pipelines != nil {
			refreshed, err = s.refreshRunRecordStatus(ctx, record, pipelines)
			if err != nil {
				return ReconcileResult{}, err
			}
		}
		if refreshed.Status != record.Status {
			result.Updated++
			if err := s.recordWorkflowEventAndAudit(ctx, refreshed.ID, EventWorkflowRunReconciled, "workflow run reconciled", "", map[string]any{
				"workflowId":     refreshed.WorkflowID,
				"workflowPlanId": refreshed.WorkflowPlanID,
				"workflowRunId":  refreshed.ID,
				"pipelineRunId":  refreshed.PipelineRunID,
				"previousStatus": string(record.Status),
				"status":         string(refreshed.Status),
			}); err != nil {
				return ReconcileResult{}, err
			}
		}
		result.WorkflowRuns = append(result.WorkflowRuns, refreshed)
	}
	return result, nil
}

func (s *Service) RetryRun(ctx context.Context, id string, input RetryInput, pipelines PipelineRunCreator) (RunResult, error) {
	if pipelines == nil {
		return RunResult{}, fmt.Errorf("%w: pipeline runtime is required", ErrInvalid)
	}
	if !input.Confirm || !input.AllowPipelineRun {
		return RunResult{}, fmt.Errorf("%w: workflow retry requires confirm=true and allowPipelineRun=true", ErrInvalid)
	}
	record, err := s.GetRun(ctx, id)
	if err != nil {
		return RunResult{}, err
	}
	if reader, ok := pipelines.(PipelineRunReader); ok {
		record, err = s.refreshRunRecordStatus(ctx, record, reader)
		if err != nil {
			return RunResult{}, err
		}
	}
	if !isRetryableRunStatus(record.Status) {
		return RunResult{}, fmt.Errorf("%w: workflow run %s is %s", ErrRunNotRetryable, record.ID, record.Status)
	}
	result, err := s.Run(ctx, RunInput{
		PlanID:               record.WorkflowPlanID,
		RepositoryID:         record.RepositoryID,
		RepositorySnapshotID: record.RepositorySnapshotID,
		ProjectID:            record.ProjectID,
		EnvironmentID:        record.EnvironmentID,
		Ref:                  record.Ref,
		ActorID:              strings.TrimSpace(input.ActorID),
		CorrelationID:        strings.TrimSpace(input.CorrelationID),
		Confirm:              input.Confirm,
		AllowPipelineRun:     input.AllowPipelineRun,
	}, pipelines)
	if err != nil {
		return RunResult{}, err
	}
	warning := fmt.Sprintf("workflow run retried from %s", record.ID)
	result.WorkflowRun.Warnings = append(result.WorkflowRun.Warnings, warning)
	result.WorkflowRun.UpdatedAt = s.now().UTC()
	if err := s.store.SaveRun(ctx, result.WorkflowRun); err != nil {
		return RunResult{}, err
	}
	if err := s.recordWorkflowEventAndAudit(ctx, result.WorkflowRun.ID, EventWorkflowRunRetried, "workflow run retried", strings.TrimSpace(input.ActorID), map[string]any{
		"workflowId":     result.WorkflowRun.WorkflowID,
		"workflowPlanId": result.WorkflowRun.WorkflowPlanID,
		"workflowRunId":  result.WorkflowRun.ID,
		"originalRunId":  record.ID,
		"pipelineRunId":  result.WorkflowRun.PipelineRunID,
		"projectId":      result.WorkflowRun.ProjectID,
		"environmentId":  result.WorkflowRun.EnvironmentID,
		"correlationId":  strings.TrimSpace(input.CorrelationID),
		"status":         string(result.WorkflowRun.Status),
	}); err != nil {
		return RunResult{}, err
	}
	result.WorkflowRun, err = s.store.GetRun(ctx, result.WorkflowRun.ID)
	if err != nil {
		return RunResult{}, err
	}
	result.Warnings = append(result.Warnings, warning)
	return result, nil
}

func (s *Service) CancelRun(ctx context.Context, id string, actorID string, pipelines PipelineRunCanceler) (RunRecord, error) {
	record, err := s.GetRun(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if isTerminalRunStatus(record.Status) {
		return record, ErrRunTerminal
	}
	if pipelines != nil && strings.TrimSpace(record.PipelineRunID) != "" {
		pipelineRecord, err := pipelines.Cancel(ctx, record.PipelineRunID, strings.TrimSpace(actorID))
		if err != nil {
			if errors.Is(err, pipelineusecase.ErrRunTerminal) {
				if status := runStatusFromPipeline(pipelineRecord.Run.Status); status != "" && status != record.Status {
					record.Status = status
					record.UpdatedAt = s.now().UTC()
					if saveErr := s.store.SaveRun(ctx, record); saveErr != nil {
						return RunRecord{}, saveErr
					}
				}
				return record, ErrRunTerminal
			}
			return RunRecord{}, err
		}
		if status := runStatusFromPipeline(pipelineRecord.Run.Status); status != "" {
			record.Status = status
		} else {
			record.Status = RunCanceled
		}
	} else {
		record.Status = RunCanceled
		record.Warnings = append(record.Warnings, "workflow run was marked canceled without a linked PipelineRun cancellation")
	}
	record.UpdatedAt = s.now().UTC()
	if err := s.store.SaveRun(ctx, record); err != nil {
		return RunRecord{}, err
	}
	loaded, err := s.store.GetRun(ctx, record.ID)
	if err != nil {
		return RunRecord{}, err
	}
	if err := s.recordWorkflowEventAndAudit(ctx, loaded.ID, EventWorkflowRunCanceled, "workflow run canceled", strings.TrimSpace(actorID), map[string]any{
		"workflowId":     loaded.WorkflowID,
		"workflowPlanId": loaded.WorkflowPlanID,
		"workflowRunId":  loaded.ID,
		"pipelineRunId":  loaded.PipelineRunID,
		"projectId":      loaded.ProjectID,
		"environmentId":  loaded.EnvironmentID,
		"status":         string(loaded.Status),
	}); err != nil {
		return RunRecord{}, err
	}
	return loaded, nil
}

func (s *Service) refreshRunRecordStatus(ctx context.Context, record RunRecord, pipelines PipelineRunReader) (RunRecord, error) {
	if pipelines == nil || strings.TrimSpace(record.PipelineRunID) == "" {
		return record, nil
	}
	pipelineRecord, err := pipelines.Get(ctx, record.PipelineRunID)
	if err != nil {
		if errors.Is(err, pipelineusecase.ErrRunNotFound) {
			return record, nil
		}
		return RunRecord{}, err
	}
	status := runStatusFromPipeline(pipelineRecord.Run.Status)
	if status == "" || status == record.Status {
		return record, nil
	}
	record.Status = status
	record.UpdatedAt = s.now().UTC()
	if err := s.store.SaveRun(ctx, record); err != nil {
		return RunRecord{}, err
	}
	return s.store.GetRun(ctx, record.ID)
}

func isTerminalRunStatus(status RunStatus) bool {
	switch status {
	case RunSucceeded, RunFailed, RunCanceled, RunTimeout:
		return true
	default:
		return false
	}
}

func isRetryableRunStatus(status RunStatus) bool {
	switch status {
	case RunFailed, RunCanceled, RunTimeout:
		return true
	default:
		return false
	}
}

func runStatusFromPipeline(status domainpipeline.PipelineRunStatus) RunStatus {
	switch status {
	case domainpipeline.PipelineRunPending:
		return RunPending
	case domainpipeline.PipelineRunQueued:
		return RunQueued
	case domainpipeline.PipelineRunRunning:
		return RunRunning
	case domainpipeline.PipelineRunPaused:
		return RunPaused
	case domainpipeline.PipelineRunSucceeded:
		return RunSucceeded
	case domainpipeline.PipelineRunFailed:
		return RunFailed
	case domainpipeline.PipelineRunCanceled:
		return RunCanceled
	case domainpipeline.PipelineRunTimeout:
		return RunTimeout
	default:
		return ""
	}
}

func (s *Service) Events(ctx context.Context, subject string) ([]event.Event, error) {
	return s.store.EventsBySubject(ctx, strings.TrimSpace(subject))
}

func (s *Service) Audits(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	return s.store.AuditsBySubject(ctx, strings.TrimSpace(subject))
}

func (s *Service) recordWorkflowEventAndAudit(ctx context.Context, subject string, eventType string, action string, actorID string, data map[string]any) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = strings.TrimSpace(valueFromData(data, "workflowRunId"))
	}
	if subject == "" {
		subject = strings.TrimSpace(valueFromData(data, "workflowPlanId"))
	}
	if subject == "" {
		subject = strings.TrimSpace(valueFromData(data, "workflowId"))
	}
	if subject == "" {
		return nil
	}
	now := s.now().UTC()
	payload := cloneEventData(data)
	payload["message"] = action
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              defaultID("evt"),
		Type:            eventType,
		Source:          "nivora/workflow",
		Subject:         subject,
		Time:            now,
		DataContentType: "application/json",
		Data:            payload,
	}
	if err := s.store.AppendEvent(ctx, subject, evt); err != nil {
		return err
	}
	scopeType := "workflow"
	scopeID := subject
	if projectID := strings.TrimSpace(valueFromData(payload, "projectId")); projectID != "" {
		scopeType = "project"
		scopeID = projectID
	} else if repositoryID := strings.TrimSpace(valueFromData(payload, "repositoryId")); repositoryID != "" {
		scopeType = "repository"
		scopeID = repositoryID
	}
	return s.store.AppendAudit(ctx, subject, audit.AuditLog{
		ID:          defaultID("audit"),
		ActorID:     strings.TrimSpace(actorID),
		Action:      action,
		Subject:     subject,
		SubjectType: "workflow",
		SubjectID:   subject,
		ScopeType:   scopeType,
		ScopeID:     scopeID,
		Metadata:    auditMetadata(payload),
		CreatedAt:   now,
	})
}

func cloneEventData(data map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range data {
		if strings.TrimSpace(key) != "" {
			out[key] = value
		}
	}
	return out
}

func valueFromData(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return value
}

func auditMetadata(data map[string]any) map[string]string {
	if len(data) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range data {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch typed := value.(type) {
		case string:
			if typed != "" {
				out[key] = typed
			}
		case fmt.Stringer:
			out[key] = typed.String()
		case bool:
			out[key] = fmt.Sprintf("%t", typed)
		case int:
			out[key] = fmt.Sprintf("%d", typed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *Service) planRecordForRun(ctx context.Context, input RunInput) (PlanRecord, error) {
	if planID := strings.TrimSpace(input.PlanID); planID != "" {
		return s.GetPlan(ctx, planID)
	}
	return s.Plan(ctx, PlanInput{
		Content:              input.Content,
		RepositoryID:         input.RepositoryID,
		RepositorySnapshotID: input.RepositorySnapshotID,
		Path:                 input.Path,
		Ref:                  input.Ref,
		Options:              input.Options,
	})
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func defaultID(prefix string) string {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UTC().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(raw[:])
}
