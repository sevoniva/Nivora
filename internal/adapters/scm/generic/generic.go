package generic

import (
	"context"
	"errors"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

var ErrNotImplemented = errors.New("generic scm adapter is not implemented")

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) ValidateCredential(ctx context.Context, credential scm.CredentialRef) error {
	return ctx.Err()
}

func (p *Provider) GetRepository(ctx context.Context, repositoryID string) (scm.Repository, error) {
	return scm.Repository{}, ErrNotImplemented
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

func (p *Provider) CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status scm.CommitStatus) error {
	return ErrNotImplemented
}
