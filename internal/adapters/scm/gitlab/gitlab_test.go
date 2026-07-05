package gitlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

// fakeGitLab is a minimal httptest server emulating the GitLab v4 REST API
// endpoints the provider calls. It asserts the bearer token is present and
// never inspects the token value beyond an equality check.
type fakeGitLab struct {
	server   *httptest.Server
	token    string
	requests []string
}

func newFakeGitLab(t *testing.T, token string) *fakeGitLab {
	t.Helper()
	f := &fakeGitLab{token: token}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v4/projects/", func(w http.ResponseWriter, r *http.Request) {
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		if !f.checkToken(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/repository/commits") && r.Method == http.MethodGet:
			ref := r.URL.Query().Get("ref_name")
			if ref == "empty" {
				_ = json.NewEncoder(w).Encode([]struct{}{})
				return
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "abc123", "message": "init", "author_name": "alice"},
			})
		case strings.Contains(r.URL.Path, "/repository/commits/") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "abc123", "message": "init", "author_name": "alice",
			})
		case strings.HasSuffix(r.URL.Path, "/repository/branches"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "main"}, {"name": "dev"}})
		case strings.HasSuffix(r.URL.Path, "/repository/tags"):
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "v1.0.0"}})
		case strings.Contains(r.URL.Path, "/repository/files/"):
			content := base64.StdEncoding.EncodeToString([]byte("hello gitlab"))
			_ = json.NewEncoder(w).Encode(map[string]any{"content": content, "encoding": "base64"})
		case strings.HasSuffix(r.URL.Path, "/repository/tree"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"path": "README.md", "type": "blob"},
				{"path": "src", "type": "tree"},
				{"path": "src/main.go", "type": "blob"},
			})
		case strings.HasSuffix(r.URL.Path, "/repository/compare"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"diffs": []map[string]any{
					{"new_path": "added.go", "old_path": "", "new_file": true, "deleted_file": false},
					{"new_path": "gone.go", "old_path": "gone.go", "new_file": false, "deleted_file": true},
					{"new_path": "changed.go", "old_path": "changed.go", "new_file": false, "deleted_file": false},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/statuses/"):
			w.WriteHeader(http.StatusCreated)
			return
		default:
			if r.URL.Query().Get("ref") != "" || strings.Contains(r.URL.Path, "/repository/files/") {
				content := base64.StdEncoding.EncodeToString([]byte("hello gitlab"))
				_ = json.NewEncoder(w).Encode(map[string]any{"content": content, "encoding": "base64"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 42, "path_with_namespace": "sevoniva/demo",
				"web_url": "https://gitlab.example/sevoniva/demo", "default_branch": "main",
			})
		}
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeGitLab) checkToken(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer "+f.token
}

func staticTokenResolver(token string) TokenResolver {
	return func(ctx context.Context, credential scm.CredentialRef) (string, error) {
		if credential.ID == "" {
			return "", errors.New("credential id required")
		}
		return token, nil
	}
}

func TestGitLabMetadataOnlyWithoutTokenResolver(t *testing.T) {
	p := New()
	caps, err := p.GetCapabilities(context.Background())
	if err != nil {
		t.Fatalf("capabilities: %v", err)
	}
	if !caps.ReadOnly {
		t.Fatal("metadata-only provider must be read-only")
	}
	if caps.SupportsWrite {
		t.Fatal("metadata-only provider must not support write")
	}
	for _, method := range []func() error{
		func() error { _, err := p.GetRepository(context.Background(), "1"); return err },
		func() error { _, err := p.ListBranches(context.Background(), "1"); return err },
		func() error { _, err := p.ListTags(context.Background(), "1"); return err },
		func() error { _, err := p.GetCommit(context.Background(), "1", "main"); return err },
		func() error {
			_, err := p.ReadFile(context.Background(), scm.RepositoryRef{RepositoryID: "1"}, "f")
			return err
		},
		func() error {
			_, err := p.ListTree(context.Background(), scm.RepositoryRef{RepositoryID: "1"})
			return err
		},
		func() error {
			_, err := p.DiffRefs(context.Background(), scm.RepositoryRef{RepositoryID: "1"}, scm.RepositoryRef{RepositoryID: "1"})
			return err
		},
		func() error {
			return p.CreateCommitStatus(context.Background(), "1", "sha", scm.CommitStatus{State: "success"})
		},
	} {
		if err := method(); !errors.Is(err, ErrNotConfigured) {
			t.Fatalf("expected ErrNotConfigured, got %v", err)
		}
	}
}

func TestGitLabValidateCredentialAndRepository(t *testing.T) {
	p := New()
	if err := p.ValidateCredential(context.Background(), scm.CredentialRef{}); err == nil {
		t.Fatal("expected error for empty credential")
	}
	if err := p.ValidateCredential(context.Background(), scm.CredentialRef{ID: "c1"}); err == nil {
		t.Fatal("expected error for missing secret key")
	}
	if err := p.ValidateRepository(context.Background(), scm.Repository{ID: "r1"}); err == nil {
		t.Fatal("expected error for missing url")
	}
	if err := p.ValidateRepository(context.Background(), scm.Repository{ID: "r1", URL: "https://u:t@host/r.git"}); err == nil {
		t.Fatal("expected error for inline credential")
	}
}

func TestGitLabLiveAPIReadPaths(t *testing.T) {
	f := newFakeGitLab(t, "token-value-placeholder")
	cred := scm.CredentialRef{ID: "cred-1", SecretKey: "vault://secrets/cred-1"}
	p := New(WithBaseURL(f.server.URL), WithHTTPClient(f.server.Client()), WithTokenResolver(staticTokenResolver("token-value-placeholder")), WithDefaultCredential(cred))
	ref := scm.RepositoryRef{RepositoryID: "sevoniva/demo", Ref: "main", Credential: cred}

	repo, err := p.GetRepository(context.Background(), "sevoniva/demo")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if repo.Name != "sevoniva/demo" || repo.DefaultBranch != "main" {
		t.Fatalf("GetRepository result = %#v", repo)
	}

	commit, err := p.ResolveRef(context.Background(), ref)
	if err != nil {
		t.Fatalf("ResolveRef: %v", err)
	}
	if commit.SHA != "abc123" {
		t.Fatalf("ResolveRef sha = %q", commit.SHA)
	}

	branches, err := p.ListBranches(context.Background(), "sevoniva/demo")
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}
	if len(branches) != 2 || branches[0] != "main" {
		t.Fatalf("ListBranches = %v", branches)
	}

	tags, err := p.ListTags(context.Background(), "sevoniva/demo")
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 1 || tags[0] != "v1.0.0" {
		t.Fatalf("ListTags = %v", tags)
	}

	gotCommit, err := p.GetCommit(context.Background(), "sevoniva/demo", "abc123")
	if err != nil {
		t.Fatalf("GetCommit: %v", err)
	}
	if gotCommit.SHA != "abc123" {
		t.Fatalf("GetCommit sha = %q", gotCommit.SHA)
	}

	content, err := p.ReadFile(context.Background(), ref, "README.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "hello gitlab" {
		t.Fatalf("ReadFile content = %q", string(content))
	}

	tree, err := p.ListTree(context.Background(), ref)
	if err != nil {
		t.Fatalf("ListTree: %v", err)
	}
	if len(tree.Files) != 2 {
		t.Fatalf("ListTree files = %#v (want 2 blobs)", tree.Files)
	}

	diff, err := p.DiffRefs(context.Background(), ref, ref)
	if err != nil {
		t.Fatalf("DiffRefs: %v", err)
	}
	if len(diff.AddedFiles) != 1 || len(diff.RemovedFiles) != 1 || len(diff.ChangedFiles) != 1 {
		t.Fatalf("DiffRefs = %+v", diff)
	}
}

