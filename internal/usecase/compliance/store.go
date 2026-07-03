package compliance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	domaincompliance "github.com/sevoniva/nivora/internal/domain/compliance"
)

var (
	ErrEvidenceBundleNotFound  = errors.New("evidence bundle not found")
	ErrRetentionPolicyNotFound = errors.New("retention policy not found")
)

type Store interface {
	AppendAuditLog(ctx context.Context, entry audit.AuditLog) error
	SearchAuditLogs(ctx context.Context, input AuditSearchInput) ([]audit.AuditLog, error)
	SaveEvidenceBundle(ctx context.Context, bundle domaincompliance.EvidenceBundle) error
	GetEvidenceBundle(ctx context.Context, id string) (domaincompliance.EvidenceBundle, error)
	SearchEvidenceBundles(ctx context.Context, subjectType string, subjectID string) ([]domaincompliance.EvidenceBundle, error)
	SaveRetentionPolicy(ctx context.Context, policy domaincompliance.RetentionPolicy) error
	GetRetentionPolicy(ctx context.Context, scopeType string, scopeID string) (domaincompliance.RetentionPolicy, error)
	VerifyAuditChain(ctx context.Context, scopeType, scopeID string) (valid bool, firstBrokenID string, err error)
}

type MemoryStore struct {
	mu        sync.RWMutex
	audits    []audit.AuditLog
	evidence  map[string]domaincompliance.EvidenceBundle
	retention map[string]domaincompliance.RetentionPolicy
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		evidence:  make(map[string]domaincompliance.EvidenceBundle),
		retention: make(map[string]domaincompliance.RetentionPolicy),
	}
}

func (s *MemoryStore) AppendAuditLog(ctx context.Context, entry audit.AuditLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, entry)
	return nil
}

func (s *MemoryStore) SearchAuditLogs(ctx context.Context, input AuditSearchInput) ([]audit.AuditLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]audit.AuditLog, 0, len(s.audits))
	for _, entry := range s.audits {
		if !auditMatches(entry, input) {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *MemoryStore) SaveEvidenceBundle(ctx context.Context, bundle domaincompliance.EvidenceBundle) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evidence[bundle.ID] = cloneEvidenceBundle(bundle)
	return nil
}

func (s *MemoryStore) GetEvidenceBundle(ctx context.Context, id string) (domaincompliance.EvidenceBundle, error) {
	if err := ctx.Err(); err != nil {
		return domaincompliance.EvidenceBundle{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	bundle, ok := s.evidence[id]
	if !ok {
		return domaincompliance.EvidenceBundle{}, ErrEvidenceBundleNotFound
	}
	return cloneEvidenceBundle(bundle), nil
}

func (s *MemoryStore) SearchEvidenceBundles(ctx context.Context, subjectType string, subjectID string) ([]domaincompliance.EvidenceBundle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domaincompliance.EvidenceBundle, 0, len(s.evidence))
	for _, bundle := range s.evidence {
		if subjectType != "" && bundle.SubjectType != subjectType {
			continue
		}
		if subjectID != "" && bundle.SubjectID != subjectID {
			continue
		}
		out = append(out, cloneEvidenceBundle(bundle))
	}
	return out, nil
}

func (s *MemoryStore) SaveRetentionPolicy(ctx context.Context, policy domaincompliance.RetentionPolicy) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retention[retentionKey(policy.ScopeType, policy.ScopeID)] = policy
	return nil
}

func (s *MemoryStore) GetRetentionPolicy(ctx context.Context, scopeType string, scopeID string) (domaincompliance.RetentionPolicy, error) {
	if err := ctx.Err(); err != nil {
		return domaincompliance.RetentionPolicy{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.retention[retentionKey(scopeType, scopeID)]
	if !ok {
		return domaincompliance.RetentionPolicy{}, ErrRetentionPolicyNotFound
	}
	return policy, nil
}

func (s *MemoryStore) VerifyAuditChain(ctx context.Context, scopeType, scopeID string) (bool, string, error) {
	return false, "", errors.New("audit hash chain verification not supported with memory store")
}

func cloneEvidenceBundle(bundle domaincompliance.EvidenceBundle) domaincompliance.EvidenceBundle {
	body, err := json.Marshal(bundle)
	if err != nil {
		return bundle
	}
	var cloned domaincompliance.EvidenceBundle
	if err := json.Unmarshal(body, &cloned); err != nil {
		return bundle
	}
	return cloned
}

func auditMatches(entry audit.AuditLog, input AuditSearchInput) bool {
	if input.Subject != "" && !strings.Contains(entry.Subject, input.Subject) && !strings.Contains(entry.SubjectID, input.Subject) && !strings.Contains(entry.SubjectType, input.Subject) {
		return false
	}
	if input.ActorID != "" && entry.ActorID != input.ActorID {
		return false
	}
	if input.Action != "" && !strings.Contains(strings.ToLower(entry.Action), strings.ToLower(input.Action)) {
		return false
	}
	if input.ScopeType != "" && entry.ScopeType != input.ScopeType {
		return false
	}
	if input.ScopeID != "" && entry.ScopeID != input.ScopeID {
		return false
	}
	if input.CorrelationID != "" && entry.CorrelationID != input.CorrelationID {
		return false
	}
	return true
}
