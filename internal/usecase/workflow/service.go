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

	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

var (
	ErrNotFound    = errors.New("workflow record not found")
	ErrRunTerminal = errors.New("workflow run is already terminal")
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

func (s *Service) Plan(ctx context.Context, input PlanInput) (PlanRecord, error) {
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
		ID:           defaultID("wplan"),
		WorkflowID:   plan.WorkflowID,
		RepositoryID: strings.TrimSpace(input.RepositoryID),
		Path:         strings.TrimSpace(input.Path),
		Ref:          strings.TrimSpace(input.Ref),
		Name:         plan.Name,
		ContentHash:  hash,
		Plan:         plan,
		CreatedAt:    now,
	}
	record.Plan.PlanID = record.ID
	record.Plan.RepositoryID = record.RepositoryID
	record.Plan.SourcePath = record.Path
	record.Plan.Ref = record.Ref
	record.Plan.ContentHash = record.ContentHash
	record.Plan.CreatedAt = now
	if err := s.store.SavePlan(ctx, record); err != nil {
		return PlanRecord{}, err
	}
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
	pipelineResult, err := pipelines.CreateQueued(ctx, pipelineusecase.CreateRunInput{
		Definition:    conversion.Definition,
		ProjectID:     strings.TrimSpace(input.ProjectID),
		EnvironmentID: strings.TrimSpace(input.EnvironmentID),
		ActorID:       strings.TrimSpace(input.ActorID),
		CorrelationID: strings.TrimSpace(input.CorrelationID),
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
		ID:             defaultID("wrun"),
		WorkflowID:     planRecord.WorkflowID,
		WorkflowPlanID: planRecord.ID,
		RepositoryID:   firstNonEmpty(strings.TrimSpace(input.RepositoryID), planRecord.RepositoryID),
		PipelineRunID:  pipelineResult.Record.Run.ID,
		PipelineID:     pipelineResult.Record.Pipeline.ID,
		ProjectID:      strings.TrimSpace(input.ProjectID),
		EnvironmentID:  strings.TrimSpace(input.EnvironmentID),
		Ref:            firstNonEmpty(strings.TrimSpace(input.Ref), planRecord.Ref),
		Status:         RunQueued,
		Warnings:       append([]string(nil), conversion.Warnings...),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.store.SaveRun(ctx, record); err != nil {
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
		}
		result.WorkflowRuns = append(result.WorkflowRuns, refreshed)
	}
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
	return s.store.GetRun(ctx, record.ID)
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

func (s *Service) planRecordForRun(ctx context.Context, input RunInput) (PlanRecord, error) {
	if planID := strings.TrimSpace(input.PlanID); planID != "" {
		return s.GetPlan(ctx, planID)
	}
	return s.Plan(ctx, PlanInput{
		Content:      input.Content,
		RepositoryID: input.RepositoryID,
		Path:         input.Path,
		Ref:          input.Ref,
		Options:      input.Options,
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
