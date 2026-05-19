package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	"github.com/sevoniva/nivora/internal/ports/artifact"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

var ErrNotImplemented = errors.New("oci artifact adapter is not implemented")

type Config struct {
	Name          string
	Endpoint      string
	Insecure      bool
	Username      string
	Password      string
	Token         string
	CredentialRef artifact.CredentialRef
}

type Option func(*Provider)

type Provider struct {
	client  *http.Client
	config  Config
	secrets portsecret.Provider
}

func New(options ...Option) *Provider {
	p := &Provider{
		client: http.DefaultClient,
	}
	for _, option := range options {
		option(p)
	}
	return p
}

func WithHTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		if client != nil {
			p.client = client
		}
	}
}

func WithConfig(config Config) Option {
	return func(p *Provider) {
		p.config = config
	}
}

func WithInsecure(insecure bool) Option {
	return func(p *Provider) {
		p.config.Insecure = insecure
	}
}

func WithSecretProvider(provider portsecret.Provider) Option {
	return func(p *Provider) {
		p.secrets = provider
	}
}

func (p *Provider) ValidateCredential(ctx context.Context, credential artifact.CredentialRef) error {
	if credential.ID == "" && credential.SecretKey == "" {
		return fmt.Errorf("credential ref is required")
	}
	_, err := p.loadCredential(ctx, credential)
	return err
}

func (p *Provider) GetArtifact(ctx context.Context, name string, reference string) (domainartifact.Artifact, error) {
	inspection, err := p.InspectReference(ctx, reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return domainartifact.Artifact{}, err
	}
	resolution, err := p.ResolveDigest(ctx, name, inspection.Reference.Normalized)
	if err != nil {
		resolution = domainartifact.Resolution{Reference: inspection.Reference, Digest: inspection.Reference.Digest, Resolved: inspection.Reference.Digest != ""}
	}
	return domainartifact.Artifact{
		Type:           inspection.Reference.Type,
		Name:           name,
		Version:        inspection.Reference.Version,
		Reference:      inspection.Reference.Normalized,
		Digest:         resolution.Digest,
		Registry:       inspection.Reference.Registry,
		Repository:     inspection.Reference.Repository,
		MediaType:      resolution.MediaType,
		SizeBytes:      resolution.SizeBytes,
		ManifestSchema: resolution.ManifestSchema,
		CreatedAt:      time.Now(),
	}, nil
}

func (p *Provider) ListArtifacts(ctx context.Context, repository string) ([]domainartifact.Artifact, error) {
	return nil, ErrNotImplemented
}

func (p *Provider) ResolveDigest(ctx context.Context, name string, reference string) (domainartifact.Resolution, error) {
	inspection, err := p.InspectReference(ctx, reference, domainartifact.ArtifactTypeImage)
	if err != nil {
		return domainartifact.Resolution{}, err
	}
	resolution := domainartifact.Resolution{
		Reference: inspection.Reference,
		Digest:    inspection.Reference.Digest,
		Resolved:  inspection.Reference.Digest != "",
		Warnings:  append([]domainartifact.Warning(nil), inspection.Warnings...),
	}
	if p.config.Insecure {
		resolution.Warnings = append(resolution.Warnings, domainartifact.Warning{
			Code:    "insecure_registry",
			Message: "OCI registry is configured as insecure; use only for local development",
		})
	}
	if resolution.Resolved {
		resolution.DigestQualifiedReference = domainartifact.DigestQualifiedReference(inspection.Reference, resolution.Digest)
		resolution.ResolvedAt = time.Now()
		return resolution, nil
	}
	if inspection.Reference.Registry == "" && p.config.Endpoint == "" {
		return resolution, fmt.Errorf("registry is required to resolve digest for %q", reference)
	}
	endpoint, err := p.registryEndpoint(inspection.Reference)
	if err != nil {
		return resolution, err
	}
	manifestURL := endpoint.ResolveReference(&url.URL{Path: "/v2/" + inspection.Reference.Repository + "/manifests/" + manifestIdentifier(inspection.Reference)})
	digest, mediaType, sizeBytes, manifestSchema, err := p.fetchManifestDigest(ctx, manifestURL.String())
	if err != nil {
		return resolution, err
	}
	resolution.Digest = digest
	resolution.DigestQualifiedReference = domainartifact.DigestQualifiedReference(inspection.Reference, digest)
	resolution.MediaType = mediaType
	resolution.SizeBytes = sizeBytes
	resolution.ManifestSchema = manifestSchema
	resolution.Resolved = true
	resolution.ResolvedAt = time.Now()
	resolution.Reference.Digest = digest
	resolution.Reference.Immutable = true
	resolution.Reference.IsDigestPinned = true
	return resolution, nil
}

func (p *Provider) InspectReference(ctx context.Context, reference string, artifactType domainartifact.ArtifactType) (domainartifact.Inspection, error) {
	select {
	case <-ctx.Done():
		return domainartifact.Inspection{}, ctx.Err()
	default:
	}
	return domainartifact.InspectReference(reference, artifactType)
}

func (p *Provider) Capabilities() artifact.Capabilities {
	return artifact.Capabilities{
		SupportsDigestResolution:     true,
		SupportsListing:              false,
		SupportsCredentialValidation: true,
	}
}

