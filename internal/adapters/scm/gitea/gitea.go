// Package gitea provides a metadata-only Gitea SCM adapter skeleton.
//
// This package satisfies the scm.SCMProvider interface without contacting any
// Gitea instance. Credential validation is format-only, repository validation
// rejects inline credentials, and every network or write operation returns
// skeleton.ErrNotImplemented. Real Gitea API integration (clone, fetch, push,
// webhook normalization, commit status) is guarded future work and must not be
// required by tests or local development.
//
// Use New() to obtain a Provider; register repository metadata with the
// returned Provider's Register method for local GetRepository lookup.
// Production repository operations must route through a real adapter once
// implemented, not this skeleton.
package gitea

import (
	"github.com/sevoniva/nivora/internal/adapters/scm/skeleton"
)

// providerName matches the repository catalog Provider constant for Gitea.
const providerName = "gitea"

// Provider is a metadata-only Gitea SCMProvider skeleton.
type Provider = skeleton.Provider

// New returns a Gitea skeleton Provider.
func New() *Provider {
	return skeleton.New(providerName)
}
