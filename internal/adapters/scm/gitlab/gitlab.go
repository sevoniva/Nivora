// Package gitlab implements a real GitLab SCM adapter over the GitLab v4 REST
// API using only the standard library (no third-party GitLab SDK, per
// docs/engineering/dependency-policy.md).
//
// The Provider satisfies the scm.SCMProvider interface. Network calls go to the
// configured GitLab base URL (default https://gitlab.com) and authenticate with
// a bearer token resolved from scm.CredentialRef via a TokenResolver function.
// When no TokenResolver is configured the provider degrades to metadata-only:
// credential and repository validation still run, but every network or write
// operation returns ErrNotConfigured. This keeps the provider safe to construct
// in dev/test without a live GitLab instance or secret access.
//
// Tokens are never logged, never returned in errors, and never stored beyond
// the lifetime of a single method call. CredentialRef metadata (id only) may
// appear in audit/logs; the resolved token value must not.
package gitlab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

const (
	providerName   = "gitlab"
	defaultBaseURL = "https://gitlab.com"
	apiVersion     = "v4"
)

// ErrNotConfigured is returned when a network or write operation is attempted
// without a TokenResolver. The provider degrades to metadata-only in that mode.
var ErrNotConfigured = errors.New("gitlab scm adapter is not configured with a token resolver; set a TokenResolver to enable live API calls")

// ErrNotFound is returned when the GitLab API responds with 404.
var ErrNotFound = errors.New("gitlab resource not found")

// TokenResolver resolves a scm.CredentialRef to a bearer token value. The
// returned token is used for a single API call and must not be retained.
// Implementations should resolve through the configured SecretProvider and
// return an error (never an empty token) when the credential is unavailable.
type TokenResolver func(ctx context.Context, credential scm.CredentialRef) (string, error)

// Provider is a real GitLab SCMProvider backed by the v4 REST API.
type Provider struct {
	baseURL           string
	httpClient        *http.Client
	tokenResolver     TokenResolver
	defaultCredential scm.CredentialRef
}

// Option configures a Provider.
type Option func(*Provider)

