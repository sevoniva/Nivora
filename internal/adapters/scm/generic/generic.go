package generic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sevoniva/nivora/internal/ports/scm"
)

var ErrNotImplemented = errors.New("generic scm adapter write operation is not implemented")

type Provider struct {
	repositories map[string]scm.Repository
}

func New() *Provider {
	return &Provider{repositories: map[string]scm.Repository{}}
}

func NewWithRepositories(repositories []scm.Repository) *Provider {
	p := New()
	for _, repository := range repositories {
		p.repositories[repository.ID] = repository
	}
	return p
}

func (p *Provider) ValidateCredential(ctx context.Context, credential scm.CredentialRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func (p *Provider) GetRepository(ctx context.Context, repositoryID string) (scm.Repository, error) {
	if err := ctx.Err(); err != nil {
		return scm.Repository{}, err
	}
	repository, ok := p.repositories[repositoryID]
	if !ok {
		return scm.Repository{}, fmt.Errorf("repository %q is not registered with generic scm provider", repositoryID)
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
	if path := localPath(repository.URL); path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat local repository path: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("local repository path %q is not a directory", path)
		}
	}
	return nil
}

func (p *Provider) ResolveRef(ctx context.Context, ref scm.RepositoryRef) (scm.Commit, error) {
	tree, err := p.ListTree(ctx, ref)
	if err != nil {
		return scm.Commit{}, err
	}
	sha := tree.CommitSHA
	if sha == "" {
		sha = tree.TreeHash
	}
	return scm.Commit{SHA: sha, Message: "generic read-only repository snapshot", Author: "nivora"}, nil
}

func (p *Provider) ListBranches(ctx context.Context, repositoryID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	repository, ok := p.repositories[repositoryID]
	if !ok {
		return []string{}, nil
	}
	branch := strings.TrimSpace(repository.DefaultBranch)
	if branch == "" {
		branch = "main"
	}
	return []string{branch}, nil
}

func (p *Provider) ListTags(ctx context.Context, repositoryID string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{}, nil
}

func (p *Provider) GetCommit(ctx context.Context, repositoryID string, ref string) (scm.Commit, error) {
	repository, err := p.GetRepository(ctx, repositoryID)
	if err != nil {
		return scm.Commit{}, err
	}
	return p.ResolveRef(ctx, scm.RepositoryRef{RepositoryID: repositoryID, URL: repository.URL, Provider: repository.Provider, Ref: ref})
}

