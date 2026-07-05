package generic

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

func TestListTreeDoesNotHashSecretLikeFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("TOKEN=should-not-be-read\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/generic\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	tree, err := New().ListTree(context.Background(), scm.RepositoryRef{RepositoryID: "repo", URL: "file://" + root})
	if err != nil {
		t.Fatalf("list tree: %v", err)
	}
	var envFound bool
	var goModHasHash bool
	for _, file := range tree.Files {
		switch file.Path {
		case ".env":
			envFound = true
			if file.Hash != "" {
				t.Fatalf(".env hash should be empty because content must not be read: %#v", file)
			}
		case "go.mod":
			goModHasHash = file.Hash != ""
		}
	}
	if !envFound {
		t.Fatalf(".env metadata should still be listed: %#v", tree.Files)
	}
	if !goModHasHash {
		t.Fatalf("non-sensitive go.mod should have a hash: %#v", tree.Files)
	}
	if len(tree.Warnings) == 0 {
		t.Fatalf("expected metadata-only warning for sensitive file")
	}
}

func TestDiffRefsComparesLocalTrees(t *testing.T) {
	base := t.TempDir()
	head := t.TempDir()
	writeTestFile(t, base, "go.mod", "module example.invalid/demo\n")
	writeTestFile(t, base, "README.md", "old\n")
	writeTestFile(t, head, "go.mod", "module example.invalid/demo\n")
	writeTestFile(t, head, "README.md", "new\n")
	writeTestFile(t, head, "cmd/main.go", "package main\n")

	diff, err := New().DiffRefs(context.Background(), scm.RepositoryRef{LocalPath: base}, scm.RepositoryRef{LocalPath: head})
	if err != nil {
		t.Fatalf("diff refs: %v", err)
	}
	assertContains(t, diff.AddedFiles, "cmd/main.go")
	assertContains(t, diff.ChangedFiles, "README.md")
	if len(diff.RemovedFiles) != 0 {
		t.Fatalf("removed files = %#v", diff.RemovedFiles)
	}
}

func writeTestFile(t *testing.T, root string, rel string, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
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
