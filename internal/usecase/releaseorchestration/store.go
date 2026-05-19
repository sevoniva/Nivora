package releaseorchestration

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/event"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

var (
	ErrPlanNotFound      = errors.New("release plan not found")
	ErrExecutionNotFound = errors.New("release execution not found")
	ErrExecutionTerminal = errors.New("release execution is already terminal")
)

type Store interface {
	SavePlan(ctx context.Context, record PlanRecord) error
	GetPlan(ctx context.Context, id string) (PlanRecord, error)
	GetLatestPlanForRelease(ctx context.Context, releaseID string) (PlanRecord, error)
	SaveExecution(ctx context.Context, record ExecutionRecord) error
	GetExecution(ctx context.Context, id string) (ExecutionRecord, error)
	ListExecutions(ctx context.Context, releaseID string) ([]ExecutionRecord, error)
	AppendEvent(ctx context.Context, executionID string, evt event.Event) error
	Events(ctx context.Context, executionID string) ([]event.Event, error)
	AppendAudit(ctx context.Context, executionID string, entry audit.AuditLog) error
	Audits(ctx context.Context, executionID string) ([]audit.AuditLog, error)
}

type MemoryStore struct {
	mu         sync.RWMutex
	plans      map[string]PlanRecord
	executions map[string]ExecutionRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		plans:      make(map[string]PlanRecord),
		executions: make(map[string]ExecutionRecord),
	}
}

func (s *MemoryStore) SavePlan(ctx context.Context, record PlanRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.plans[record.Plan.ID]; ok {
		if record.Events == nil {
			record.Events = existing.Events
		}
		if record.Audits == nil {
			record.Audits = existing.Audits
		}
	}
	s.plans[record.Plan.ID] = clonePlan(record)
	return nil
}

func (s *MemoryStore) GetPlan(ctx context.Context, id string) (PlanRecord, error) {
	select {
	case <-ctx.Done():
		return PlanRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.plans[id]
	if !ok {
		return PlanRecord{}, ErrPlanNotFound
	}
	return clonePlan(record), nil
}

func (s *MemoryStore) GetLatestPlanForRelease(ctx context.Context, releaseID string) (PlanRecord, error) {
	select {
	case <-ctx.Done():
		return PlanRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var matches []PlanRecord
	for _, record := range s.plans {
		if record.Plan.ReleaseID == releaseID {
			matches = append(matches, clonePlan(record))
		}
	}
	if len(matches) == 0 {
		return PlanRecord{}, ErrPlanNotFound
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Plan.CreatedAt.Before(matches[j].Plan.CreatedAt)
	})
	return matches[len(matches)-1], nil
}

func (s *MemoryStore) SaveExecution(ctx context.Context, record ExecutionRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.executions[record.Execution.ID]; ok {
		if record.Events == nil {
			record.Events = existing.Events
		}
		if record.Audits == nil {
			record.Audits = existing.Audits
		}
	}
	s.executions[record.Execution.ID] = cloneExecution(record)
	return nil
}

func (s *MemoryStore) GetExecution(ctx context.Context, id string) (ExecutionRecord, error) {
	select {
	case <-ctx.Done():
		return ExecutionRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.executions[id]
	if !ok {
		return ExecutionRecord{}, ErrExecutionNotFound
	}
	return cloneExecution(record), nil
}

func (s *MemoryStore) ListExecutions(ctx context.Context, releaseID string) ([]ExecutionRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var records []ExecutionRecord
	for _, record := range s.executions {
		if releaseID == "" || record.Execution.ReleaseID == releaseID {
			records = append(records, cloneExecution(record))
		}
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Execution.CreatedAt.Before(records[j].Execution.CreatedAt)
	})
	return records, nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, executionID string, evt event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.executions[executionID]
	if !ok {
		return ErrExecutionNotFound
	}
	record.Events = append(record.Events, evt)
	s.executions[executionID] = record
	return nil
}

func (s *MemoryStore) Events(ctx context.Context, executionID string) ([]event.Event, error) {
	record, err := s.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	events := append([]event.Event(nil), record.Events...)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})
	return events, nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, executionID string, entry audit.AuditLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.executions[executionID]
	if !ok {
		return ErrExecutionNotFound
	}
	record.Audits = append(record.Audits, entry)
	s.executions[executionID] = record
	return nil
}

func (s *MemoryStore) Audits(ctx context.Context, executionID string) ([]audit.AuditLog, error) {
	record, err := s.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}
	audits := append([]audit.AuditLog(nil), record.Audits...)
	sort.Slice(audits, func(i, j int) bool {
		return audits[i].CreatedAt.Before(audits[j].CreatedAt)
	})
	return audits, nil
}

func clonePlan(record PlanRecord) PlanRecord {
	record.Plan.Targets = append([]environment.ReleaseTarget(nil), record.Plan.Targets...)
	record.Plan.ArtifactSummary = append([]string(nil), record.Plan.ArtifactSummary...)
	record.Plan.PolicyResults = append([]PolicyResult(nil), record.Plan.PolicyResults...)
	record.Plan.DeploymentPlans = append([]deploymentusecase.DeploymentPlan(nil), record.Plan.DeploymentPlans...)
	record.Plan.Ordering = append([]string(nil), record.Plan.Ordering...)
	record.Plan.Warnings = append([]string(nil), record.Plan.Warnings...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	record.Security.Events = append([]event.Event(nil), record.Security.Events...)
	record.Security.Audits = append([]audit.AuditLog(nil), record.Security.Audits...)
	return record
}

func cloneExecution(record ExecutionRecord) ExecutionRecord {
	record.Plan = clonePlan(PlanRecord{Plan: record.Plan}).Plan
	record.Execution.DeploymentRunIDs = append([]string(nil), record.Execution.DeploymentRunIDs...)
	record.Execution.Targets = append([]TargetExecution(nil), record.Execution.Targets...)
	record.Deployments = append([]deploymentusecase.RunRecord(nil), record.Deployments...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	record.Security.Events = append([]event.Event(nil), record.Security.Events...)
	record.Security.Audits = append([]audit.AuditLog(nil), record.Security.Audits...)
	return record
}
