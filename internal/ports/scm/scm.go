package scm

import "context"

type CredentialRef struct {
	ID        string
	SecretKey string
}

type Repository struct {
	ID            string
	Name          string
	URL           string
	DefaultBranch string
}

type Commit struct {
	SHA     string
	Message string
	Author  string
}

type CommitStatus struct {
	State       string
	Description string
	TargetURL   string
}

type SCMProvider interface {
	ValidateCredential(ctx context.Context, credential CredentialRef) error
	GetRepository(ctx context.Context, repositoryID string) (Repository, error)
	ListBranches(ctx context.Context, repositoryID string) ([]string, error)
	ListTags(ctx context.Context, repositoryID string) ([]string, error)
	GetCommit(ctx context.Context, repositoryID string, ref string) (Commit, error)
	CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status CommitStatus) error
}
