// Package skeleton provides a shared metadata-only SCMProvider implementation
// used by the gitea, github, and gitlab provider skeletons. It satisfies the
// scm.SCMProvider interface without contacting any external service: credential
// validation is format-only, repository validation rejects inline credentials,
// and every network or write operation returns ErrNotImplemented.
//
// Real GitLab/Gitea/GitHub Enterprise API integration is guarded future work
// and must not be required by tests or local development.
package skeleton

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

// ErrNotImplemented is returned by every skeleton method that would require
// external network access or write semantics. Read-only metadata methods that
// can be answered locally (ValidateCredential, ValidateRepository,
// GetCapabilities) do not return this error.
var ErrNotImplemented = errors.New("scm skeleton adapter does not perform network or write operations; real provider integration is guarded future work")

// Provider is a metadata-only SCMProvider skeleton parameterized by provider
// name. It stores repositories registered through Register for local metadata
// lookup; it never clones, fetches, or writes.
type Provider struct {
	provider     string
	repositories map[string]scm.Repository
}

// New returns a skeleton Provider for the given provider name
// (e.g. "gitlab", "gitea", "github_enterprise").
func New(provider string) *Provider {
	return &Provider{provider: provider, repositories: map[string]scm.Repository{}}
}

// Register stores repository metadata for local lookup by GetRepository.
// It does not contact the external provider.
func (p *Provider) Register(repository scm.Repository) {
	if repository.ID == "" {
		return
	}
	p.repositories[repository.ID] = repository
}

func (p *Provider) ValidateCredential(ctx context.Context, credential scm.CredentialRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(credential.ID) == "" {
		return errors.New("credential id is required")
	}
	if strings.TrimSpace(credential.SecretKey) == "" {
		return errors.New("credential secret key is required; use CredentialRef metadata, never inline values")
	}
	return nil
}

func (p *Provider) GetRepository(ctx context.Context, repositoryID string) (scm.Repository, error) {
	if err := ctx.Err(); err != nil {
		return scm.Repository{}, err
	}
	repository, ok := p.repositories[strings.TrimSpace(repositoryID)]
	if !ok {
		return scm.Repository{}, fmt.Errorf("repository %q is not registered with the %s skeleton; live provider fetch is not implemented", repositoryID, p.provider)
	}
	return repository, nil
}

func (p *Provider) ValidateRepository(ctx context.Context, repository scm.Repository) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(repository.ID) == "" {
		return errors.New("repository id is required")
	}
	if strings.TrimSpace(repository.URL) == "" {
		return errors.New("repository url is required")
	}
	if hasInlineCredential(repository.URL) {
		return errors.New("repository url must not contain inline credentials; use CredentialRef")
	}
	return nil
}

func (p *Provider) ResolveRef(ctx context.Context, ref scm.RepositoryRef) (scm.Commit, error) {
	return scm.Commit{}, ErrNotImplemented
}

func (p *Provider) ListBranches(ctx context.Context, repositoryID string) ([]string, error) {
	return nil, ErrNotImplemented
}

func (p *Provider) ListTags(ctx context.Context, repositoryID string) ([]string, error) {
	return nil, ErrNotImplemented
}

func (p *Provider) GetCommit(ctx context.Context, repositoryID string, ref string) (scm.Commit, error) {
	return scm.Commit{}, ErrNotImplemented
}

func (p *Provider) ReadFile(ctx context.Context, ref scm.RepositoryRef, path string) ([]byte, error) {
	return nil, ErrNotImplemented
}

func (p *Provider) ListTree(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	return scm.Tree{}, ErrNotImplemented
}

func (p *Provider) CreateSnapshot(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	return scm.Tree{}, ErrNotImplemented
}

func (p *Provider) DiffRefs(ctx context.Context, base scm.RepositoryRef, head scm.RepositoryRef) (scm.DiffSummary, error) {
	return scm.DiffSummary{}, ErrNotImplemented
}

func (p *Provider) GetCapabilities(ctx context.Context) (scm.Capabilities, error) {
	if err := ctx.Err(); err != nil {
		return scm.Capabilities{}, err
	}
	return scm.Capabilities{
		Provider:           p.provider,
		ReadOnly:           true,
		RequiresCredential: true,
		SupportsWrite:      false,
		Warnings: []string{
			p.provider + " skeleton is metadata-only; live API clone/fetch/write integration is guarded future work",
			"do not route production repository operations through this skeleton",
		},
	}, nil
}

func (p *Provider) CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status scm.CommitStatus) error {
	return ErrNotImplemented
}

// hasInlineCredential rejects URLs with userinfo passwords such as
// https://user:token@host/path. CredentialRef must be used instead.
func hasInlineCredential(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if u.User == nil {
		return false
	}
	if _, ok := u.User.Password(); ok {
		return true
	}
	if u.User.Username() != "" {
		return true
	}
	return false
}