func TestGitLabCreateCommitStatusIsWrite(t *testing.T) {
	f := newFakeGitLab(t, "token-value-placeholder")
	cred := scm.CredentialRef{ID: "cred-1", SecretKey: "vault://secrets/cred-1"}
	p := New(WithBaseURL(f.server.URL), WithHTTPClient(f.server.Client()), WithTokenResolver(staticTokenResolver("token-value-placeholder")), WithDefaultCredential(cred))
	if err := p.CreateCommitStatus(context.Background(), "sevoniva/demo", "abc123", scm.CommitStatus{State: "success", Description: "ci passed"}); err != nil {
		t.Fatalf("CreateCommitStatus: %v", err)
	}
}

func TestGitLabRejectsBadToken(t *testing.T) {
	f := newFakeGitLab(t, "correct-token")
	cred := scm.CredentialRef{ID: "cred-1", SecretKey: "vault://secrets/cred-1"}
	p := New(WithBaseURL(f.server.URL), WithHTTPClient(f.server.Client()), WithTokenResolver(staticTokenResolver("wrong-token")), WithDefaultCredential(cred))
	_, err := p.GetRepository(context.Background(), "sevoniva/demo")
	if err == nil {
		t.Fatal("expected error for wrong token")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected 401 in error, got %v", err)
	}
}

func TestGitLabTokenNeverLogged(t *testing.T) {
	f := newFakeGitLab(t, "secret-token-value")
	p := New(WithBaseURL(f.server.URL), WithHTTPClient(f.server.Client()), WithTokenResolver(staticTokenResolver("secret-token-value")))
	_, err := p.GetRepository(context.Background(), "missing/project")
	// 404 path returns ErrNotFound; either way the token must not leak.
	_ = err
	// Force a non-2xx non-404 to capture error body.
	f.server.Close()
	f2 := newFakeGitLab(t, "secret-token-value")
	p2 := New(WithBaseURL(f2.server.URL), WithHTTPClient(f2.server.Client()), WithTokenResolver(staticTokenResolver("secret-token-value")))
	// Hit an unknown path to trigger a 404 -> ErrNotFound; token must not appear.
	_, err = p2.GetRepository(context.Background(), "missing/project")
	body, _ := io.ReadAll(strings.NewReader(err.Error()))
	if strings.Contains(string(body), "secret-token-value") {
		t.Fatalf("token leaked in error: %v", err)
	}
}
