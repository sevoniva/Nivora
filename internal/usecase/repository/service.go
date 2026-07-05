package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

var ErrInvalid = errors.New("repository input is invalid")

type Service struct {
	store Store
	scm   scm.SCMProvider
	now   func() time.Time
}

func NewService(store Store, provider scm.SCMProvider) *Service {
	return &Service{store: store, scm: provider, now: time.Now}
}

func (s *Service) SaveRepository(ctx context.Context, repository Repository) (Repository, error) {
	if err := validateRepository(repository); err != nil {
		return Repository{}, err
	}
	now := s.now().UTC()
	if repository.ID == "" {
		repository.ID = defaultID("repo")
	}
	if repository.Status == "" {
		repository.Status = RepositoryStatusActive
	}
	if repository.CreatedAt.IsZero() {
		repository.CreatedAt = now
	}
	repository.UpdatedAt = now
	if err := s.store.SaveRepository(ctx, repository); err != nil {
		return Repository{}, err
	}
	return repository, nil
}

func (s *Service) GetRepository(ctx context.Context, id string) (Repository, error) {
	return s.store.GetRepository(ctx, strings.TrimSpace(id))
}

func (s *Service) ListRepositories(ctx context.Context, projectID string) ([]Repository, error) {
	return s.store.ListRepositories(ctx, strings.TrimSpace(projectID))
}

func (s *Service) CreateSnapshot(ctx context.Context, input SnapshotInput) (RepositorySnapshot, error) {
	repository := input.Repository
	if repository.ID == "" {
		var err error
		repository, err = s.store.GetRepository(ctx, strings.TrimSpace(input.Repository.ID))
		if err != nil {
			return RepositorySnapshot{}, err
		}
	}
	if err := validateRepository(repository); err != nil {
		return RepositorySnapshot{}, err
	}
	tree, err := s.scm.CreateSnapshot(ctx, scm.RepositoryRef{
		RepositoryID: repository.ID,
		URL:          repository.URL,
		Provider:     string(repository.Provider),
		Ref:          firstNonEmpty(input.Ref, repository.DefaultBranch),
		LocalPath:    input.LocalPath,
		Credential:   scm.CredentialRef{ID: repository.CredentialRef},
	})
	if err != nil {
		return RepositorySnapshot{}, err
	}
	now := s.now().UTC()
	snapshot := RepositorySnapshot{
		ID:           defaultID("repo-snapshot"),
		RepositoryID: repository.ID,
		Ref:          firstNonEmpty(input.Ref, repository.DefaultBranch, tree.Ref),
		CommitSHA:    tree.CommitSHA,
		TreeHash:     tree.TreeHash,
		Files:        toRepositoryFiles(tree.Files),
		Warnings:     append([]string(nil), tree.Warnings...),
		Metadata:     map[string]string{"provider": string(repository.Provider)},
		CreatedAt:    now,
	}
	applyDetection(&snapshot)
	if err := s.store.SaveSnapshot(ctx, snapshot); err != nil {
		return RepositorySnapshot{}, err
	}
	intelligence := AnalyzeSnapshot(snapshot, now)
	if err := s.store.SaveIntelligence(ctx, intelligence); err != nil {
		return RepositorySnapshot{}, err
	}
	return snapshot, nil
}

func (s *Service) AnalyzeLatest(ctx context.Context, repositoryID string) (RepositoryIntelligence, error) {
	snapshot, err := s.store.GetLatestSnapshot(ctx, strings.TrimSpace(repositoryID))
	if err != nil {
		return RepositoryIntelligence{}, err
	}
	intelligence := AnalyzeSnapshot(snapshot, s.now().UTC())
	if err := s.store.SaveIntelligence(ctx, intelligence); err != nil {
		return RepositoryIntelligence{}, err
	}
	return intelligence, nil
}

func (s *Service) AnalyzeSnapshot(ctx context.Context, snapshotID string) (RepositoryIntelligence, error) {
	snapshot, err := s.store.GetSnapshot(ctx, strings.TrimSpace(snapshotID))
	if err != nil {
		return RepositoryIntelligence{}, err
	}
	intelligence := AnalyzeSnapshot(snapshot, s.now().UTC())
	if err := s.store.SaveIntelligence(ctx, intelligence); err != nil {
		return RepositoryIntelligence{}, err
	}
	return intelligence, nil
}

func (s *Service) GetLatestSnapshot(ctx context.Context, repositoryID string) (RepositorySnapshot, error) {
	return s.store.GetLatestSnapshot(ctx, strings.TrimSpace(repositoryID))
}

func (s *Service) ListSnapshots(ctx context.Context, repositoryID string) ([]RepositorySnapshot, error) {
	return s.store.ListSnapshots(ctx, strings.TrimSpace(repositoryID))
}

func (s *Service) GetIntelligence(ctx context.Context, repositoryID string, snapshotID string) (RepositoryIntelligence, error) {
	return s.store.GetIntelligence(ctx, strings.TrimSpace(repositoryID), strings.TrimSpace(snapshotID))
}

