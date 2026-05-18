package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func safePath(root string, path string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("working tree root is required")
	}
	cleanRoot, err := filepath.Abs(root)
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
