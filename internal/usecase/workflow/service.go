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

	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

var ErrNotFound = errors.New("workflow record not found")

type PipelineRunCreator interface {
	CreateQueued(ctx context.Context, input pipelineusecase.CreateRunInput) (pipelineusecase.CreateRunResult, error)
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

func (s *Service) GetRun(ctx context.Context, id string) (RunRecord, error) {
	return s.store.GetRun(ctx, strings.TrimSpace(id))
}

func (s *Service) ListRuns(ctx context.Context, filter RunListFilter) ([]RunRecord, error) {
	filter.RepositoryID = strings.TrimSpace(filter.RepositoryID)
	filter.WorkflowID = strings.TrimSpace(filter.WorkflowID)
	filter.ProjectID = strings.TrimSpace(filter.ProjectID)
	return s.store.ListRuns(ctx, filter)
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
