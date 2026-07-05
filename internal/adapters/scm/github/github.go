// Package github provides a metadata-only GitHub Enterprise SCM adapter skeleton.
//
// This package satisfies the scm.SCMProvider interface without contacting any
// GitHub instance. Credential validation is format-only, repository validation
// rejects inline credentials, and every network or write operation returns
// skeleton.ErrNotImplemented. Real GitHub Enterprise API integration (clone,
// fetch, push, webhook normalization, commit status) is guarded future work and
// must not be required by tests or local development.
//
// Use New() to obtain a Provider; register repository metadata with the
// returned Provider's Register method for local GetRepository lookup.
// Production repository operations must route through a real adapter once
// implemented, not this skeleton.
package github

import (
	"github.com/sevoniva/nivora/internal/adapters/scm/skeleton"
)

// providerName matches the repository catalog Provider constant for GitHub.
const providerName = "github"

// Provider is a metadata-only GitHub Enterprise SCMProvider skeleton.
type Provider = skeleton.Provider

// New returns a GitHub Enterprise skeleton Provider.
func New() *Provider {
	return skeleton.New(providerName)
}
