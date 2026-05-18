package pipeline

import (
	"context"
	"errors"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainpipeline "github.com/sevoniva/nivora/internal/domain/pipeline"
)

var ErrRunNotFound = errors.New("pipeline run not found")

type Store interface {
	Save(ctx context.Context, record RunRecord) error
	Get(ctx context.Context, id string) (RunRecord, error)
	AppendLog(ctx context.Context, runID string, log LogRecord) error
	AppendEvent(ctx context.Context, runID string, evt event.Event) error
	AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error
}

type MemoryStore struct {
	mu   sync.RWMutex
	runs map[string]RunRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{runs: make(map[string]RunRecord)}
}

func (s *MemoryStore) Save(ctx context.Context, record RunRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.runs[record.Run.ID]; ok {
		if record.Logs == nil {
			record.Logs = existing.Logs
		}
		if record.Events == nil {
			record.Events = existing.Events
		}
		if record.Audits == nil {
			record.Audits = existing.Audits
		}
	}
	s.runs[record.Run.ID] = cloneRecord(record)
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (RunRecord, error) {
	select {
	case <-ctx.Done():
		return RunRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.runs[id]
	if !ok {
		return RunRecord{}, ErrRunNotFound
	}
	return cloneRecord(record), nil
}

func (s *MemoryStore) AppendLog(ctx context.Context, runID string, log LogRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[runID]
	if !ok {
		return ErrRunNotFound
	}
	record.Logs = append(record.Logs, log)
	s.runs[runID] = record
	return nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, runID string, evt event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[runID]
	if !ok {
		return ErrRunNotFound
	}
	record.Events = append(record.Events, evt)
	s.runs[runID] = record
	return nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.runs[runID]
	if !ok {
		return ErrRunNotFound
	}
	record.Audits = append(record.Audits, entry)
	s.runs[runID] = record
	return nil
}

func cloneRecord(record RunRecord) RunRecord {
	record.Stages = append([]StageRecord(nil), record.Stages...)
	for i := range record.Stages {
		record.Stages[i].Jobs = append([]JobRecord(nil), record.Stages[i].Jobs...)
		for j := range record.Stages[i].Jobs {
			steps := record.Stages[i].Jobs[j].Steps
			record.Stages[i].Jobs[j].Steps = append([]domainpipeline.StepRun(nil), steps...)
		}
	}
	record.Logs = append([]LogRecord(nil), record.Logs...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	return record
}
