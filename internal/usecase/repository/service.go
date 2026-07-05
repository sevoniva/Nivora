package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/domain/audit"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/ports/scm"
)

var ErrInvalid = errors.New("repository input is invalid")

const (
	EventRepositoryCreated               = "devops.repository.created"
	EventRepositorySnapshotCreated       = "devops.repository.snapshot.created"
	EventRepositoryIntelligenceCompleted = "devops.repository.intelligence.completed"
	EventDevOpsPlanCreated               = "devops.devops.plan.created"
	EventDevOpsReadinessReviewCreated    = "devops.devops.readiness_review.created"
)

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
	if err := s.recordRepositoryEventAndAudit(ctx, repository.ID, EventRepositoryCreated, "repository created", repository.ProjectID, map[string]any{
		"repositoryId": repository.ID,
		"provider":     repository.Provider,
		"status":       repository.Status,
	}); err != nil {
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
	if err := s.recordRepositoryEventAndAudit(ctx, snapshot.ID, EventRepositorySnapshotCreated, "repository snapshot created", repository.ProjectID, map[string]any{
		"repositoryId": snapshot.RepositoryID,
		"snapshotId":   snapshot.ID,
		"ref":          snapshot.Ref,
		"fileCount":    len(snapshot.Files),
	}); err != nil {
		return RepositorySnapshot{}, err
	}
	intelligence := AnalyzeSnapshot(snapshot, now)
	if err := s.store.SaveIntelligence(ctx, intelligence); err != nil {
		return RepositorySnapshot{}, err
	}
	if err := s.recordRepositoryEventAndAudit(ctx, snapshot.ID, EventRepositoryIntelligenceCompleted, "repository intelligence completed", repository.ProjectID, map[string]any{
		"repositoryId": snapshot.RepositoryID,
		"snapshotId":   snapshot.ID,
		"languages":    len(intelligence.LanguageSummary),
		"buildPlans":   len(intelligence.BuildCommandCandidates),
		"testPlans":    len(intelligence.TestCommandCandidates),
	}); err != nil {
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
	projectID := ""
	if repository, err := s.store.GetRepository(ctx, snapshot.RepositoryID); err == nil {
		projectID = repository.ProjectID
	}
	if err := s.recordRepositoryEventAndAudit(ctx, snapshot.ID, EventRepositoryIntelligenceCompleted, "repository intelligence completed", projectID, map[string]any{
		"repositoryId": snapshot.RepositoryID,
		"snapshotId":   snapshot.ID,
		"languages":    len(intelligence.LanguageSummary),
		"buildPlans":   len(intelligence.BuildCommandCandidates),
		"testPlans":    len(intelligence.TestCommandCandidates),
	}); err != nil {
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
	projectID := ""
	if repository, err := s.store.GetRepository(ctx, snapshot.RepositoryID); err == nil {
		projectID = repository.ProjectID
	}
	if err := s.recordRepositoryEventAndAudit(ctx, snapshot.ID, EventRepositoryIntelligenceCompleted, "repository intelligence completed", projectID, map[string]any{
		"repositoryId": snapshot.RepositoryID,
		"snapshotId":   snapshot.ID,
		"languages":    len(intelligence.LanguageSummary),
		"buildPlans":   len(intelligence.BuildCommandCandidates),
		"testPlans":    len(intelligence.TestCommandCandidates),
	}); err != nil {
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
	build := BuildPlan{RepositoryID: snapshot.RepositoryID, SnapshotID: snapshot.ID, Commands: intelligence.BuildCommandCandidates, Warnings: commandPlanWarnings("build"), CreatedAt: now}
	test := TestPlan{RepositoryID: snapshot.RepositoryID, SnapshotID: snapshot.ID, Commands: intelligence.TestCommandCandidates, Warnings: commandPlanWarnings("test"), CreatedAt: now}
	pkg := PackagePlan{RepositoryID: snapshot.RepositoryID, SnapshotID: snapshot.ID, Commands: intelligence.PackageCommandCandidates, Warnings: commandPlanWarnings("package"), CreatedAt: now}
	security := SecurityScanPlan{RepositoryID: snapshot.RepositoryID, SnapshotID: snapshot.ID, Candidates: append([]string(nil), intelligence.SecurityScanCandidates...), Warnings: []string{"security scans are plan-only; no scanner is executed by repository planning"}, CreatedAt: now}
	releaseCandidate := releaseCandidatePlan(snapshot, intelligence, now)
	plan := DevOpsPlan{
		RepositoryID:      snapshot.RepositoryID,
		SnapshotID:        snapshot.ID,
		Build:             build,
		Test:              test,
		Package:           pkg,
		Security:          security,
		ReleaseCandidate:  releaseCandidate,
		SecurityScans:     append([]string(nil), intelligence.SecurityScanCandidates...),
		DeploymentTargets: append([]string(nil), intelligence.DeploymentTargetCandidates...),
		ReleaseReady:      releaseCandidate.Eligible,
		Warnings:          append([]string{"plan-only: detected commands are not executed by repository intelligence"}, intelligence.Warnings...),
		Metadata:          map[string]string{"source": "repository-intelligence"},
		CreatedAt:         now,
	}
	projectID := ""
	if repository, err := s.store.GetRepository(ctx, snapshot.RepositoryID); err == nil {
		projectID = repository.ProjectID
	}
	planID := defaultID("devops-plan")
	plan.PlanID = planID
	plan.ContentHash = devOpsPlanContentHash(plan)
	record := DevOpsPlanRecord{
		ID:           planID,
		RepositoryID: plan.RepositoryID,
		SnapshotID:   plan.SnapshotID,
		ProjectID:    projectID,
		ContentHash:  plan.ContentHash,
		Plan:         plan,
		CreatedAt:    now,
	}
	if err := s.store.SaveDevOpsPlan(ctx, record); err != nil {
		return DevOpsPlan{}, err
	}
	if err := s.recordRepositoryEventAndAudit(ctx, snapshot.ID, EventDevOpsPlanCreated, "repository DevOps plan created", projectID, map[string]any{
		"repositoryId": snapshot.RepositoryID,
		"snapshotId":   snapshot.ID,
		"planId":       planID,
		"contentHash":  plan.ContentHash,
		"planOnly":     true,
		"releaseReady": plan.ReleaseReady,
	}); err != nil {
		return DevOpsPlan{}, err
	}
	return plan, nil
}

func (s *Service) GetDevOpsPlan(ctx context.Context, id string) (DevOpsPlanRecord, error) {
	return s.store.GetDevOpsPlan(ctx, strings.TrimSpace(id))
}

func (s *Service) GetLatestDevOpsPlan(ctx context.Context, repositoryID string) (DevOpsPlanRecord, error) {
	return s.store.GetLatestDevOpsPlan(ctx, strings.TrimSpace(repositoryID))
}

func (s *Service) ListDevOpsPlans(ctx context.Context, repositoryID string) ([]DevOpsPlanRecord, error) {
	return s.store.ListDevOpsPlans(ctx, strings.TrimSpace(repositoryID))
}

func (s *Service) DevOpsReadinessReview(ctx context.Context, repositoryID string) (DevOpsReadinessReview, error) {
	plan, err := s.DevOpsPlan(ctx, repositoryID)
	if err != nil {
		return DevOpsReadinessReview{}, err
	}
	now := s.now().UTC()
	review := DevOpsReadinessReview{
		RepositoryID:           plan.RepositoryID,
		SnapshotID:             plan.SnapshotID,
		PlanOnly:               true,
		ReleaseReady:           plan.ReleaseReady,
		BuildPlanAvailable:     len(plan.Build.Commands) > 0,
		TestPlanAvailable:      len(plan.Test.Commands) > 0,
		PackagePlanAvailable:   len(plan.Package.Commands) > 0,
		SecurityPlanAvailable:  len(plan.Security.Candidates) > 0,
		DeploymentTargets:      append([]string(nil), plan.DeploymentTargets...),
		Warnings:               append([]string{"readiness review is plan-only and does not execute repository commands, create releases, trigger scanners, or deploy"}, plan.Warnings...),
		Metadata:               map[string]string{"source": "repository-intelligence", "devopsPlanId": plan.PlanID, "devopsPlanContentHash": plan.ContentHash, "devopsPlanSnapshotId": plan.SnapshotID},
		CreatedAt:              now,
		RecommendedNextActions: []string{"Review generated Nivora Workflow draft before enabling guarded execution"},
	}
	if review.BuildPlanAvailable {
		review.Strengths = append(review.Strengths, "build command candidates detected")
	} else {
		review.Blockers = append(review.Blockers, "no build command candidates detected")
		review.RecommendedNextActions = append(review.RecommendedNextActions, "Add an explicit Nivora Workflow build job or repository build metadata")
	}
	if review.TestPlanAvailable {
		review.Strengths = append(review.Strengths, "test command candidates detected")
	} else {
		review.Blockers = append(review.Blockers, "no test command candidates detected")
		review.RecommendedNextActions = append(review.RecommendedNextActions, "Add an explicit Nivora Workflow test job before release automation")
	}
	if review.PackagePlanAvailable || len(plan.ReleaseCandidate.ArtifactCandidates) > 0 {
		review.Strengths = append(review.Strengths, "artifact or package candidates detected")
	} else {
		review.Blockers = append(review.Blockers, "no artifact or package candidates detected")
		review.RecommendedNextActions = append(review.RecommendedNextActions, "Define package output metadata before binding ReleaseArtifacts")
	}
	if review.SecurityPlanAvailable {
		review.Strengths = append(review.Strengths, "security scan candidates detected")
	} else {
		review.Warnings = append(review.Warnings, "no security scan candidates detected")
		review.RecommendedNextActions = append(review.RecommendedNextActions, "Add a security scan intent before promotion gates")
	}
	if len(review.DeploymentTargets) > 0 {
		review.Strengths = append(review.Strengths, "deployment target candidates detected")
	} else {
		review.Warnings = append(review.Warnings, "no deployment target candidates detected")
		review.RecommendedNextActions = append(review.RecommendedNextActions, "Add deployment intent only after artifact identity and policy gates are clear")
	}
	review.Strengths = dedupeSorted(review.Strengths)
	review.Blockers = dedupeSorted(review.Blockers)
	review.Warnings = dedupeSorted(review.Warnings)
	review.RecommendedNextActions = dedupeSorted(review.RecommendedNextActions)
	switch {
	case len(review.Blockers) == 0 && review.ReleaseReady:
		review.Status = "plan_ready"
	case len(review.Blockers) == 0:
		review.Status = "needs_review"
	default:
		review.Status = "blocked"
	}
	projectID := ""
	if repository, err := s.store.GetRepository(ctx, plan.RepositoryID); err == nil {
		projectID = repository.ProjectID
	}
	if err := s.recordRepositoryEventAndAudit(ctx, plan.SnapshotID, EventDevOpsReadinessReviewCreated, "repository readiness review created", projectID, map[string]any{
		"repositoryId": review.RepositoryID,
		"snapshotId":   review.SnapshotID,
		"planId":       plan.PlanID,
		"contentHash":  plan.ContentHash,
		"status":       review.Status,
		"planOnly":     review.PlanOnly,
	}); err != nil {
		return DevOpsReadinessReview{}, err
	}
	return review, nil
}

func devOpsPlanContentHash(plan DevOpsPlan) string {
	stable := copyDevOpsPlan(plan)
	stable.PlanID = ""
	stable.ContentHash = ""
	stable.CreatedAt = time.Time{}
	stable.Build.CreatedAt = time.Time{}
	stable.Test.CreatedAt = time.Time{}
	stable.Package.CreatedAt = time.Time{}
	stable.Security.CreatedAt = time.Time{}
	stable.ReleaseCandidate.CreatedAt = time.Time{}
	body, _ := json.Marshal(stable)
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (s *Service) Events(ctx context.Context, subject string) ([]event.Event, error) {
	return s.store.EventsBySubject(ctx, strings.TrimSpace(subject))
}

func (s *Service) Audits(ctx context.Context, subject string) ([]audit.AuditLog, error) {
	return s.store.AuditsBySubject(ctx, strings.TrimSpace(subject))
}

func (s *Service) recordRepositoryEventAndAudit(ctx context.Context, subject string, eventType string, action string, projectID string, data map[string]any) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		subject = strings.TrimSpace(valueFromData(data, "repositoryId"))
	}
	if subject == "" {
		return nil
	}
	now := s.now().UTC()
	payload := cloneEventData(data)
	payload["message"] = action
	evt := event.Event{
		SpecVersion:     "1.0",
		ID:              defaultID("evt"),
		Type:            eventType,
		Source:          "nivora/repository",
		Subject:         subject,
		Time:            now,
		DataContentType: "application/json",
		Data:            payload,
	}
	if err := s.store.AppendEvent(ctx, subject, evt); err != nil {
		return err
	}
	return s.store.AppendAudit(ctx, subject, audit.AuditLog{
		ID:          defaultID("audit"),
		Action:      action,
		Subject:     subject,
		SubjectType: "repository",
		SubjectID:   subject,
		ScopeType:   "project",
		ScopeID:     strings.TrimSpace(projectID),
		Metadata:    auditMetadata(payload),
		CreatedAt:   now,
	})
}

func cloneEventData(data map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range data {
		if strings.TrimSpace(key) != "" {
			out[key] = value
		}
	}
	return out
}

func valueFromData(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return value
}

func auditMetadata(data map[string]any) map[string]string {
	if len(data) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range data {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch typed := value.(type) {
		case string:
			if typed != "" {
				out[key] = typed
			}
		case fmt.Stringer:
			out[key] = typed.String()
		case bool:
			out[key] = fmt.Sprintf("%t", typed)
		case int:
			out[key] = fmt.Sprintf("%d", typed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func commandPlanWarnings(kind string) []string {
	return []string{kind + " commands are detection candidates and are not executed by repository planning"}
}

func releaseCandidatePlan(snapshot RepositorySnapshot, intelligence RepositoryIntelligence, now time.Time) ReleaseCandidatePlan {
	artifacts := []string{}
	for _, candidate := range intelligence.PackageCommandCandidates {
		artifacts = append(artifacts, candidate.Name)
	}
	if len(artifacts) == 0 && len(intelligence.BuildCommandCandidates) > 0 {
		artifacts = append(artifacts, "build-output")
	}
	requiredChecks := []string{}
	if len(intelligence.TestCommandCandidates) > 0 {
		requiredChecks = append(requiredChecks, "tests")
	}
	if len(intelligence.SecurityScanCandidates) > 0 {
		requiredChecks = append(requiredChecks, "security-scans")
	}
	warnings := []string{"release candidate plan is metadata-only; no Release or ReleaseArtifact is created"}
	if len(artifacts) == 0 {
		warnings = append(warnings, "no artifact-producing build or package candidate was detected")
	}
	return ReleaseCandidatePlan{
		RepositoryID:       snapshot.RepositoryID,
		SnapshotID:         snapshot.ID,
		Eligible:           len(artifacts) > 0,
		ArtifactCandidates: dedupeSorted(artifacts),
		RequiredChecks:     dedupeSorted(requiredChecks),
		Warnings:           dedupeSorted(warnings),
		CreatedAt:          now,
	}
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
