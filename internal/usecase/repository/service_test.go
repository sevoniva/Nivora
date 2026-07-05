package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

func TestRepositoryRejectsInlineCredentials(t *testing.T) {
	service := NewService(NewMemoryStore(), fakeSCMProvider{})
	_, err := service.SaveRepository(context.Background(), Repository{
		ID:       "repo-secret-url",
		Name:     "secret-url",
		Provider: ProviderGenericGit,
		URL:      "https://user:password@example.invalid/team/app.git",
	})
	if err == nil {
		t.Fatal("expected inline credential URL to be rejected")
	}
	if strings.Contains(err.Error(), "password") {
		t.Fatalf("error leaked credential value: %v", err)
	}
}

func TestRepositorySnapshotAndIntelligenceForLocalRepo(t *testing.T) {
	repository := Repository{
		ID:            "repo-local",
		Name:          "local",
		Provider:      ProviderGenericGit,
		URL:           "https://example.invalid/team/repo.git",
		DefaultBranch: "main",
		ProjectID:     "project-a",
		CredentialRef: "credential-ref-placeholder",
	}
	service := NewService(NewMemoryStore(), fakeSCMProvider{tree: scm.Tree{
		RepositoryID: "repo-local",
		Ref:          "main",
		TreeHash:     "sha256:fake-tree",
		Files: []scm.FileInfo{
			{Path: "go.mod", Size: 32, Hash: "sha256:gomod"},
			{Path: "cmd/demo/main.go", Size: 28, Hash: "sha256:main"},
			{Path: "package.json", Size: 44, Hash: "sha256:package"},
			{Path: "vite.config.ts", Size: 18, Hash: "sha256:vite"},
			{Path: "src/App.tsx", Size: 37, Hash: "sha256:app"},
			{Path: "Dockerfile", Size: 13, Hash: "sha256:dockerfile"},
			{Path: "deploy/k8s/deployment.yaml", Size: 42, Hash: "sha256:k8s"},
			{Path: "charts/demo/Chart.yaml", Size: 28, Hash: "sha256:chart"},
			{Path: ".github/workflows/ci.yaml", Size: 9, Hash: "sha256:gha"},
			{Path: ".gitlab-ci.yml", Size: 14, Hash: "sha256:gitlab"},
			{Path: ".nivora/workflows/ci.yaml", Size: 15, Hash: "sha256:nivora"},
			{Path: ".env", Size: 25},
		},
		Warnings: []string{`secret-like environment file ".env" detected; values were not read`},
	}})
	saved, err := service.SaveRepository(context.Background(), repository)
	if err != nil {
		t.Fatalf("save repository: %v", err)
	}
	snapshot, err := service.CreateSnapshot(context.Background(), SnapshotInput{Repository: saved})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	if snapshot.TreeHash == "" {
		t.Fatal("expected tree hash")
	}
	assertContains(t, snapshot.DetectedLanguages, "Go")
	assertContains(t, snapshot.DetectedLanguages, "TypeScript")
	assertContains(t, snapshot.DetectedFrameworks, "React")
	assertContains(t, snapshot.DetectedFrameworks, "Vite")
	assertContains(t, snapshot.DetectedFrameworks, "Helm")
	assertContains(t, snapshot.DetectedFrameworks, "GitHub Actions workflow import signal")
	assertContains(t, snapshot.DetectedFrameworks, "GitLab CI")
	assertContains(t, snapshot.DetectedFrameworks, "Nivora Workflow")
	assertContains(t, snapshot.DetectedDeploymentFiles, "deploy/k8s/deployment.yaml")
	assertContains(t, snapshot.DetectedWorkflowFiles, ".nivora/workflows/ci.yaml")
	if len(snapshot.Warnings) == 0 || !strings.Contains(strings.Join(snapshot.Warnings, "\n"), ".env") {
		t.Fatalf("expected .env warning, got %#v", snapshot.Warnings)
	}

	intelligence, err := service.GetIntelligence(context.Background(), snapshot.RepositoryID, snapshot.ID)
	if err != nil {
		t.Fatalf("get intelligence: %v", err)
	}
	assertCommand(t, intelligence.BuildCommandCandidates, "go build ./...")
	assertCommand(t, intelligence.BuildCommandCandidates, "npm run build")
	assertCommand(t, intelligence.TestCommandCandidates, "go test ./...")
	assertCommand(t, intelligence.PackageCommandCandidates, "docker build -t <image> .")
	if !strings.Contains(intelligence.RecommendedNivoraWorkflowDraft, "kind: Workflow") {
		t.Fatalf("workflow draft missing kind: %s", intelligence.RecommendedNivoraWorkflowDraft)
	}
	if strings.Contains(intelligence.RecommendedNivoraWorkflowDraft, "TOKEN=should-not-be-read") {
		t.Fatal("workflow draft leaked environment value")
	}

	plan, err := service.DevOpsPlan(context.Background(), repository.ID)
	if err != nil {
		t.Fatalf("devops plan: %v", err)
	}
	if !plan.ReleaseReady {
		t.Fatal("expected plan to be release ready from detected build/package candidates")
	}
	if !plan.ReleaseCandidate.Eligible || len(plan.ReleaseCandidate.ArtifactCandidates) == 0 {
		t.Fatalf("release candidate plan missing artifact candidates: %#v", plan.ReleaseCandidate)
	}
	if len(plan.Security.Candidates) == 0 || !strings.Contains(strings.Join(plan.Security.Warnings, "\n"), "plan-only") {
		t.Fatalf("security plan missing candidates/warnings: %#v", plan.Security)
	}
	if !strings.Contains(strings.Join(plan.Build.Warnings, "\n"), "not executed") {
		t.Fatalf("build plan should be plan-only: %#v", plan.Build.Warnings)
	}
	if !strings.Contains(strings.Join(plan.Warnings, "\n"), "not executed") {
		t.Fatalf("expected plan-only warning, got %#v", plan.Warnings)
	}

	review, err := service.DevOpsReadinessReview(context.Background(), repository.ID)
	if err != nil {
		t.Fatalf("readiness review: %v", err)
	}
	if review.Status != "plan_ready" || !review.PlanOnly || !review.ReleaseReady {
		t.Fatalf("unexpected readiness review status: %#v", review)
	}
	if len(review.Strengths) == 0 || len(review.Blockers) != 0 {
		t.Fatalf("unexpected readiness review findings: strengths=%#v blockers=%#v", review.Strengths, review.Blockers)
	}
	if !strings.Contains(strings.Join(review.Warnings, "\n"), "does not execute") {
		t.Fatalf("readiness review should be plan-only: %#v", review.Warnings)
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, value := range values {
		if value == expected {
			return
		}
	}
	t.Fatalf("%q not found in %#v", expected, values)
}

type fakeSCMProvider struct {
	tree scm.Tree
}

func (p fakeSCMProvider) ValidateCredential(ctx context.Context, credential scm.CredentialRef) error {
	return ctx.Err()
}

func (p fakeSCMProvider) GetRepository(ctx context.Context, repositoryID string) (scm.Repository, error) {
	if err := ctx.Err(); err != nil {
		return scm.Repository{}, err
	}
	return scm.Repository{ID: repositoryID, URL: "https://example.invalid/repo.git", Provider: string(ProviderGenericGit)}, nil
}

func (p fakeSCMProvider) ValidateRepository(ctx context.Context, repository scm.Repository) error {
	return ctx.Err()
}

func (p fakeSCMProvider) ResolveRef(ctx context.Context, ref scm.RepositoryRef) (scm.Commit, error) {
	if err := ctx.Err(); err != nil {
		return scm.Commit{}, err
	}
	return scm.Commit{SHA: "sha256:fake-tree"}, nil
}

func (p fakeSCMProvider) ListBranches(ctx context.Context, repositoryID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{"main"}, nil
}

func (p fakeSCMProvider) ListTags(ctx context.Context, repositoryID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{}, nil
}

func (p fakeSCMProvider) GetCommit(ctx context.Context, repositoryID string, ref string) (scm.Commit, error) {
	return p.ResolveRef(ctx, scm.RepositoryRef{RepositoryID: repositoryID, Ref: ref})
}

func (p fakeSCMProvider) ReadFile(ctx context.Context, ref scm.RepositoryRef, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, errors.New("file not found")
}

func (p fakeSCMProvider) ListTree(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	return p.CreateSnapshot(ctx, ref)
}

func (p fakeSCMProvider) CreateSnapshot(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	if err := ctx.Err(); err != nil {
		return scm.Tree{}, err
	}
	tree := p.tree
	if tree.RepositoryID == "" {
		tree.RepositoryID = ref.RepositoryID
	}
	if tree.Ref == "" {
		tree.Ref = ref.Ref
	}
	return tree, nil
}

func (p fakeSCMProvider) DiffRefs(ctx context.Context, base scm.RepositoryRef, head scm.RepositoryRef) (scm.DiffSummary, error) {
	if err := ctx.Err(); err != nil {
		return scm.DiffSummary{}, err
	}
	return scm.DiffSummary{}, nil
}

func (p fakeSCMProvider) GetCapabilities(ctx context.Context) (scm.Capabilities, error) {
	if err := ctx.Err(); err != nil {
		return scm.Capabilities{}, err
	}
	return scm.Capabilities{Provider: "fake", ReadOnly: true}, nil
}

func (p fakeSCMProvider) CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status scm.CommitStatus) error {
	return ctx.Err()
}

func assertCommand(t *testing.T, values []CommandCandidate, expected string, optional ...bool) {
	t.Helper()
	for _, value := range values {
		if value.Command == expected {
			return
		}
	}
	if len(optional) > 0 && optional[0] {
		return
	}
	t.Fatalf("command %q not found in %#v", expected, values)
}
