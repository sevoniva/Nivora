package security

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
)

var ErrScanNotFound = errors.New("security scan not found")

type Store interface {
	Save(ctx context.Context, record ScanRecord) error
	Get(ctx context.Context, id string) (ScanRecord, error)
	AppendEvent(ctx context.Context, scanID string, evt event.Event) error
	Events(ctx context.Context, scanID string) ([]event.Event, error)
	AppendAudit(ctx context.Context, scanID string, entry audit.AuditLog) error
}

type MemoryStore struct {
	mu    sync.RWMutex
	scans map[string]ScanRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{scans: make(map[string]ScanRecord)}
}

func (s *MemoryStore) Save(ctx context.Context, record ScanRecord) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.scans[record.Scan.ID]; ok {
		if record.Events == nil {
			record.Events = existing.Events
		}
		if record.Audits == nil {
			record.Audits = existing.Audits
		}
	}
	s.scans[record.Scan.ID] = cloneRecord(record)
	return nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (ScanRecord, error) {
	select {
	case <-ctx.Done():
		return ScanRecord{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.scans[id]
	if !ok {
		return ScanRecord{}, ErrScanNotFound
	}
	return cloneRecord(record), nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, scanID string, evt event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.scans[scanID]
	if !ok {
		return ErrScanNotFound
	}
	record.Events = append(record.Events, evt)
	s.scans[scanID] = record
	return nil
}

func (s *MemoryStore) Events(ctx context.Context, scanID string) ([]event.Event, error) {
	record, err := s.Get(ctx, scanID)
	if err != nil {
		return nil, err
	}
	events := append([]event.Event(nil), record.Events...)
	sort.Slice(events, func(i, j int) bool { return events[i].Time.Before(events[j].Time) })
	return events, nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, scanID string, entry audit.AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.scans[scanID]
	if !ok {
		return ErrScanNotFound
	}
	record.Audits = append(record.Audits, entry)
	s.scans[scanID] = record
	return nil
}

func cloneRecord(record ScanRecord) ScanRecord {
	record.Scan.Findings = append([]domainsecurity.SecurityFinding(nil), record.Scan.Findings...)
	record.Policy.Findings = append([]domainsecurity.SecurityFinding(nil), record.Policy.Findings...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	record.Warnings = append([]string(nil), record.Warnings...)
	return record
}
