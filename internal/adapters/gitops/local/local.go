package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	gitops "github.com/sevoniva/nivora/internal/ports/gitops"
)

type WorkingTree struct{}

func New() WorkingTree {
	return WorkingTree{}
}

func (WorkingTree) ReadFile(ctx context.Context, root string, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	fullPath, err := safePath(root, path)
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (WorkingTree) WriteFile(ctx context.Context, root string, path string, content string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	fullPath, err := safePath(root, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0o644)
}

func (WorkingTree) Diff(ctx context.Context, root string, path string, before string, after string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if before == after {
		return "", nil
	}
	return fmt.Sprintf("--- %s\n+++ %s\n@@\n-%s\n+%s\n", path, path, strings.TrimSuffix(before, "\n"), strings.TrimSuffix(after, "\n")), nil
}

func (WorkingTree) CurrentRevision(ctx context.Context, root string) (string, error) {
	cleanRoot, err := safeRoot(root)
	if err != nil {
		return "", err
	}
	if !isGitRepository(cleanRoot) {
		return "working-tree-unversioned", nil
	}
	out, err := runGit(ctx, cleanRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (WorkingTree) Commit(ctx context.Context, root string, message string, files []string) (gitops.CommitResult, error) {
	cleanRoot, err := safeRoot(root)
	if err != nil {
		return gitops.CommitResult{}, err
	}
	if strings.TrimSpace(message) == "" {
		return gitops.CommitResult{}, fmt.Errorf("gitops commit message is required")
	}
	safeFiles, err := safeFiles(root, files)
	if err != nil {
		return gitops.CommitResult{}, err
	}
	if len(safeFiles) == 0 {
		return gitops.CommitResult{}, fmt.Errorf("gitops commit requires at least one file")
	}
	if !isGitRepository(cleanRoot) {
		revision, err := pseudoRevision(cleanRoot, message, safeFiles)
		if err != nil {
			return gitops.CommitResult{}, err
		}
		return gitops.CommitResult{
			Message:   message,
			Revision:  revision,
			Files:     safeFiles,
			Committed: true,
			Warnings:  []string{"working tree is not a Git repository; generated local pseudo revision"},
		}, nil
	}
	args := append([]string{"add", "--"}, safeFiles...)
	if _, err := runGit(ctx, cleanRoot, args...); err != nil {
		return gitops.CommitResult{}, err
	}
	if _, err := runGit(ctx, cleanRoot, "diff", "--cached", "--quiet"); err == nil {
		revision, revErr := WorkingTree{}.CurrentRevision(ctx, cleanRoot)
		if revErr != nil {
			return gitops.CommitResult{}, revErr
		}
		return gitops.CommitResult{
			Message:  message,
			Revision: revision,
			Files:    safeFiles,
			Warnings: []string{"no staged GitOps changes to commit"},
		}, nil
	}
	if _, err := runGit(ctx, cleanRoot, "-c", "commit.gpgsign=false", "commit", "-m", message); err != nil {
		return gitops.CommitResult{}, err
	}
	revision, err := WorkingTree{}.CurrentRevision(ctx, cleanRoot)
	if err != nil {
		return gitops.CommitResult{}, err
	}
	return gitops.CommitResult{Message: message, Revision: revision, Files: safeFiles, Committed: true}, nil
}

func (WorkingTree) Push(ctx context.Context, root string, remote string, branch string, allowPush bool) (gitops.CommitResult, error) {
	cleanRoot, err := safeRoot(root)
	if err != nil {
		return gitops.CommitResult{}, err
	}
	if !allowPush {
		return gitops.CommitResult{}, fmt.Errorf("gitops push requires allowPush=true")
	}
	if !isGitRepository(cleanRoot) {
		return gitops.CommitResult{}, fmt.Errorf("gitops push requires a Git repository")
	}
	if strings.TrimSpace(remote) == "" {
		remote = "origin"
	}
	if strings.TrimSpace(branch) == "" {
		branch = "HEAD"
	}
	if !safeGitRef(remote) || !safeGitRef(branch) {
		return gitops.CommitResult{}, fmt.Errorf("unsafe git remote or branch")
	}
	if _, err := runGit(ctx, cleanRoot, "push", remote, branch); err != nil {
		return gitops.CommitResult{}, err
	}
	revision, err := WorkingTree{}.CurrentRevision(ctx, cleanRoot)
	if err != nil {
		return gitops.CommitResult{}, err
	}
	return gitops.CommitResult{Revision: revision, Pushed: true}, nil
}

func (WorkingTree) CheckoutRevision(ctx context.Context, root string, revision string, confirm bool) (gitops.CommitResult, error) {
	cleanRoot, err := safeRoot(root)
	if err != nil {
		return gitops.CommitResult{}, err
	}
	if !confirm {
		return gitops.CommitResult{}, fmt.Errorf("gitops revision checkout requires confirm=true")
	}
	if strings.TrimSpace(revision) == "" {
		return gitops.CommitResult{}, fmt.Errorf("gitops rollback revision is required")
	}
	if !safeGitRef(revision) {
		return gitops.CommitResult{}, fmt.Errorf("unsafe git revision")
	}
	if !isGitRepository(cleanRoot) {
		return gitops.CommitResult{}, fmt.Errorf("gitops rollback by revision requires a Git repository")
	}
	if _, err := runGit(ctx, cleanRoot, "checkout", revision); err != nil {
		return gitops.CommitResult{}, err
	}
	current, err := WorkingTree{}.CurrentRevision(ctx, cleanRoot)
	if err != nil {
		return gitops.CommitResult{}, err
	}
	return gitops.CommitResult{Revision: current, Committed: false}, nil
}

func safePath(root string, path string) (string, error) {
	cleanRoot, err := safeRoot(root)
	if err != nil {
		return "", err
	}
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || cleanPath == "." || strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("unsafe gitops path %q", path)
	}
	fullPath := filepath.Join(cleanRoot, cleanPath)
	if !strings.HasPrefix(fullPath, cleanRoot+string(os.PathSeparator)) && fullPath != cleanRoot {
		return "", fmt.Errorf("gitops path escapes working tree")
	}
	return fullPath, nil
}

func safeRoot(root string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("working tree root is required")
	}
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return cleanRoot, nil
}

func safeFiles(root string, files []string) ([]string, error) {
	out := make([]string, 0, len(files))
	for _, file := range files {
		if _, err := safePath(root, file); err != nil {
			return nil, err
		}
		out = append(out, filepath.ToSlash(filepath.Clean(file)))
	}
	return out, nil
}

func isGitRepository(root string) bool {
	info, err := os.Stat(filepath.Join(root, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

func runGit(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func pseudoRevision(root string, message string, files []string) (string, error) {
	hash := sha256.New()
	_, _ = hash.Write([]byte(message))
	for _, file := range files {
		_, _ = hash.Write([]byte(file))
		body, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			return "", err
		}
		_, _ = hash.Write(body)
	}
	return "pseudo-" + hex.EncodeToString(hash.Sum(nil))[:16], nil
}

var safeGitRefPattern = regexp.MustCompile(`^[A-Za-z0-9._/\-]+$`)

func safeGitRef(value string) bool {
	return safeGitRefPattern.MatchString(value) && !strings.Contains(value, "..")
}
