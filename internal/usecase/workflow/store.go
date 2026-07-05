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
	SaveRun(ctx context.Context, record RunRecord) error
	GetRun(ctx context.Context, id string) (RunRecord, error)
	ListRuns(ctx context.Context, filter RunListFilter) ([]RunRecord, error)
}

type MemoryStore struct {
	mu    sync.RWMutex
	plans map[string]PlanRecord
	runs  map[string]RunRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{plans: map[string]PlanRecord{}, runs: map[string]RunRecord{}}
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

func (s *MemoryStore) SaveRun(ctx context.Context, record RunRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[record.ID] = copyRunRecord(record)
	return nil
}

func (s *MemoryStore) GetRun(ctx context.Context, id string) (RunRecord, error) {
	if err := ctx.Err(); err != nil {
		return RunRecord{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.runs[strings.TrimSpace(id)]
	if !ok {
		return RunRecord{}, ErrNotFound
	}
	return copyRunRecord(record), nil
}

func (s *MemoryStore) ListRuns(ctx context.Context, filter RunListFilter) ([]RunRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []RunRecord
	for _, record := range s.runs {
		if filter.RepositoryID != "" && record.RepositoryID != filter.RepositoryID {
			continue
		}
		if filter.WorkflowID != "" && record.WorkflowID != filter.WorkflowID {
			continue
		}
		if filter.ProjectID != "" && record.ProjectID != filter.ProjectID {
			continue
		}
		if filter.Status != "" && record.Status != filter.Status {
			continue
		}
		out = append(out, copyRunRecord(record))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return applyRunPage(out, filter), nil
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

func applyRunPage(records []RunRecord, filter RunListFilter) []RunRecord {
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(records) {
		return []RunRecord{}
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

func copyRunRecord(in RunRecord) RunRecord {
	out := in
	out.Warnings = append([]string(nil), in.Warnings...)
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
	for i := range out.ArtifactOutputs {
		out.ArtifactOutputs[i].Metadata = copyStringMap(in.ArtifactOutputs[i].Metadata)
	}
	out.CacheHints = append([]CacheSpec(nil), in.CacheHints...)
	for i := range out.CacheHints {
		out.CacheHints[i].Path = append([]string(nil), in.CacheHints[i].Path...)
		out.CacheHints[i].RestoreKeys = append([]string(nil), in.CacheHints[i].RestoreKeys...)
		out.CacheHints[i].Metadata = copyStringMap(in.CacheHints[i].Metadata)
	}
	out.SecurityIntent = copySecurityIntent(in.SecurityIntent)
	out.ReleaseIntent = copyReleaseIntent(in.ReleaseIntent)
	out.DeploymentIntent = copyDeploymentIntent(in.DeploymentIntent)
	out.SecurityWarnings = append([]string(nil), in.SecurityWarnings...)
	out.UnsupportedFeatures = append([]string(nil), in.UnsupportedFeatures...)
	out.Warnings = append([]string(nil), in.Warnings...)
	return out
}

func copySecurityIntent(in *SecurityIntentPlan) *SecurityIntentPlan {
	if in == nil {
		return nil
	}
	out := *in
	out.Scanners = append([]string(nil), in.Scanners...)
	out.Warnings = append([]string(nil), in.Warnings...)
	out.UnsupportedKeys = append([]string(nil), in.UnsupportedKeys...)
	return &out
}

func copyReleaseIntent(in *ReleaseIntentPlan) *ReleaseIntentPlan {
	if in == nil {
		return nil
	}
	out := *in
	out.Artifacts = append([]string(nil), in.Artifacts...)
	out.Warnings = append([]string(nil), in.Warnings...)
	out.UnsupportedKeys = append([]string(nil), in.UnsupportedKeys...)
	return &out
}

func copyDeploymentIntent(in *DeploymentIntentPlan) *DeploymentIntentPlan {
	if in == nil {
		return nil
	}
	out := *in
	out.Warnings = append([]string(nil), in.Warnings...)
	out.UnsupportedKeys = append([]string(nil), in.UnsupportedKeys...)
	return &out
}