// WithBaseURL overrides the default GitLab base URL (https://gitlab.com).
// Use this for self-hosted GitLab instances. The URL must include the scheme
// and must not contain inline credentials.
func WithBaseURL(baseURL string) Option {
	return func(p *Provider) {
		if strings.TrimSpace(baseURL) != "" {
			p.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// WithHTTPClient overrides the default HTTP client. Use for custom transports,
// timeouts, or test doubles.
func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		if client != nil {
			p.httpClient = client
		}
	}
}

// WithTokenResolver enables live API calls by resolving CredentialRef metadata
// to a bearer token through the configured SecretProvider. Without this, the
// provider degrades to metadata-only.
func WithTokenResolver(resolver TokenResolver) Option {
	return func(p *Provider) {
		p.tokenResolver = resolver
	}
}

// WithDefaultCredential sets the credential used by methods whose
// scm.SCMProvider signature does not carry a CredentialRef (GetRepository,
// ListBranches, ListTags, GetCommit, CreateCommitStatus). Methods that take a
// RepositoryRef use that ref's Credential instead.
func WithDefaultCredential(credential scm.CredentialRef) Option {
	return func(p *Provider) {
		p.defaultCredential = credential
	}
}

// New returns a GitLab Provider. Without options it is metadata-only; pass
// WithTokenResolver (and optionally WithBaseURL) to enable live API calls.
func New(opts ...Option) *Provider {
	p := &Provider{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// projectPath builds the API path segment for a project. GitLab accepts either
// a numeric id or a URL-encoded "namespace/project" path. We pass the value
// through url.PathEscape so namespace/project works transparently.
func projectPathSegment(repositoryID string) string {
	return url.PathEscape(strings.TrimSpace(repositoryID))
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

func (p *Provider) GetCapabilities(ctx context.Context) (scm.Capabilities, error) {
	if err := ctx.Err(); err != nil {
		return scm.Capabilities{}, err
	}
	caps := scm.Capabilities{
		Provider:           providerName,
		ReadOnly:           false,
		RequiresCredential: true,
		SupportsWrite:      true,
		Warnings: []string{
			"gitlab adapter performs live REST API calls when a token resolver is configured",
			"commit status writes require explicit operator opt-in through RBAC and audit",
		},
	}
	if p.tokenResolver == nil {
		caps.ReadOnly = true
		caps.SupportsWrite = false
		caps.Warnings = []string{
			"gitlab adapter is metadata-only until a token resolver is configured",
			"live API clone/fetch/write integration requires WithTokenResolver",
		}
	}
	return caps, nil
}

// GetRepository fetches project metadata from GET /api/v4/projects/{id}.
func (p *Provider) GetRepository(ctx context.Context, repositoryID string) (scm.Repository, error) {
	if err := ctx.Err(); err != nil {
		return scm.Repository{}, err
	}
	if p.tokenResolver == nil {
		return scm.Repository{}, ErrNotConfigured
	}
	var project struct {
		ID                int    `json:"id"`
		PathWithNamespace string `json:"path_with_namespace"`
		WebURL            string `json:"web_url"`
		DefaultBranch     string `json:"default_branch"`
	}
	if err := p.get(ctx, p.defaultCredential, fmt.Sprintf("/projects/%s", projectPathSegment(repositoryID)), &project); err != nil {
		return scm.Repository{}, err
	}
	return scm.Repository{
		ID:            strconv.Itoa(project.ID),
		Name:          project.PathWithNamespace,
		URL:           project.WebURL,
		Provider:      providerName,
		DefaultBranch: project.DefaultBranch,
	}, nil
}

func (p *Provider) ResolveRef(ctx context.Context, ref scm.RepositoryRef) (scm.Commit, error) {
	if err := ctx.Err(); err != nil {
		return scm.Commit{}, err
	}
	if p.tokenResolver == nil {
		return scm.Commit{}, ErrNotConfigured
	}
	branch := ref.Ref
	if branch == "" {
		branch = "main"
	}
	var commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  string `json:"author_name"`
	}
	endpoint := fmt.Sprintf("/projects/%s/repository/commits?ref_name=%s", projectPathSegment(ref.RepositoryID), url.QueryEscape(branch))
	if err := p.get(ctx, ref.Credential, endpoint, &commits); err != nil {
		return scm.Commit{}, err
	}
	if len(commits) == 0 {
		return scm.Commit{}, ErrNotFound
	}
	return scm.Commit{SHA: commits[0].ID, Message: commits[0].Message, Author: commits[0].Author}, nil
}

func (p *Provider) ListBranches(ctx context.Context, repositoryID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.tokenResolver == nil {
		return nil, ErrNotConfigured
	}
	var branches []struct {
		Name string `json:"name"`
	}
	if err := p.get(ctx, p.defaultCredential, fmt.Sprintf("/projects/%s/repository/branches", projectPathSegment(repositoryID)), &branches); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(branches))
	for _, b := range branches {
		names = append(names, b.Name)
	}
	return names, nil
}

func (p *Provider) ListTags(ctx context.Context, repositoryID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.tokenResolver == nil {
		return nil, ErrNotConfigured
	}
	var tags []struct {
		Name string `json:"name"`
	}
	if err := p.get(ctx, p.defaultCredential, fmt.Sprintf("/projects/%s/repository/tags", projectPathSegment(repositoryID)), &tags); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
	}
	return names, nil
}

func (p *Provider) GetCommit(ctx context.Context, repositoryID string, ref string) (scm.Commit, error) {
	if err := ctx.Err(); err != nil {
		return scm.Commit{}, err
	}
	if p.tokenResolver == nil {
		return scm.Commit{}, ErrNotConfigured
	}
	var commit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  string `json:"author_name"`
	}
	if err := p.get(ctx, p.defaultCredential, fmt.Sprintf("/projects/%s/repository/commits/%s", projectPathSegment(repositoryID), url.PathEscape(ref)), &commit); err != nil {
		return scm.Commit{}, err
	}
	return scm.Commit{SHA: commit.ID, Message: commit.Message, Author: commit.Author}, nil
}

func (p *Provider) ReadFile(ctx context.Context, ref scm.RepositoryRef, filePath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if p.tokenResolver == nil {
		return nil, ErrNotConfigured
	}
	branch := ref.Ref
	if branch == "" {
		branch = "main"
	}
	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	endpoint := fmt.Sprintf("/projects/%s/repository/files/%s?ref=%s", projectPathSegment(ref.RepositoryID), url.PathEscape(filePath), url.QueryEscape(branch))
	if err := p.get(ctx, ref.Credential, endpoint, &file); err != nil {
		return nil, err
	}
	if file.Encoding != "base64" {
		return []byte(file.Content), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return nil, fmt.Errorf("decode gitlab file content: %w", err)
	}
	return decoded, nil
}

func (p *Provider) ListTree(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	if err := ctx.Err(); err != nil {
		return scm.Tree{}, err
	}
	if p.tokenResolver == nil {
		return scm.Tree{}, ErrNotConfigured
	}
	branch := ref.Ref
	if branch == "" {
		branch = "main"
	}
	var entries []struct {
		Path string `json:"path"`
		Type string `json:"type"`
		Mode string `json:"mode"`
	}
	endpoint := fmt.Sprintf("/projects/%s/repository/tree?ref=%s&recursive=true&per_page=100", projectPathSegment(ref.RepositoryID), url.QueryEscape(branch))
	if err := p.get(ctx, ref.Credential, endpoint, &entries); err != nil {
		return scm.Tree{}, err
	}
	files := make([]scm.FileInfo, 0, len(entries))
	for _, e := range entries {
		if e.Type == "blob" {
			files = append(files, scm.FileInfo{Path: e.Path})
		}
	}
	return scm.Tree{RepositoryID: ref.RepositoryID, Ref: branch, Files: files}, nil
}