func (p *Provider) registryEndpoint(ref domainartifact.Reference) (*url.URL, error) {
	raw := strings.TrimSpace(p.config.Endpoint)
	if raw == "" {
		raw = ref.Registry
	}
	if raw == "" {
		return nil, fmt.Errorf("registry endpoint is required")
	}
	if !strings.Contains(raw, "://") {
		scheme := "https"
		if p.config.Insecure {
			scheme = "http"
		}
		raw = scheme + "://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid registry endpoint: %w", err)
	}
	if parsed.Scheme == "http" && !p.config.Insecure {
		return nil, fmt.Errorf("insecure registry endpoint requires explicit insecure=true")
	}
	return parsed, nil
}

func manifestIdentifier(ref domainartifact.Reference) string {
	if ref.Digest != "" {
		return ref.Digest
	}
	if ref.Tag != "" {
		return ref.Tag
	}
	return "latest"
}

func (p *Provider) fetchManifestDigest(ctx context.Context, manifestURL string) (string, string, int64, string, error) {
	digest, mediaType, sizeBytes, _, err := p.manifestRequest(ctx, http.MethodHead, manifestURL)
	if err != nil {
		return "", "", 0, "", err
	}
	if digest != "" {
		return digest, mediaType, sizeBytes, "", nil
	}
	return p.manifestRequestWithBody(ctx, manifestURL)
}

func (p *Provider) manifestRequest(ctx context.Context, method string, manifestURL string) (string, string, int64, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, manifestURL, nil)
	if err != nil {
		return "", "", 0, 0, err
	}
	setManifestHeaders(req)
	if err := p.setAuth(ctx, req); err != nil {
		return "", "", 0, 0, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("resolve OCI digest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", "", 0, resp.StatusCode, fmt.Errorf("registry authorization failed or credentials are required")
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return "", "", 0, resp.StatusCode, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", 0, resp.StatusCode, fmt.Errorf("registry manifest request failed with status %d", resp.StatusCode)
	}
	return resp.Header.Get("Docker-Content-Digest"), resp.Header.Get("Content-Type"), resp.ContentLength, resp.StatusCode, nil
}

func (p *Provider) manifestRequestWithBody(ctx context.Context, manifestURL string) (string, string, int64, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return "", "", 0, "", err
	}
	setManifestHeaders(req)
	if err := p.setAuth(ctx, req); err != nil {
		return "", "", 0, "", err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", 0, "", fmt.Errorf("resolve OCI digest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", "", 0, "", fmt.Errorf("registry authorization failed or credentials are required")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", 0, "", fmt.Errorf("registry manifest request failed with status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	mediaType := resp.Header.Get("Content-Type")
	manifestSchema := ""
	if mediaType == "" {
		var payload struct {
			MediaType     string `json:"mediaType"`
			SchemaVersion int    `json:"schemaVersion"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			mediaType = payload.MediaType
			if payload.SchemaVersion > 0 {
				manifestSchema = fmt.Sprintf("schemaVersion:%d", payload.SchemaVersion)
			}
		}
	} else {
		var payload struct {
			SchemaVersion int `json:"schemaVersion"`
		}
		if err := json.Unmarshal(body, &payload); err == nil && payload.SchemaVersion > 0 {
			manifestSchema = fmt.Sprintf("schemaVersion:%d", payload.SchemaVersion)
		}
	}
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", "", 0, "", fmt.Errorf("registry manifest response did not include Docker-Content-Digest")
	}
	sizeBytes := resp.ContentLength
	if sizeBytes < 0 {
		sizeBytes = int64(len(body))
	}
	return digest, mediaType, sizeBytes, manifestSchema, nil
}

func setManifestHeaders(req *http.Request) {
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", "))
}

type registryCredential struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func (p *Provider) setAuth(ctx context.Context, req *http.Request) error {
	credential, err := p.loadCredential(ctx, p.config.CredentialRef)
	if err != nil {
		return err
	}
	if credential.Token != "" {
		req.Header.Set("Authorization", "Bearer "+credential.Token)
		return nil
	}
	if credential.Username != "" || credential.Password != "" {
		req.SetBasicAuth(credential.Username, credential.Password)
	}
	return nil
}

func (p *Provider) loadCredential(ctx context.Context, ref artifact.CredentialRef) (registryCredential, error) {
	credential := registryCredential{Username: p.config.Username, Password: p.config.Password, Token: p.config.Token}
	if credential.Username != "" || credential.Password != "" || credential.Token != "" {
		return credential, nil
	}
	if ref.ID == "" && ref.SecretKey == "" {
		return registryCredential{}, nil
	}
	if p.secrets == nil {
		return registryCredential{}, fmt.Errorf("secret provider is required for registry credential ref")
	}
	body, err := p.secrets.GetSecret(ctx, domaincredential.SecretRef{ID: ref.ID, Key: ref.SecretKey})
	if err != nil {
		return registryCredential{}, fmt.Errorf("registry credential lookup failed")
	}
	if err := json.Unmarshal(body, &credential); err == nil {
		return credential, nil
	}
	token := strings.TrimSpace(string(body))
	if token == "" {
		return registryCredential{}, fmt.Errorf("registry credential secret is empty")
	}
	return registryCredential{Token: token}, nil
}
