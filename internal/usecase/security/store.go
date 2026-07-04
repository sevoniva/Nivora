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

var (
	ErrScanNotFound         = errors.New("security scan not found")
	ErrFindingNotFound      = errors.New("security finding not found")
	ErrPolicyResultNotFound = errors.New("security policy result not found")
)

type Store interface {
	Save(ctx context.Context, record ScanRecord) error
	Get(ctx context.Context, id string) (ScanRecord, error)
	List(ctx context.Context) ([]ScanRecord, error)
	AppendEvent(ctx context.Context, scanID string, evt event.Event) error
	Events(ctx context.Context, scanID string) ([]event.Event, error)
	AppendAudit(ctx context.Context, scanID string, entry audit.AuditLog) error
	SavePolicyResult(ctx context.Context, result domainsecurity.PolicyResult) error
	GetPolicyResult(ctx context.Context, id string) (domainsecurity.PolicyResult, error)
	ListPolicyResults(ctx context.Context, input ListPolicyResultsInput) ([]domainsecurity.PolicyResult, error)
}

type MemoryStore struct {
	mu            sync.RWMutex
	scans         map[string]ScanRecord
	policyResults map[string]domainsecurity.PolicyResult
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{scans: make(map[string]ScanRecord), policyResults: make(map[string]domainsecurity.PolicyResult)}
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
	if record.Policy.ID != "" {
		s.policyResults[record.Policy.ID] = clonePolicyResult(record.Policy)
	}
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

func (s *MemoryStore) List(ctx context.Context) ([]ScanRecord, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]ScanRecord, 0, len(s.scans))
	for _, record := range s.scans {
		records = append(records, cloneRecord(record))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Scan.CreatedAt.Before(records[j].Scan.CreatedAt)
	})
	return records, nil
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

func (s *MemoryStore) SavePolicyResult(ctx context.Context, result domainsecurity.PolicyResult) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if result.ID == "" {
		return ErrPolicyResultNotFound
	}
	s.policyResults[result.ID] = clonePolicyResult(result)
	return nil
}

func (s *MemoryStore) GetPolicyResult(ctx context.Context, id string) (domainsecurity.PolicyResult, error) {
	select {
	case <-ctx.Done():
		return domainsecurity.PolicyResult{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result, ok := s.policyResults[id]
	if !ok {
		return domainsecurity.PolicyResult{}, ErrPolicyResultNotFound
	}
	return clonePolicyResult(result), nil
}

func (s *MemoryStore) ListPolicyResults(ctx context.Context, input ListPolicyResultsInput) ([]domainsecurity.PolicyResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]domainsecurity.PolicyResult, 0, len(s.policyResults))
	for _, result := range s.policyResults {
		if input.PolicyID != "" && result.PolicyID != input.PolicyID {
			continue
		}
		if input.SubjectType != "" && result.SubjectType != input.SubjectType {
			continue
		}
		if input.SubjectID != "" && result.SubjectID != input.SubjectID {
			continue
		}
		if input.ProjectID != "" && result.ProjectID != input.ProjectID {
			continue
		}
		if input.EnvironmentID != "" && result.EnvironmentID != input.EnvironmentID {
			continue
		}
		if input.Decision != "" && result.Decision != input.Decision {
			continue
		}
		results = append(results, clonePolicyResult(result))
	}
	sort.Slice(results, func(i, j int) bool { return results[i].EvaluatedAt.Before(results[j].EvaluatedAt) })
	return results, nil
}

func cloneRecord(record ScanRecord) ScanRecord {
	record.Scan.Findings = cloneFindings(record.Scan.Findings)
	record.Policy = clonePolicyResult(record.Policy)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	record.Warnings = append([]string(nil), record.Warnings...)
	return record
}

func clonePolicyResult(result domainsecurity.PolicyResult) domainsecurity.PolicyResult {
	result.Findings = cloneFindings(result.Findings)
	return result
}

func cloneFindings(findings []domainsecurity.SecurityFinding) []domainsecurity.SecurityFinding {
	out := append([]domainsecurity.SecurityFinding(nil), findings...)
	for i := range out {
		if out[i].Metadata == nil {
			continue
		}
		metadata := make(map[string]string, len(out[i].Metadata))
		for key, value := range out[i].Metadata {
			metadata[key] = value
		}
		out[i].Metadata = metadata
	}
	return out
}