func (s *Service) DevOpsPlan(ctx context.Context, repositoryID string) (DevOpsPlan, error) {
	snapshot, err := s.store.GetLatestSnapshot(ctx, strings.TrimSpace(repositoryID))
	if err != nil {
		return DevOpsPlan{}, err
	}
	intelligence := AnalyzeSnapshot(snapshot, s.now().UTC())
	now := s.now().UTC()
	plan := DevOpsPlan{
		RepositoryID:      snapshot.RepositoryID,
		SnapshotID:        snapshot.ID,
		Build:             BuildPlan{RepositoryID: snapshot.RepositoryID, SnapshotID: snapshot.ID, Commands: intelligence.BuildCommandCandidates, CreatedAt: now},
		Test:              TestPlan{RepositoryID: snapshot.RepositoryID, SnapshotID: snapshot.ID, Commands: intelligence.TestCommandCandidates, CreatedAt: now},
		Package:           PackagePlan{RepositoryID: snapshot.RepositoryID, SnapshotID: snapshot.ID, Commands: intelligence.PackageCommandCandidates, CreatedAt: now},
		SecurityScans:     append([]string(nil), intelligence.SecurityScanCandidates...),
		DeploymentTargets: append([]string(nil), intelligence.DeploymentTargetCandidates...),
		ReleaseReady:      len(intelligence.BuildCommandCandidates) > 0 || len(intelligence.PackageCommandCandidates) > 0,
		Warnings:          append([]string{"plan-only: detected commands are not executed by repository intelligence"}, intelligence.Warnings...),
		Metadata:          map[string]string{"source": "repository-intelligence"},
		CreatedAt:         now,
	}
	return plan, nil
}

func AnalyzeSnapshot(snapshot RepositorySnapshot, now time.Time) RepositoryIntelligence {
	copy := copySnapshot(snapshot)
	applyDetection(&copy)
	intelligence := RepositoryIntelligence{
		RepositoryID:                   copy.RepositoryID,
		SnapshotID:                     copy.ID,
		LanguageSummary:                copy.DetectedLanguages,
		FrameworkSummary:               copy.DetectedFrameworks,
		BuildCommandCandidates:         buildCandidates(copy),
		TestCommandCandidates:          testCandidates(copy),
		PackageCommandCandidates:       packageCandidates(copy),
		DeploymentTargetCandidates:     deploymentCandidates(copy),
		SecurityScanCandidates:         securityCandidates(copy),
		RecommendedNivoraWorkflowDraft: workflowDraft(copy),
		Warnings:                       append([]string(nil), copy.Warnings...),
		CreatedAt:                      now,
	}
	if len(intelligence.BuildCommandCandidates)+len(intelligence.TestCommandCandidates)+len(intelligence.PackageCommandCandidates) == 0 {
		intelligence.Warnings = append(intelligence.Warnings, "no build/test/package command candidates detected")
	}
	return intelligence
}

func validateRepository(repository Repository) error {
	if strings.TrimSpace(repository.ID) == "" {
		return fmt.Errorf("%w: repository id is required", ErrInvalid)
	}
	if strings.TrimSpace(repository.Name) == "" {
		return fmt.Errorf("%w: repository name is required", ErrInvalid)
	}
	if err := validateProvider(repository.Provider); err != nil {
		return err
	}
	if strings.TrimSpace(repository.URL) == "" {
		return fmt.Errorf("%w: repository url is required", ErrInvalid)
	}
	if hasInlineCredential(repository.URL) {
		return fmt.Errorf("%w: repository url must not contain inline credentials; use CredentialRef", ErrInvalid)
	}
	return nil
}

func validateProvider(provider Provider) error {
	switch provider {
	case ProviderGenericGit, ProviderGitHub, ProviderGitLab, ProviderGitea, ProviderLocal, ProviderArchive:
		return nil
	case "":
		return fmt.Errorf("%w: repository provider is required", ErrInvalid)
	default:
		return fmt.Errorf("%w: unsupported repository provider %q", ErrInvalid, provider)
	}
}

func hasInlineCredential(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return false
	}
	if parsed.User.Username() != "" {
		return true
	}
	_, ok := parsed.User.Password()
	return ok
}

func toRepositoryFiles(files []scm.FileInfo) []RepositoryFile {
	out := make([]RepositoryFile, 0, len(files))
	for _, file := range files {
		out = append(out, RepositoryFile{Path: file.Path, Size: file.Size, Hash: file.Hash})
	}
	return out
}

func defaultID(prefix string) string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func hasFile(snapshot RepositorySnapshot, path string) bool {
	path = filepath.ToSlash(path)
	for _, file := range snapshot.Files {
		if file.Path == path {
			return true
		}
	}
	return false
}

func hasPrefix(snapshot RepositorySnapshot, prefix string) bool {
	prefix = filepath.ToSlash(prefix)
	for _, file := range snapshot.Files {
		if strings.HasPrefix(file.Path, prefix) {
			return true
		}
	}
	return false
}

func hasSuffix(snapshot RepositorySnapshot, suffix string) bool {
	for _, file := range snapshot.Files {
		if strings.HasSuffix(file.Path, suffix) {
			return true
		}
	}
	return false
}
