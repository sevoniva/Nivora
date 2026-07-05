package gitlab

import (
	"testing"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

func TestGitLabProviderImplementsSCMProvider(t *testing.T) {
	var _ scm.SCMProvider = (*Provider)(nil)
}
