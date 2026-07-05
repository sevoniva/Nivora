package router

import (
	"context"
	"errors"
	"testing"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

// fakeProvider is a minimal SCMProvider that records its name on every call so
// tests can assert which adapter handled a given RepositoryRef.
type fakeProvider struct {
	name      string
	capsError bool
}

func (f fakeProvider) ValidateCredential(ctx context.Context, c scm.CredentialRef) error {
	return nil
}
func (f fakeProvider) GetRepository(ctx context.Context, id string) (scm.Repository, error) {
	return scm.Repository{ID: id, Provider: f.name}, nil
}
func (f fakeProvider) ValidateRepository(ctx context.Context, r scm.Repository) error {
	return nil
}
func (f fakeProvider) ResolveRef(ctx context.Context, ref scm.RepositoryRef) (scm.Commit, error) {
	return scm.Commit{SHA: f.name}, nil
}
func (f fakeProvider) ListBranches(ctx context.Context, id string) ([]string, error) {
	return []string{f.name}, nil
}
func (f fakeProvider) ListTags(ctx context.Context, id string) ([]string, error) {
	return nil, nil
}
func (f fakeProvider) GetCommit(ctx context.Context, id, ref string) (scm.Commit, error) {
	return scm.Commit{SHA: f.name}, nil
}
func (f fakeProvider) ReadFile(ctx context.Context, ref scm.RepositoryRef, path string) ([]byte, error) {
	return []byte(f.name), nil
}
func (f fakeProvider) ListTree(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	return scm.Tree{RepositoryID: ref.RepositoryID, Files: []scm.FileInfo{{Path: f.name}}}, nil
}
func (f fakeProvider) CreateSnapshot(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	return scm.Tree{RepositoryID: ref.RepositoryID, Files: []scm.FileInfo{{Path: f.name}}}, nil
}
func (f fakeProvider) DiffRefs(ctx context.Context, base, head scm.RepositoryRef) (scm.DiffSummary, error) {
	return scm.DiffSummary{BaseRef: f.name}, nil
}
func (f fakeProvider) GetCapabilities(ctx context.Context) (scm.Capabilities, error) {
	if f.capsError {
		return scm.Capabilities{}, errors.New("caps error")
	}
	return scm.Capabilities{Provider: f.name}, nil
}
func (f fakeProvider) CreateCommitStatus(ctx context.Context, id, sha string, status scm.CommitStatus) error {
	return nil
}

func TestRouterDispatchesByProvider(t *testing.T) {
	r := New()
	r.Register("generic_git", fakeProvider{name: "generic"})
	r.Register("gitlab", fakeProvider{name: "gitlab"})

	// ref.Provider=gitlab -> gitlab adapter
	commit, err := r.ResolveRef(context.Background(), scm.RepositoryRef{RepositoryID: "1", Provider: "gitlab"})
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if commit.SHA != "gitlab" {
		t.Fatalf("ResolveRef routed to %q, want gitlab", commit.SHA)
	}

	// ref.Provider=generic -> generic_git adapter (normalization)
	tree, err := r.ListTree(context.Background(), scm.RepositoryRef{RepositoryID: "1", Provider: "generic"})
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	if len(tree.Files) != 1 || tree.Files[0].Path != "generic" {
		t.Fatalf("ListTree routed to %v, want generic", tree.Files)
	}
}

func TestRouterUsesDefaultForProviderlessMethods(t *testing.T) {
	r := New(WithDefaultProvider("gitlab"))
	r.Register("generic_git", fakeProvider{name: "generic"})
	r.Register("gitlab", fakeProvider{name: "gitlab"})

	// GetRepository has no provider hint -> default (gitlab)
	repo, err := r.GetRepository(context.Background(), "1")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if repo.Provider != "gitlab" {
		t.Fatalf("GetRepository routed to %q, want gitlab", repo.Provider)
	}

	branches, err := r.ListBranches(context.Background(), "1")
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) != 1 || branches[0] != "gitlab" {
		t.Fatalf("ListBranches routed to %v, want gitlab", branches)
	}
}

func TestRouterErrorsOnUnknownProvider(t *testing.T) {
	r := New()
	r.Register("generic_git", fakeProvider{name: "generic"})
	_, err := r.ResolveRef(context.Background(), scm.RepositoryRef{RepositoryID: "1", Provider: "bitbucket"})
	if err == nil {
		t.Fatal("expected error for unregistered provider")
	}
}

func TestRouterImplementsSCMProvider(t *testing.T) {
	var _ scm.SCMProvider = (*Provider)(nil)
}
