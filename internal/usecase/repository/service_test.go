package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	scmgeneric "github.com/sevoniva/nivora/internal/adapters/scm/generic"
	"github.com/sevoniva/nivora/internal/ports/scm"
)

func TestRepositoryRejectsInlineCredentials(t *testing.T) {
	service := NewService(NewMemoryStore(), scmgeneric.New())
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
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/demo\n\ngo 1.22\n")
	writeFile(t, root, "cmd/demo/main.go", "package main\nfunc main(){}\n")
	writeFile(t, root, "package.json", `{"scripts":{"build":"vite build","test":"vitest"}}`)
	writeFile(t, root, "vite.config.ts", "export default {}\n")
	writeFile(t, root, "src/App.tsx", "export function App(){ return null }\n")
	writeFile(t, root, "Dockerfile", "FROM scratch\n")
	writeFile(t, root, "deploy/k8s/deployment.yaml", "apiVersion: apps/v1\nkind: Deployment\n")
	writeFile(t, root, "charts/demo/Chart.yaml", "apiVersion: v2\nname: demo\n")
	writeFile(t, root, ".github/workflows/ci.yaml", "name: ci\n")
	writeFile(t, root, ".gitlab-ci.yml", "stages: [test]\n")
	writeFile(t, root, ".nivora/workflows/ci.yaml", "kind: Workflow\n")
	writeFile(t, root, ".env", "TOKEN=should-not-be-read\n")

	repository := Repository{
		ID:            "repo-local",
		Name:          "local",
		Provider:      ProviderLocal,
		URL:           root,
		DefaultBranch: "main",
		ProjectID:     "project-a",
		CredentialRef: "credential-ref-placeholder",
	}
	service := NewService(NewMemoryStore(), scmgeneric.NewWithRepositories(nil))
	saved, err := service.SaveRepository(context.Background(), repository)
	if err != nil {
		t.Fatalf("save repository: %v", err)
	}
	snapshot, err := service.CreateSnapshot(context.Background(), SnapshotInput{Repository: saved, LocalPath: root})
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
	if !strings.Contains(strings.Join(plan.Warnings, "\n"), "not executed") {
		t.Fatalf("expected plan-only warning, got %#v", plan.Warnings)
	}
}

func TestGenericProviderDiffsLocalTrees(t *testing.T) {
	base := t.TempDir()
	head := t.TempDir()
	writeFile(t, base, "go.mod", "module example.com/demo\n")
	writeFile(t, base, "README.md", "old\n")
	writeFile(t, head, "go.mod", "module example.com/demo\n")
	writeFile(t, head, "README.md", "new\n")
	writeFile(t, head, "cmd/main.go", "package main\n")

	provider := scmgeneric.New()
	diff, err := provider.DiffRefs(context.Background(), localRef(base), localRef(head))
	if err != nil {
		t.Fatalf("diff refs: %v", err)
	}
	assertContains(t, diff.AddedFiles, "cmd/main.go")
	assertContains(t, diff.ChangedFiles, "README.md")
	if len(diff.RemovedFiles) != 0 {
		t.Fatalf("removed files = %#v", diff.RemovedFiles)
	}
}

func localRef(path string) scm.RepositoryRef {
	return scm.RepositoryRef{LocalPath: path}
}

func writeFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
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
