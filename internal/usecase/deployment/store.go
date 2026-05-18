package deployment

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
)

var (
	ErrRunNotFound = errors.New("deployment run not found")
	ErrRunTerminal = errors.New("deployment run is already terminal")
)

type Store interface {
	Save(ctx context.Context, record RunRecord) error
	Get(ctx context.Context, id string) (RunRecord, error)
	List(ctx context.Context) ([]RunRecord, error)
	AppendLog(ctx context.Context, runID string, log event.LogChunk) error
	Logs(ctx context.Context, runID string) ([]event.LogChunk, error)
	AppendEvent(ctx context.Context, runID string, evt event.Event) error
	Events(ctx context.Context, runID string) ([]event.Event, error)
	AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error
	Audits(ctx context.Context, subject string) ([]audit.AuditLog, error)
}

type MemoryStore struct {
	mu      sync.RWMutex
	runs    map[string]RunRecord
	nextSeq map[string]int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		runs:    make(map[string]RunRecord),
		nextSeq: make(map[string]int64),
	}
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

func (s *MemoryStore) List(ctx context.Context) ([]RunRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]RunRecord, 0, len(s.runs))
	for _, record := range s.runs {
		records = append(records, cloneRecord(record))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Run.CreatedAt.Before(records[j].Run.CreatedAt)
	})
	return records, nil
}

func (s *MemoryStore) AppendLog(ctx context.Context, runID string, log event.LogChunk) error {
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
	s.nextSeq[runID]++
	log.Sequence = s.nextSeq[runID]
	record.Logs = append(record.Logs, log)
	s.runs[runID] = record
	return nil
}

func (s *MemoryStore) Logs(ctx context.Context, runID string) ([]event.LogChunk, error) {
	record, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	logs := append([]event.LogChunk(nil), record.Logs...)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Sequence < logs[j].Sequence
	})
	return logs, nil
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

func (s *MemoryStore) Events(ctx context.Context, runID string) ([]event.Event, error) {
	record, err := s.Get(ctx, runID)
	if err != nil {
		return nil, err
	}
	events := append([]event.Event(nil), record.Events...)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})
	return events, nil
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

func (s *MemoryStore) Audits(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	var entries []audit.AuditLog
	for _, record := range records {
		for _, entry := range record.Audits {
			if entry.Subject == subject {
				entries = append(entries, entry)
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CreatedAt.Before(entries[j].CreatedAt)
	})
	return entries, nil
}

func cloneRecord(record RunRecord) RunRecord {
	record.Artifacts = append([]release.ReleaseArtifact(nil), record.Artifacts...)
	record.Steps = append([]domaindeployment.DeploymentStep(nil), record.Steps...)
	record.Plan.Resources = append([]ManifestResourceSummary(nil), record.Plan.Resources...)
	record.Plan.Artifacts = append([]string(nil), record.Plan.Artifacts...)
	record.Plan.Actions = append([]string(nil), record.Plan.Actions...)
	record.Plan.Warnings = append([]string(nil), record.Plan.Warnings...)
	record.Logs = append([]event.LogChunk(nil), record.Logs...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	record.Definition.Spec.Artifacts = append([]Artifact(nil), record.Definition.Spec.Artifacts...)
	record.Definition.Spec.Manifests = append([]string(nil), record.Definition.Spec.Manifests...)
	return record
}
