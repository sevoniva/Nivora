package local

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkingTreeCommitInLocalGitRepo(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	runGitForTest(t, root, "init")
	runGitForTest(t, root, "config", "user.email", "nivora@example.invalid")
	runGitForTest(t, root, "config", "user.name", "Nivora Test")

	path := filepath.Join(root, "apps/demo/deployment.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("image: example/demo:old\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := New().Commit(ctx, root, "chore: update demo", []string{"apps/demo/deployment.yaml"})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if !result.Committed || result.Revision == "" {
		t.Fatalf("commit result = %#v", result)
	}
}

func TestWorkingTreePushRequiresAllowPush(t *testing.T) {
	_, err := New().Push(context.Background(), t.TempDir(), "origin", "main", false)
	if err == nil || !strings.Contains(err.Error(), "allowPush=true") {
		t.Fatalf("push guard error = %v", err)
	}
}

func TestWorkingTreeCheckoutRevisionRequiresConfirm(t *testing.T) {
	_, err := New().CheckoutRevision(context.Background(), t.TempDir(), "main", false)
	if err == nil || !strings.Contains(err.Error(), "confirm=true") {
		t.Fatalf("checkout guard error = %v", err)
	}
}

func runGitForTest(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, string(out))
	}
}
