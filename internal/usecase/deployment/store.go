package deployment

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/domain/release"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	portgitops "github.com/sevoniva/nivora/internal/ports/gitops"
)

var (
	ErrRunNotFound       = errors.New("deployment run not found")
	ErrRunTerminal       = errors.New("deployment run is already terminal")
	ErrHostGroupNotFound = errors.New("host group not found")
)

type Store interface {
	Save(ctx context.Context, record RunRecord) error
	Get(ctx context.Context, id string) (RunRecord, error)
	List(ctx context.Context) ([]RunRecord, error)
	SaveHostGroup(ctx context.Context, group HostGroup) error
	GetHostGroup(ctx context.Context, id string) (HostGroup, error)
	ListHostGroups(ctx context.Context) ([]HostGroup, error)
	AppendLog(ctx context.Context, runID string, log event.LogChunk) error
	Logs(ctx context.Context, runID string) ([]event.LogChunk, error)
	AppendEvent(ctx context.Context, runID string, evt event.Event) error
	Events(ctx context.Context, runID string) ([]event.Event, error)
	AppendAudit(ctx context.Context, runID string, entry audit.AuditLog) error
	Audits(ctx context.Context, subject string) ([]audit.AuditLog, error)
	ListNonTerminalDeploymentRuns(ctx context.Context, limit int) ([]RunRecord, error)
	ListStaleDeploymentRuns(ctx context.Context, olderThan time.Time, limit int) ([]RunRecord, error)
}

type MemoryStore struct {
	mu      sync.RWMutex
	runs    map[string]RunRecord
	groups  map[string]HostGroup
	nextSeq map[string]int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		runs:    make(map[string]RunRecord),
		groups:  make(map[string]HostGroup),
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

func (s *MemoryStore) SaveHostGroup(ctx context.Context, group HostGroup) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.groups[group.ID] = cloneHostGroup(group)
	return nil
}

func (s *MemoryStore) GetHostGroup(ctx context.Context, id string) (HostGroup, error) {
	select {
	case <-ctx.Done():
		return HostGroup{}, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	group, ok := s.groups[id]
	if !ok {
		return HostGroup{}, ErrHostGroupNotFound
	}
	return cloneHostGroup(group), nil
}

func (s *MemoryStore) ListHostGroups(ctx context.Context) ([]HostGroup, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	groups := make([]HostGroup, 0, len(s.groups))
	for _, group := range s.groups {
		groups = append(groups, cloneHostGroup(group))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].CreatedAt.Before(groups[j].CreatedAt) })
	return groups, nil
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