func (p *Provider) ReadFile(ctx context.Context, ref scm.RepositoryRef, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := p.resolveLocalRoot(ref)
	if err != nil {
		return nil, err
	}
	clean := filepath.Clean(path)
	if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return nil, fmt.Errorf("repository file path %q is outside repository root", path)
	}
	fullPath := filepath.Join(root, clean)
	if !strings.HasPrefix(fullPath, root+string(os.PathSeparator)) && fullPath != root {
		return nil, fmt.Errorf("repository file path %q is outside repository root", path)
	}
	body, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (p *Provider) ListTree(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	if err := ctx.Err(); err != nil {
		return scm.Tree{}, err
	}
	root, err := p.resolveLocalRoot(ref)
	if err != nil {
		return scm.Tree{}, err
	}
	var files []scm.FileInfo
	warnings := []string{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, fmt.Sprintf("skip unreadable path %q: %v", safeRel(root, path), walkErr))
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		rel := safeRel(root, path)
		if entry.IsDir() {
			if skipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if skipFile(rel) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skip unreadable file %q: %v", rel, err))
			return nil
		}
		hash := ""
		if sensitiveFile(rel) {
			warnings = append(warnings, fmt.Sprintf("secret-like file %q recorded as metadata only; content was not read", rel))
		} else {
			var err error
			hash, err = fileHash(path, info.Size())
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("skip file hash for %q: %v", rel, err))
			}
		}
		files = append(files, scm.FileInfo{Path: filepath.ToSlash(rel), Size: info.Size(), Hash: hash})
		return nil
	})
	if err != nil {
		return scm.Tree{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	treeHash := hashTree(files)
	return scm.Tree{
		RepositoryID: strings.TrimSpace(ref.RepositoryID),
		Ref:          defaultRef(ref.Ref),
		TreeHash:     treeHash,
		Files:        files,
		Warnings:     warnings,
	}, nil
}

func (p *Provider) CreateSnapshot(ctx context.Context, ref scm.RepositoryRef) (scm.Tree, error) {
	return p.ListTree(ctx, ref)
}

func (p *Provider) DiffRefs(ctx context.Context, base scm.RepositoryRef, head scm.RepositoryRef) (scm.DiffSummary, error) {
	if err := ctx.Err(); err != nil {
		return scm.DiffSummary{}, err
	}
	baseTree, err := p.ListTree(ctx, base)
	if err != nil {
		return scm.DiffSummary{}, err
	}
	headTree, err := p.ListTree(ctx, head)
	if err != nil {
		return scm.DiffSummary{}, err
	}
	baseFiles := map[string]string{}
	for _, file := range baseTree.Files {
		baseFiles[file.Path] = file.Hash
	}
	headFiles := map[string]string{}
	for _, file := range headTree.Files {
		headFiles[file.Path] = file.Hash
	}
	var added, removed, changed []string
	for path, hash := range headFiles {
		baseHash, ok := baseFiles[path]
		if !ok {
			added = append(added, path)
			continue
		}
		if baseHash != hash {
			changed = append(changed, path)
		}
	}
	for path := range baseFiles {
		if _, ok := headFiles[path]; !ok {
			removed = append(removed, path)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)
	return scm.DiffSummary{
		BaseRef:      defaultRef(base.Ref),
		HeadRef:      defaultRef(head.Ref),
		AddedFiles:   added,
		RemovedFiles: removed,
		ChangedFiles: changed,
		Warnings:     append(baseTree.Warnings, headTree.Warnings...),
	}, nil
}

func (p *Provider) GetCapabilities(ctx context.Context) (scm.Capabilities, error) {
	if err := ctx.Err(); err != nil {
		return scm.Capabilities{}, err
	}
	return scm.Capabilities{
		Provider:          "generic_git",
		ReadOnly:          true,
		SupportsLocalPath: true,
		SupportsWrite:     false,
		Warnings:          []string{"generic provider is read-only; external network clone/write integrations are guarded future work"},
	}, nil
}

func (p *Provider) CreateCommitStatus(ctx context.Context, repositoryID string, sha string, status scm.CommitStatus) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrNotImplemented
}

func (p *Provider) resolveLocalRoot(ref scm.RepositoryRef) (string, error) {
	path := strings.TrimSpace(ref.LocalPath)
	if path == "" {
		path = localPath(ref.URL)
	}
	if path == "" && ref.RepositoryID != "" {
		if repository, ok := p.repositories[ref.RepositoryID]; ok {
			path = localPath(repository.URL)
		}
	}
	if path == "" {
		return "", errors.New("generic provider requires a local path or file:// repository URL for read-only snapshots")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository root %q is not a directory", abs)
	}
	return abs, nil
}

func localPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "file://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return parsed.Path
	}
	if strings.HasPrefix(raw, ".") || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "~") {
		return raw
	}
	return ""
}

func hasInlineCredential(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return false
	}
	return parsed.User.Username() != "" || hasPassword(parsed.User)
}

func hasPassword(user *url.Userinfo) bool {
	if user == nil {
		return false
	}
	_, ok := user.Password()
	return ok
}

func defaultRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "HEAD"
	}
	return ref
}

func safeRel(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func skipDir(rel string) bool {
	name := filepath.Base(rel)
	switch name {
	case ".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", ".next", ".vite", "coverage", ".terraform":
		return true
	default:
		return false
	}
}

func skipFile(rel string) bool {
	name := filepath.Base(rel)
	if strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".key") {
		return true
	}
	return false
}

func sensitiveFile(rel string) bool {
	name := strings.ToLower(filepath.Base(rel))
	switch name {
	case ".env", ".env.local", ".env.production", ".envrc", ".npmrc", ".pypirc", ".netrc", ".dockerconfigjson", "kubeconfig", "config":
		return true
	}
	return strings.Contains(name, "secret") ||
		strings.Contains(name, "password") ||
		strings.Contains(name, "token") ||
		strings.Contains(name, "credential") ||
		strings.HasPrefix(name, "id_rsa") ||
		strings.HasPrefix(name, "id_ed25519")
}

func fileHash(path string, size int64) (string, error) {
	const maxHashSize = 2 * 1024 * 1024
	if size > maxHashSize {
		return fmt.Sprintf("size:%d", size), nil
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func hashTree(files []scm.FileInfo) string {
	h := sha256.New()
	for _, file := range files {
		_, _ = h.Write([]byte(file.Path))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(file.Hash))
		_, _ = h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}
