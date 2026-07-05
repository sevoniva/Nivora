package workflow

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type Store interface {
	SavePlan(ctx context.Context, record PlanRecord) error
	GetPlan(ctx context.Context, id string) (PlanRecord, error)
	GetLatestPlan(ctx context.Context, workflowID string) (PlanRecord, error)
	ListPlans(ctx context.Context, filter PlanListFilter) ([]PlanRecord, error)
}

type MemoryStore struct {
	mu    sync.RWMutex
	plans map[string]PlanRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{plans: map[string]PlanRecord{}}
}

func (s *MemoryStore) SavePlan(ctx context.Context, record PlanRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.plans[record.ID] = copyPlanRecord(record)
	return nil
}

func (s *MemoryStore) GetPlan(ctx context.Context, id string) (PlanRecord, error) {
	if err := ctx.Err(); err != nil {
		return PlanRecord{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.plans[strings.TrimSpace(id)]
	if !ok {
		return PlanRecord{}, ErrNotFound
	}
	return copyPlanRecord(record), nil
}

func (s *MemoryStore) GetLatestPlan(ctx context.Context, workflowID string) (PlanRecord, error) {
	plans, err := s.ListPlans(ctx, PlanListFilter{WorkflowID: workflowID})
	if err != nil {
		return PlanRecord{}, err
	}
	if len(plans) == 0 {
		return PlanRecord{}, ErrNotFound
	}
	return plans[0], nil
}

func (s *MemoryStore) ListPlans(ctx context.Context, filter PlanListFilter) ([]PlanRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []PlanRecord
	for _, record := range s.plans {
		if filter.RepositoryID != "" && record.RepositoryID != filter.RepositoryID {
			continue
		}
		if filter.WorkflowID != "" && record.WorkflowID != filter.WorkflowID {
			continue
		}
		out = append(out, copyPlanRecord(record))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return applyPlanPage(out, filter), nil
}

func applyPlanPage(records []PlanRecord, filter PlanListFilter) []PlanRecord {
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(records) {
		return []PlanRecord{}
	}
	records = records[offset:]
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	if len(records) > limit {
		records = records[:limit]
	}
	return records
}

func copyPlanRecord(in PlanRecord) PlanRecord {
	out := in
	out.Plan = copyPlan(in.Plan)
	return out
}

func copyPlan(in Plan) Plan {
	out := in
	out.Triggers = append([]string(nil), in.Triggers...)
	out.Jobs = append([]PlannedJob(nil), in.Jobs...)
	for i := range out.Jobs {
		out.Jobs[i].Needs = append([]string(nil), in.Jobs[i].Needs...)
		out.Jobs[i].RunsOn = append([]string(nil), in.Jobs[i].RunsOn...)
		out.Jobs[i].Matrix = copyStringMap(in.Jobs[i].Matrix)
	}
	out.Steps = append([]PlannedStep(nil), in.Steps...)
	for i := range out.Steps {
		out.Steps[i].Env = copyStringMap(in.Steps[i].Env)
	}
	out.Edges = append([]Edge(nil), in.Edges...)
	out.MatrixExpansions = append([]MatrixExpansion(nil), in.MatrixExpansions...)
	for i := range out.MatrixExpansions {
		out.MatrixExpansions[i].Values = copyStringMap(in.MatrixExpansions[i].Values)
	}
	out.RunnerRequirements = append([]RunnerRequirement(nil), in.RunnerRequirements...)
	for i := range out.RunnerRequirements {
		out.RunnerRequirements[i].RunsOn = append([]string(nil), in.RunnerRequirements[i].RunsOn...)
	}
	out.ArtifactOutputs = append([]ArtifactSpec(nil), in.ArtifactOutputs...)
	out.CacheHints = append([]CacheSpec(nil), in.CacheHints...)
	out.SecurityWarnings = append([]string(nil), in.SecurityWarnings...)
	out.UnsupportedFeatures = append([]string(nil), in.UnsupportedFeatures...)
	out.Warnings = append([]string(nil), in.Warnings...)
	return out
}