func (s *MemoryStore) ListNonTerminalDeploymentRuns(ctx context.Context, limit int) ([]RunRecord, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RunRecord, 0, len(records))
	for _, record := range records {
		if isTerminalDeploymentStatus(record.Run.Status) {
			continue
		}
		out = append(out, record)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *MemoryStore) ListStaleDeploymentRuns(ctx context.Context, olderThan time.Time, limit int) ([]RunRecord, error) {
	records, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RunRecord, 0, len(records))
	for _, record := range records {
		if isTerminalDeploymentStatus(record.Run.Status) {
			continue
		}
		stale := record.Run.UpdatedAt.Before(olderThan)
		if record.Run.LeaseExpiresAt != nil && record.Run.LeaseExpiresAt.Before(olderThan) {
			stale = true
		}
		if !stale {
			continue
		}
		out = append(out, record)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func cloneRecord(record RunRecord) RunRecord {
	record.Artifacts = append([]release.ReleaseArtifact(nil), record.Artifacts...)
	record.Steps = append([]domaindeployment.DeploymentStep(nil), record.Steps...)
	record.Plan.Resources = append([]ManifestResourceSummary(nil), record.Plan.Resources...)
	record.Plan.Artifacts = append([]string(nil), record.Plan.Artifacts...)
	record.Plan.ArtifactDetails = append([]domainartifact.Inspection(nil), record.Plan.ArtifactDetails...)
	record.Plan.ManifestImages = append([]ManifestImage(nil), record.Plan.ManifestImages...)
	record.Plan.Actions = append([]string(nil), record.Plan.Actions...)
	record.Plan.Warnings = append([]string(nil), record.Plan.Warnings...)
	record.GitOpsPlan.Files = append([]string(nil), record.GitOpsPlan.Files...)
	record.GitOpsPlan.FileChanges = append([]portgitops.FileChange(nil), record.GitOpsPlan.FileChanges...)
	record.GitOpsPlan.ArtifactChanges = append([]string(nil), record.GitOpsPlan.ArtifactChanges...)
	record.GitOpsPlan.ManifestValueChanges = append([]string(nil), record.GitOpsPlan.ManifestValueChanges...)
	record.GitOpsPlan.Warnings = append([]string(nil), record.GitOpsPlan.Warnings...)
	record.GitOpsDiff.Files = append([]portgitops.FileChange(nil), record.GitOpsDiff.Files...)
	record.GitOpsCommit.Files = append([]string(nil), record.GitOpsCommit.Files...)
	record.GitOpsCommit.Warnings = append([]string(nil), record.GitOpsCommit.Warnings...)
	record.GitOpsPush.Files = append([]string(nil), record.GitOpsPush.Files...)
	record.GitOpsPush.Warnings = append([]string(nil), record.GitOpsPush.Warnings...)
	record.GitOpsRollback.Files = append([]string(nil), record.GitOpsRollback.Files...)
	record.GitOpsRollback.Warnings = append([]string(nil), record.GitOpsRollback.Warnings...)
	record.HostPlan.Hosts = append([]HostDeploymentStep(nil), record.HostPlan.Hosts...)
	record.HostPlan.HealthChecks = append([]HostHealthCheck(nil), record.HostPlan.HealthChecks...)
	record.HostPlan.Actions = append([]string(nil), record.HostPlan.Actions...)
	record.HostPlan.Warnings = append([]string(nil), record.HostPlan.Warnings...)
	record.HostPlan.RollbackPlan.Resources = append([]ManifestResourceSummary(nil), record.HostPlan.RollbackPlan.Resources...)
	record.HostPlan.RollbackPlan.Warnings = append([]string(nil), record.HostPlan.RollbackPlan.Warnings...)
	record.HostDetails = append([]HostDeploymentRunDetail(nil), record.HostDetails...)
	record.ArgoCD.Resources = append([]portargocd.ResourceStatus(nil), record.ArgoCD.Resources...)
	record.ArgoCD.Warnings = append([]string(nil), record.ArgoCD.Warnings...)
	record.ArgoCD.Conditions = append([]string(nil), record.ArgoCD.Conditions...)
	record.ArgoCDResources = append([]portargocd.ResourceStatus(nil), record.ArgoCDResources...)
	record.ArgoCDWatch = append([]portargocd.ApplicationStatus(nil), record.ArgoCDWatch...)
	record.Inventory.Desired = append([]ManifestResourceSummary(nil), record.Inventory.Desired...)
	record.Inventory.Applied = append([]ManifestResourceSummary(nil), record.Inventory.Applied...)
	record.Inventory.Warnings = append([]string(nil), record.Inventory.Warnings...)
	record.Health.Resources = append([]ResourceHealthSummary(nil), record.Health.Resources...)
	record.Health.Warnings = append([]string(nil), record.Health.Warnings...)
	record.Diff.AddedResources = append([]string(nil), record.Diff.AddedResources...)
	record.Diff.RemovedResources = append([]string(nil), record.Diff.RemovedResources...)
	record.Diff.ChangedResources = append([]string(nil), record.Diff.ChangedResources...)
	record.Diff.Unchanged = append([]string(nil), record.Diff.Unchanged...)
	record.Diff.UnknownLiveState = append([]string(nil), record.Diff.UnknownLiveState...)
	record.Diff.Warnings = append([]string(nil), record.Diff.Warnings...)
	record.RollbackPlan.Resources = append([]ManifestResourceSummary(nil), record.RollbackPlan.Resources...)
	record.RollbackPlan.Warnings = append([]string(nil), record.RollbackPlan.Warnings...)
	record.DryRun.Resources = append([]ManifestResourceSummary(nil), record.DryRun.Resources...)
	record.DryRun.Warnings = append([]string(nil), record.DryRun.Warnings...)
	record.Apply.Resources = append([]ManifestResourceSummary(nil), record.Apply.Resources...)
	record.Apply.Warnings = append([]string(nil), record.Apply.Warnings...)
	record.Rollout.Resources = append([]ManifestResourceSummary(nil), record.Rollout.Resources...)
	record.Rollout.Warnings = append([]string(nil), record.Rollout.Warnings...)
	if record.Rollback != nil {
		rollback := *record.Rollback
		rollback.ResourceRefs = append([]string(nil), record.Rollback.ResourceRefs...)
		record.Rollback = &rollback
	}
	record.Logs = append([]event.LogChunk(nil), record.Logs...)
	record.Events = append([]event.Event(nil), record.Events...)
	record.Audits = append([]audit.AuditLog(nil), record.Audits...)
	record.Security.Scan.Findings = append([]domainsecurity.SecurityFinding(nil), record.Security.Scan.Findings...)
	record.Security.Policy.Findings = append([]domainsecurity.SecurityFinding(nil), record.Security.Policy.Findings...)
	record.Security.Events = append([]event.Event(nil), record.Security.Events...)
	record.Security.Audits = append([]audit.AuditLog(nil), record.Security.Audits...)
	record.Security.Warnings = append([]string(nil), record.Security.Warnings...)
	record.Definition.Spec.Artifacts = append([]Artifact(nil), record.Definition.Spec.Artifacts...)
	record.Definition.Spec.Host.Hosts = append([]Host(nil), record.Definition.Spec.Host.Hosts...)
	record.Definition.Spec.Manifests = append([]string(nil), record.Definition.Spec.Manifests...)
	return record
}

func cloneHostGroup(group HostGroup) HostGroup {
	group.Hosts = append([]HostTarget(nil), group.Hosts...)
	if group.Labels != nil {
		labels := make(map[string]string, len(group.Labels))
		for k, v := range group.Labels {
			labels[k] = v
		}
		group.Labels = labels
	}
	return group
}
