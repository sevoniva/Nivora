package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
)

var (
	ErrNotFound      = errors.New("repository record not found")
	ErrAlreadyExists = errors.New("repository record already exists")
)

type Store interface {
	SaveRepository(ctx context.Context, repository Repository) error
	GetRepository(ctx context.Context, id string) (Repository, error)
	ListRepositories(ctx context.Context, projectID string) ([]Repository, error)
	SaveSnapshot(ctx context.Context, snapshot RepositorySnapshot) error
	GetSnapshot(ctx context.Context, id string) (RepositorySnapshot, error)
	GetLatestSnapshot(ctx context.Context, repositoryID string) (RepositorySnapshot, error)
	ListSnapshots(ctx context.Context, repositoryID string) ([]RepositorySnapshot, error)
	SaveIntelligence(ctx context.Context, intelligence RepositoryIntelligence) error
	GetIntelligence(ctx context.Context, repositoryID string, snapshotID string) (RepositoryIntelligence, error)
	AppendEvent(ctx context.Context, subject string, evt event.Event) error
	EventsBySubject(ctx context.Context, subject string) ([]event.Event, error)
	AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error
	AuditsBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error)
}

type MemoryStore struct {
	mu           sync.RWMutex
	repositories map[string]Repository
	snapshots    map[string]RepositorySnapshot
	intelligence map[string]RepositoryIntelligence
	events       map[string][]event.Event
	audits       map[string][]audit.AuditLog
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		repositories: map[string]Repository{},
		snapshots:    map[string]RepositorySnapshot{},
		intelligence: map[string]RepositoryIntelligence{},
		events:       map[string][]event.Event{},
		audits:       map[string][]audit.AuditLog{},
	}
}

func (s *MemoryStore) SaveRepository(ctx context.Context, repository Repository) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repositories[repository.ID] = copyRepository(repository)
	return nil
}

func (s *MemoryStore) GetRepository(ctx context.Context, id string) (Repository, error) {
	if err := ctx.Err(); err != nil {
		return Repository{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	repository, ok := s.repositories[id]
	if !ok {
		return Repository{}, ErrNotFound
	}
	return copyRepository(repository), nil
}

func (s *MemoryStore) ListRepositories(ctx context.Context, projectID string) ([]Repository, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Repository, 0, len(s.repositories))
	for _, repository := range s.repositories {
		if projectID == "" || repository.ProjectID == projectID {
			out = append(out, copyRepository(repository))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *MemoryStore) SaveSnapshot(ctx context.Context, snapshot RepositorySnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.snapshots[snapshot.ID] = copySnapshot(snapshot)
	return nil
}

func (s *MemoryStore) GetSnapshot(ctx context.Context, id string) (RepositorySnapshot, error) {
	if err := ctx.Err(); err != nil {
		return RepositorySnapshot{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot, ok := s.snapshots[id]
	if !ok {
		return RepositorySnapshot{}, ErrNotFound
	}
	return copySnapshot(snapshot), nil
}

func (s *MemoryStore) GetLatestSnapshot(ctx context.Context, repositoryID string) (RepositorySnapshot, error) {
	snapshots, err := s.ListSnapshots(ctx, repositoryID)
	if err != nil {
		return RepositorySnapshot{}, err
	}
	if len(snapshots) == 0 {
		return RepositorySnapshot{}, ErrNotFound
	}
	return snapshots[len(snapshots)-1], nil
}

func (s *MemoryStore) ListSnapshots(ctx context.Context, repositoryID string) ([]RepositorySnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []RepositorySnapshot
	for _, snapshot := range s.snapshots {
		if repositoryID == "" || snapshot.RepositoryID == repositoryID {
			out = append(out, copySnapshot(snapshot))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

func (s *MemoryStore) SaveIntelligence(ctx context.Context, intelligence RepositoryIntelligence) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.intelligence[intelligenceKey(intelligence.RepositoryID, intelligence.SnapshotID)] = copyIntelligence(intelligence)
	return nil
}

func (s *MemoryStore) GetIntelligence(ctx context.Context, repositoryID string, snapshotID string) (RepositoryIntelligence, error) {
	if err := ctx.Err(); err != nil {
		return RepositoryIntelligence{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	intelligence, ok := s.intelligence[intelligenceKey(repositoryID, snapshotID)]
	if !ok {
		return RepositoryIntelligence{}, ErrNotFound
	}
	return copyIntelligence(intelligence), nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, subject string, evt event.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events[subject] = append(s.events[subject], evt)
	return nil
}

func (s *MemoryStore) EventsBySubject(ctx context.Context, subject string) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := append([]event.Event(nil), s.events[subject]...)
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time.Equal(events[j].Time) {
			return events[i].ID < events[j].ID
		}
		return events[i].Time.Before(events[j].Time)
	})
	return events, nil
}

func (s *MemoryStore) AppendAudit(ctx context.Context, subject string, entry audit.AuditLog) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits[subject] = append(s.audits[subject], entry)
	return nil
}

func (s *MemoryStore) AuditsBySubject(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	audits := append([]audit.AuditLog(nil), s.audits[subject]...)
	sort.Slice(audits, func(i, j int) bool {
		if audits[i].CreatedAt.Equal(audits[j].CreatedAt) {
			return audits[i].ID < audits[j].ID
		}
		return audits[i].CreatedAt.Before(audits[j].CreatedAt)
	})
	return audits, nil
}

func intelligenceKey(repositoryID string, snapshotID string) string {
	return repositoryID + "\x00" + snapshotID
}

func copyRepository(in Repository) Repository {
	out := in
	out.Labels = copyMap(in.Labels)
	out.Metadata = copyMap(in.Metadata)
	return out
}

func copySnapshot(in RepositorySnapshot) RepositorySnapshot {
	out := in
	out.Files = append([]RepositoryFile(nil), in.Files...)
	out.DetectedLanguages = append([]string(nil), in.DetectedLanguages...)
	out.DetectedFrameworks = append([]string(nil), in.DetectedFrameworks...)
	out.DetectedBuildTools = append([]string(nil), in.DetectedBuildTools...)
	out.DetectedPackageManagers = append([]string(nil), in.DetectedPackageManagers...)
	out.DetectedDeploymentFiles = append([]string(nil), in.DetectedDeploymentFiles...)
	out.DetectedWorkflowFiles = append([]string(nil), in.DetectedWorkflowFiles...)
	out.DetectedSecurityFiles = append([]string(nil), in.DetectedSecurityFiles...)
	out.Warnings = append([]string(nil), in.Warnings...)
	out.Metadata = copyMap(in.Metadata)
	return out
}

func copyIntelligence(in RepositoryIntelligence) RepositoryIntelligence {
	out := in
	out.LanguageSummary = append([]string(nil), in.LanguageSummary...)
	out.FrameworkSummary = append([]string(nil), in.FrameworkSummary...)
	out.BuildCommandCandidates = append([]CommandCandidate(nil), in.BuildCommandCandidates...)
	out.TestCommandCandidates = append([]CommandCandidate(nil), in.TestCommandCandidates...)
	out.PackageCommandCandidates = append([]CommandCandidate(nil), in.PackageCommandCandidates...)
	out.DeploymentTargetCandidates = append([]string(nil), in.DeploymentTargetCandidates...)
	out.SecurityScanCandidates = append([]string(nil), in.SecurityScanCandidates...)
	out.Warnings = append([]string(nil), in.Warnings...)
	return out
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
