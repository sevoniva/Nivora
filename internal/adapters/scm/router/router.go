// Package router is a routing scm.SCMProvider that dispatches each call to a
// concrete adapter based on scm.RepositoryRef.Provider (or the repository's
// declared provider for methods that take only a repository id). It lets the
// repository service hold a single SCMProvider while still supporting generic,
// gitlab, gitea, and github adapters concurrently.
//
// Methods that do not carry a provider hint (GetRepository, ListBranches,
// ListTags, GetCommit, CreateCommitStatus) use a default provider configured
// at construction time; the gitlab adapter's default credential applies when
// the default provider is gitlab.
package router

import (
	"context"
	"fmt"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

// Provider is a routing SCMProvider.
type Provider struct {
	providers      map[string]scm.SCMProvider
	defaultProvider string
}

// Option configures a Provider.
type Option func(*Provider)

// WithDefaultProvider sets the provider used by methods that do not carry a
// provider hint (GetRepository, ListBranches, ListTags, GetCommit,
// CreateCommitStatus). Defaults to "generic_git".
func WithDefaultProvider(name string) Option {
	return func(p *Provider) {
		if name != "" {
			p.defaultProvider = name
		}
	}
}

// New returns a routing Provider. Register adapters with Register.
func New(opts ...Option) *Provider {
	p := &Provider{
		providers:       map[string]scm.SCMProvider{},
		defaultProvider: "generic_git",
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Register associates a concrete adapter with a provider name. Names should
// match the repository catalog Provider constants (generic_git, gitlab, gitea,
// github, local, archive).
func (p *Provider) Register(name string, adapter scm.SCMProvider) {
	if name == "" || adapter == nil {
		return
	}
	p.providers[name] = adapter
}

func (p *Provider) resolve(providerName string) (scm.SCMProvider, error) {
	if providerName == "" {
		providerName = p.defaultProvider
	}
	// "generic" is normalized to "generic_git" to match the catalog convention.
	if providerName == "generic" {
		providerName = "generic_git"
	}
	adapter, ok := p.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("scm router: no adapter registered for provider %q", providerName)
	}
	return adapter, nil
}

func (p *Provider) ValidateCredential(ctx context.Context, credential scm.CredentialRef) error {
	// Credential validation is provider-agnostic in this foundation; delegate to
	// the default provider so format checks still run.
	adapter, err := p.resolve(p.defaultProvider)
	if err != nil {
		return err
	}
	return adapter.ValidateCredential(ctx, credential)
}

func (p *Provider) GetRepository(ctx context.Context, repositoryID string) (scm.Repository, error) {
	adapter, err := p.resolve(p.defaultProvider)
	if err != nil {
		return scm.Repository{}, err
	}
	return adapter.GetRepository(ctx, repositoryID)
}

func (p *Provider) ValidateRepository(ctx context.Context, repository scm.Repository) error {
	adapter, err := p.resolve(string(repository.Provider))
	if err != nil {
		return err
	}
	return adapter.ValidateRepository(ctx, repository)
}

func (p *Provider) ResolveRef(ctx context.Context, ref scm.RepositoryRef) (scm.Commit, error) {
	adapter, err := p.resolve(ref.Provider)
	if err != nil {
		return scm.Commit{}, err
	}
	return adapter.ResolveRef(ctx, ref)
}

func (p *Provider) ListBranches(ctx context.Context, repositoryID string) ([]string, error) {
	adapter, err := p.resolve(p.defaultProvider)
	if err != nil {
		return nil, err
	}
	return adapter.ListBranches(ctx, repositoryID)
}

func (p *Provider) ListTags(ctx context.Context, repositoryID string) ([]string, error) {
	adapter, err := p.resolve(p.defaultProvider)
	if err != nil {
		return nil, err
	}
	return adapter.ListTags(ctx, repositoryID)
}

func (p *Provider) GetCommit(ctx context.Context, repositoryID string, ref string) (scm.Commit, error) {
	adapter, err := p.resolve(p.defaultProvider)
	if err != nil {
		return scm.Commit{}, err
	}
	return adapter.GetCommit(ctx, repositoryID, ref)
}

func (p *Provider) ReadFile(ctx context.Context, ref scm.RepositoryRef, path string) ([]byte, error) {
	adapter, err := p.resolve(ref.Provider)
	if err != nil {
		return nil, err
	}
	return adapter.ReadFile(ctx, ref, path)
}

func (p *Provider) ListTree(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	adapter, err := p.resolve(ref.Provider)
	if err != nil {
		return scm.Tree{}, err
	}
	return adapter.ListTree(ctx, ref)
}

func (p *Provider) CreateSnapshot(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	adapter, err := p.resolve(ref.Provider)
	if err != nil {
		return scm.Tree{}, err
	}
	return adapter.CreateSnapshot(ctx, ref)
}

func (p *Provider) DiffRefs(ctx context.Context, base scm.RepositoryRef, head scm.RepositoryRef) (scm.DiffSummary, error) {
	adapter, err := p.resolve(base.Provider)
	if err != nil {
		return scm.DiffSummary{}, err
	}
	return adapter.DiffRefs(ctx, base, head)
}

func (p *Provider) GetCapabilities(ctx context.Context) (scm.Capabilities, error) {
	adapter, err := p.resolve(p.defaultProvider)
	if err != nil {
		return scm.Capabilities{}, err
	}
	return adapter.GetCapabilities(ctx)
}

func (p *Provider) CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status scm.CommitStatus) error {
	adapter, err := p.resolve(p.defaultProvider)
	if err != nil {
		return err
	}
	return adapter.CreateCommitStatus(ctx, repositoryID, sha, status)
}