func (p *Provider) CreateSnapshot(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	// ponytail: snapshot is a tree read in this adapter; GitLab has no separate
	// snapshot endpoint. If a content hash is needed, upgrade ListTree to fetch
	// blob SHAs and hash them.
	return p.ListTree(ctx, ref)
}

func (p *Provider) DiffRefs(ctx context.Context, base scm.RepositoryRef, head scm.RepositoryRef) (scm.DiffSummary, error) {
	if err := ctx.Err(); err != nil {
		return scm.DiffSummary{}, err
	}
	if p.tokenResolver == nil {
		return scm.DiffSummary{}, ErrNotConfigured
	}
	var compare struct {
		Diffs []struct {
			NewPath string `json:"new_path"`
			OldPath string `json:"old_path"`
			NewFile bool   `json:"new_file"`
			Deleted bool   `json:"deleted_file"`
		} `json:"diffs"`
	}
	endpoint := fmt.Sprintf("/projects/%s/repository/compare?from=%s&to=%s", projectPathSegment(base.RepositoryID), url.QueryEscape(base.Ref), url.QueryEscape(head.Ref))
	if err := p.get(ctx, base.Credential, endpoint, &compare); err != nil {
		return scm.DiffSummary{}, err
	}
	summary := scm.DiffSummary{BaseRef: base.Ref, HeadRef: head.Ref}
	for _, d := range compare.Diffs {
		switch {
		case d.NewFile:
			summary.AddedFiles = append(summary.AddedFiles, d.NewPath)
		case d.Deleted:
			summary.RemovedFiles = append(summary.RemovedFiles, d.OldPath)
		default:
			summary.ChangedFiles = append(summary.ChangedFiles, d.NewPath)
		}
	}
	return summary, nil
}

// CreateCommitStatus posts a commit status to POST /api/v4/projects/{id}/statuses/{sha}.
// This is a write operation; callers must gate it behind RBAC, explicit allow
// flags, confirmation, and audit. MCP must not trigger it directly.
func (p *Provider) CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status scm.CommitStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if p.tokenResolver == nil {
		return ErrNotConfigured
	}
	form := url.Values{}
	form.Set("state", status.State)
	if status.Description != "" {
		form.Set("description", status.Description)
	}
	if status.TargetURL != "" {
		form.Set("target_url", status.TargetURL)
	}
	endpoint := fmt.Sprintf("/projects/%s/statuses/%s", projectPathSegment(repositoryID), url.PathEscape(sha))
	return p.postForm(ctx, p.defaultCredential, endpoint, form, nil)
}

// get performs an authenticated GET against the GitLab API. The credential is
// resolved to a token on each call; the token is used in the Authorization
// header and discarded. Errors never include the token value.
func (p *Provider) get(ctx context.Context, credential scm.CredentialRef, endpoint string, target any) error {
	return p.do(ctx, http.MethodGet, credential, endpoint, nil, target)
}

// postForm performs an authenticated URL-encoded POST against the GitLab API.
func (p *Provider) postForm(ctx context.Context, credential scm.CredentialRef, endpoint string, form url.Values, target any) error {
	body := strings.NewReader(form.Encode())
	req, err := p.newRequest(ctx, http.MethodPost, credential, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return p.execute(req, target)
}

func (p *Provider) do(ctx context.Context, method string, credential scm.CredentialRef, endpoint string, body io.Reader, target any) error {
	req, err := p.newRequest(ctx, method, credential, endpoint, body)
	if err != nil {
		return err
	}
	return p.execute(req, target)
}

func (p *Provider) newRequest(ctx context.Context, method string, credential scm.CredentialRef, endpoint string, body io.Reader) (*http.Request, error) {
	u := p.baseURL + "/api/" + apiVersion + endpoint
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if credential.ID != "" || credential.SecretKey != "" {
		token, err := p.tokenResolver(ctx, credential)
		if err != nil {
			return nil, fmt.Errorf("resolve gitlab credential: %w", err)
		}
		if token == "" {
			return nil, errors.New("resolved gitlab token is empty; credential misconfigured")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

func (p *Provider) execute(req *http.Request, target any) error {
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("gitlab api request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("gitlab api %s %s returned status %d: %s", req.Method, path.Base(req.URL.Path), resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if target == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode gitlab response: %w", err)
	}
	return nil
}

// hasInlineCredential rejects URLs with userinfo such as
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
	return u.User.Username() != ""
}
