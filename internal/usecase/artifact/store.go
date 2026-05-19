package artifact

import (
	"context"
	"errors"
	"sort"
	"sync"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
)

var ErrReleaseNotFound = errors.New("release not found")

type Store interface {
	SaveRelease(ctx context.Context, record ReleaseRecord) error
	GetRelease(ctx context.Context, id string) (ReleaseRecord, error)
	ListReleases(ctx context.Context) ([]ReleaseRecord, error)
	AppendEvent(ctx context.Context, subject string, evt event.Event) error
	AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error
}

type MemoryStore struct {
	mu       sync.RWMutex
	releases map[string]ReleaseRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{releases: make(map[string]ReleaseRecord)}
}

func (s *MemoryStore) SaveRelease(ctx context.Context, record ReleaseRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releases[record.Release.ID] = cloneReleaseRecord(record)
	return nil
}

func (s *MemoryStore) GetRelease(ctx context.Context, id string) (ReleaseRecord, error) {
	select {
	case <-ctx.Done():
		return ReleaseRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.releases[id]
	if !ok {
		return ReleaseRecord{}, ErrReleaseNotFound
	}
	return cloneReleaseRecord(record), nil
}

func (s *MemoryStore) ListReleases(ctx context.Context) ([]ReleaseRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]ReleaseRecord, 0, len(s.releases))
	for _, record := range s.releases {
		records = append(records, cloneReleaseRecord(record))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Release.CreatedAt.Before(records[j].Release.CreatedAt)
	})
	return records, nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, subject string, evt event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.releases[subject]
	if !ok {
		return ErrReleaseNotFound
	}
	record.Events = append(record.Events, evt)
	s.releases[subject] = cloneReleaseRecord(record)
	return nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.releases[subject]
	if !ok {
		return ErrReleaseNotFound
	}
	record.Audits = append(record.Audits, entry)
	s.releases[subject] = cloneReleaseRecord(record)
	return nil
}

func cloneReleaseRecord(record ReleaseRecord) ReleaseRecord {
	record.Artifacts = append([]domainartifact.Artifact(nil), record.Artifacts...)
	record.Bindings = append([]release.ReleaseArtifact(nil), record.Bindings...)
	record.Inspections = append([]domainartifact.Inspection(nil), record.Inspections...)
	record.Resolutions = append([]domainartifact.Resolution(nil), record.Resolutions...)
	record.Warnings = append([]domainartifact.Warning(nil), record.Warnings...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	return record
}
