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
	Provider      string
	DefaultBranch string
	CredentialRef string
	ProjectID     string
}

type Commit struct {
	SHA     string
	Message string
	Author  string
}

type RepositoryRef struct {
	RepositoryID string
	URL          string
	Provider     string
	Ref          string
	LocalPath    string
	Credential   CredentialRef
}

type FileInfo struct {
	Path string
	Size int64
	Hash string
}

type Tree struct {
	RepositoryID string
	Ref          string
	CommitSHA    string
	TreeHash     string
	Files        []FileInfo
	Warnings     []string
}

type DiffSummary struct {
	BaseRef      string
	HeadRef      string
	AddedFiles   []string
	RemovedFiles []string
	ChangedFiles []string
	Warnings     []string
}

type Capabilities struct {
	Provider             string
	ReadOnly             bool
	RequiresCredential   bool
	SupportsLocalPath    bool
	SupportsNetworkClone bool
	SupportsWrite        bool
	Warnings             []string
}

type CommitStatus struct {
	State       string
	Description string
	TargetURL   string
}

type SCMProvider interface {
	ValidateCredential(ctx context.Context, credential CredentialRef) error
	GetRepository(ctx context.Context, repositoryID string) (Repository, error)
	ValidateRepository(ctx context.Context, repository Repository) error
	ResolveRef(ctx context.Context, ref RepositoryRef) (Commit, error)
	ListBranches(ctx context.Context, repositoryID string) ([]string, error)
	ListTags(ctx context.Context, repositoryID string) ([]string, error)
	GetCommit(ctx context.Context, repositoryID string, ref string) (Commit, error)
	ReadFile(ctx context.Context, ref RepositoryRef, path string) ([]byte, error)
	ListTree(ctx context.Context, ref RepositoryRef) (Tree, error)
	CreateSnapshot(ctx context.Context, ref RepositoryRef) (Tree, error)
	DiffRefs(ctx context.Context, base RepositoryRef, head RepositoryRef) (DiffSummary, error)
	GetCapabilities(ctx context.Context) (Capabilities, error)
	CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status CommitStatus) error
}
