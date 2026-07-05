package skeleton

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

func TestSkeletonCapabilitiesAreMetadataOnly(t *testing.T) {
	for _, name := range []string{"gitlab", "gitea", "github"} {
		caps, err := New(name).GetCapabilities(context.Background())
		if err != nil {
			t.Fatalf("%s capabilities: %v", name, err)
		}
		if caps.Provider != name {
			t.Fatalf("%s capabilities provider = %q, want %q", name, caps.Provider, name)
		}
		if !caps.ReadOnly {
			t.Fatalf("%s skeleton must be read-only", name)
		}
		if caps.SupportsWrite {
			t.Fatalf("%s skeleton must not support write", name)
		}
		if !caps.RequiresCredential {
			t.Fatalf("%s skeleton must declare credential requirement for future integration", name)
		}
		if len(caps.Warnings) == 0 {
			t.Fatalf("%s skeleton must warn that it is metadata-only", name)
		}
	}
}

func TestSkeletonValidateCredentialIsFormatOnly(t *testing.T) {
	p := New("gitlab")
	if err := p.ValidateCredential(context.Background(), scm.CredentialRef{}); err == nil {
		t.Fatal("expected error for empty credential id")
	}
	if err := p.ValidateCredential(context.Background(), scm.CredentialRef{ID: "cred-1"}); err == nil {
		t.Fatal("expected error for missing secret key")
	}
	if err := p.ValidateCredential(context.Background(), scm.CredentialRef{ID: "cred-1", SecretKey: "vault://secrets/cred-1"}); err != nil {
		t.Fatalf("valid credential ref should pass format check: %v", err)
	}
}

func TestSkeletonValidateRepositoryRejectsInlineCredential(t *testing.T) {
	p := New("gitea")
	if err := p.ValidateRepository(context.Background(), scm.Repository{ID: "repo-1"}); err == nil {
		t.Fatal("expected error for missing url")
	}
	if err := p.ValidateRepository(context.Background(), scm.Repository{ID: "repo-1", URL: "https://user:token@example.invalid/repo.git"}); err == nil {
		t.Fatal("expected error for inline credential in url")
	}
	if err := p.ValidateRepository(context.Background(), scm.Repository{ID: "repo-1", URL: "https://example.invalid/repo.git"}); err != nil {
		t.Fatalf("valid repository should pass: %v", err)
	}
}

func TestSkeletonNetworkAndWriteOperationsReturnNotImplemented(t *testing.T) {
	p := New("github")
	ctx := context.Background()
	ref := scm.RepositoryRef{RepositoryID: "repo-1", URL: "https://example.invalid/repo.git"}

	cases := []struct {
		name string
		fn   func() error
	}{
		{"ResolveRef", func() error { _, err := p.ResolveRef(ctx, ref); return err }},
		{"ListBranches", func() error { _, err := p.ListBranches(ctx, "repo-1"); return err }},
		{"ListTags", func() error { _, err := p.ListTags(ctx, "repo-1"); return err }},
		{"GetCommit", func() error { _, err := p.GetCommit(ctx, "repo-1", "main"); return err }},
		{"ReadFile", func() error { _, err := p.ReadFile(ctx, ref, "README.md"); return err }},
		{"ListTree", func() error { _, err := p.ListTree(ctx, ref); return err }},
		{"CreateSnapshot", func() error { _, err := p.CreateSnapshot(ctx, ref); return err }},
		{"DiffRefs", func() error { _, err := p.DiffRefs(ctx, ref, ref); return err }},
		{"CreateCommitStatus", func() error { return p.CreateCommitStatus(ctx, "repo-1", "sha", scm.CommitStatus{State: "success"}) }},
	}
	for _, tc := range cases {
		if err := tc.fn(); !errors.Is(err, ErrNotImplemented) {
			t.Fatalf("%s expected ErrNotImplemented, got %v", tc.name, err)
		}
	}
}

func TestSkeletonGetRepositoryLookupRegisteredMetadata(t *testing.T) {
	p := New("gitlab")
	if _, err := p.GetRepository(context.Background(), "repo-1"); err == nil {
		t.Fatal("expected error for unregistered repository")
	}
	p.Register(scm.Repository{ID: "repo-1", Name: "demo", URL: "https://example.invalid/repo.git", Provider: "gitlab"})
	got, err := p.GetRepository(context.Background(), "repo-1")
	if err != nil {
		t.Fatalf("registered repository lookup: %v", err)
	}
	if got.Name != "demo" {
		t.Fatalf("got repository name = %q, want demo", got.Name)
	}
}

func TestSkeletonDoesNotLeakCredentialValues(t *testing.T) {
	p := New("gitea")
	p.Register(scm.Repository{ID: "repo-1", Name: "demo", URL: "https://example.invalid/repo.git", CredentialRef: "cred-ref-1"})
	got, err := p.GetRepository(context.Background(), "repo-1")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	// CredentialRef is metadata (an id), not a secret value; assert only the id is present.
	if !strings.Contains(got.CredentialRef, "cred-ref-1") {
		t.Fatalf("credential ref metadata missing: %#v", got)
	}
}
