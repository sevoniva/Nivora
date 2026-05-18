package local

import (
	"context"
	"strings"
	"testing"
)

func TestWorkingTreeReadWriteDiff(t *testing.T) {
	tree := New()
	root := t.TempDir()
	if err := tree.WriteFile(context.Background(), root, "apps/demo/deployment.yaml", "old\n"); err != nil {
		t.Fatalf("write: %v", err)
	}
	body, err := tree.ReadFile(context.Background(), root, "apps/demo/deployment.yaml")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	diff, err := tree.Diff(context.Background(), root, "apps/demo/deployment.yaml", body, "new\n")
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !strings.Contains(diff, "-old") || !strings.Contains(diff, "+new") {
		t.Fatalf("diff = %s", diff)
	}
}

func TestWorkingTreeRejectsEscapingPath(t *testing.T) {
	tree := New()
	if err := tree.WriteFile(context.Background(), t.TempDir(), "../escape.yaml", "bad"); err == nil {
		t.Fatal("expected unsafe path error")
	}
}
