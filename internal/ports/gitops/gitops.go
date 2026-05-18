package gitops

import "context"

type FileChange struct {
	Path      string `json:"path"`
	Before    string `json:"before,omitempty"`
	After     string `json:"after,omitempty"`
	Diff      string `json:"diff,omitempty"`
	Changed   bool   `json:"changed"`
	Operation string `json:"operation,omitempty"`
	Warning   string `json:"warning,omitempty"`
}

type WorkingTree interface {
	ReadFile(ctx context.Context, root string, path string) (string, error)
	WriteFile(ctx context.Context, root string, path string, content string) error
	Diff(ctx context.Context, root string, path string, before string, after string) (string, error)
}
