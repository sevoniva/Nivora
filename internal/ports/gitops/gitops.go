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

type CommitResult struct {
	Message   string   `json:"message,omitempty"`
	Revision  string   `json:"revision,omitempty"`
	Files     []string `json:"files,omitempty"`
	Committed bool     `json:"committed"`
	Pushed    bool     `json:"pushed"`
	Warnings  []string `json:"warnings,omitempty"`
}

type WorkingTree interface {
	ReadFile(ctx context.Context, root string, path string) (string, error)
	WriteFile(ctx context.Context, root string, path string, content string) error
	Diff(ctx context.Context, root string, path string, before string, after string) (string, error)
	CurrentRevision(ctx context.Context, root string) (string, error)
	Commit(ctx context.Context, root string, message string, files []string) (CommitResult, error)
	Push(ctx context.Context, root string, remote string, branch string, allowPush bool) (CommitResult, error)
	CheckoutRevision(ctx context.Context, root string, revision string, confirm bool) (CommitResult, error)
}
