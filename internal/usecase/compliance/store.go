package compliance

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

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
	PreviewRetention(ctx context.Context, policy domaincompliance.RetentionPolicy, now time.Time) ([]domaincompliance.RetentionTargetResult, error)
	ApplyRetention(ctx context.Context, policy domaincompliance.RetentionPolicy, now time.Time) ([]domaincompliance.RetentionTargetResult, error)
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

func (s *MemoryStore) PreviewRetention(ctx context.Context, policy domaincompliance.RetentionPolicy, now time.Time) ([]domaincompliance.RetentionTargetResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.retentionTargetsLocked(policy, now, false), nil
}

func (s *MemoryStore) ApplyRetention(ctx context.Context, policy domaincompliance.RetentionPolicy, now time.Time) ([]domaincompliance.RetentionTargetResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	targets := s.retentionTargetsLocked(policy, now, true)
	if len(targets) > 3 && targets[3].Deleted > 0 {
		cutoff := retentionCutoff(policy.EvidenceDays, now)
		for id, bundle := range s.evidence {
			if retentionScopeMatches(policy, bundle.ScopeType, bundle.ScopeID) && !cutoff.IsZero() && bundle.GeneratedAt.Before(cutoff) {
				delete(s.evidence, id)
			}
		}
	}
	return targets, nil
}

func (s *MemoryStore) VerifyAuditChain(ctx context.Context, scopeType, scopeID string) (bool, string, error) {
	return false, "", errors.New("audit hash chain verification not supported with memory store")
}

func (s *MemoryStore) retentionTargetsLocked(policy domaincompliance.RetentionPolicy, now time.Time, apply bool) []domaincompliance.RetentionTargetResult {
	logs := unsupportedRetentionTarget(domaincompliance.RetentionTargetLogs, policy.LogDays, now, "log retention spans runtime stores and is preview-only in this foundation")
	audit := supportedRetentionTarget(domaincompliance.RetentionTargetAudit, policy.AuditDays, now)
	audit.Immutable = true
	audit.Warnings = append(audit.Warnings, "audit records are immutable in this foundation; retention reports candidates but does not delete them")
	if cutoff := retentionCutoff(policy.AuditDays, now); !cutoff.IsZero() {
		for _, entry := range s.audits {
			if retentionScopeMatches(policy, entry.ScopeType, entry.ScopeID) && entry.CreatedAt.Before(cutoff) {
				audit.Candidates++
			}
		}
	}
	events := unsupportedRetentionTarget(domaincompliance.RetentionTargetEvents, policy.EventDays, now, "event retention spans runtime stores and is preview-only in this foundation")
	evidence := supportedRetentionTarget(domaincompliance.RetentionTargetEvidence, policy.EvidenceDays, now)
	if cutoff := retentionCutoff(policy.EvidenceDays, now); !cutoff.IsZero() {
		for _, bundle := range s.evidence {
			if retentionScopeMatches(policy, bundle.ScopeType, bundle.ScopeID) && bundle.GeneratedAt.Before(cutoff) {
				evidence.Candidates++
			}
		}
		if apply {
			evidence.Deleted = evidence.Candidates
		}
	}
	return []domaincompliance.RetentionTargetResult{logs, audit, events, evidence}
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
	if input.SubjectType != "" && entry.SubjectType != input.SubjectType {
		return false
	}
	if input.SubjectID != "" && entry.SubjectID != input.SubjectID {
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
	if input.RequestID != "" && entry.RequestID != input.RequestID {
		return false
	}
	if input.CorrelationID != "" && entry.CorrelationID != input.CorrelationID {
		return false
	}
	return true
}

func retentionScopeMatches(policy domaincompliance.RetentionPolicy, scopeType string, scopeID string) bool {
	if policy.ScopeType == "" || policy.ScopeType == "global" {
		return true
	}
	if scopeType != policy.ScopeType {
		return false
	}
	if policy.ScopeID == "" {
		return true
	}
	return scopeID == policy.ScopeID
}

func retentionCutoff(days int, now time.Time) time.Time {
	if days <= 0 {
		return time.Time{}
	}
	return now.AddDate(0, 0, -days)
}

func supportedRetentionTarget(target string, days int, now time.Time) domaincompliance.RetentionTargetResult {
	result := domaincompliance.RetentionTargetResult{Target: target, Supported: true, RetentionDays: days, Cutoff: retentionCutoff(days, now)}
	if days <= 0 {
		result.Warnings = append(result.Warnings, "retention is disabled for this target because retentionDays is zero")
	}
	return result
}

func unsupportedRetentionTarget(target string, days int, now time.Time, warning string) domaincompliance.RetentionTargetResult {
	result := supportedRetentionTarget(target, days, now)
	result.Supported = false
	result.Warnings = append(result.Warnings, warning)
	return result
}
