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
	"github.com/sevoniva/nivora/internal/ports/artifact"
)

var ErrNotImplemented = errors.New("oci artifact adapter is not implemented")

type Config struct {
	Name     string
	Endpoint string
	Insecure bool
	Username string
	Password string
}

type Option func(*Provider)

type Provider struct {
	client *http.Client
	config Config
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

func (p *Provider) ValidateCredential(ctx context.Context, credential artifact.CredentialRef) error {
	return ctx.Err()
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
		Type:       inspection.Reference.Type,
		Name:       name,
		Version:    inspection.Reference.Version,
		Reference:  inspection.Reference.Normalized,
		Digest:     resolution.Digest,
		Registry:   inspection.Reference.Registry,
		Repository: inspection.Reference.Repository,
		MediaType:  resolution.MediaType,
		CreatedAt:  time.Now(),
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
	digest, mediaType, err := p.fetchManifestDigest(ctx, manifestURL.String())
	if err != nil {
		return resolution, err
	}
	resolution.Digest = digest
	resolution.DigestQualifiedReference = domainartifact.DigestQualifiedReference(inspection.Reference, digest)
	resolution.MediaType = mediaType
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
		SupportsCredentialValidation: false,
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

func (p *Provider) fetchManifestDigest(ctx context.Context, manifestURL string) (string, string, error) {
	digest, mediaType, _, err := p.manifestRequest(ctx, http.MethodHead, manifestURL)
	if err != nil {
		return "", "", err
	}
	if digest != "" {
		return digest, mediaType, nil
	}
	return p.manifestRequestWithBody(ctx, manifestURL)
}

func (p *Provider) manifestRequest(ctx context.Context, method string, manifestURL string) (string, string, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, manifestURL, nil)
	if err != nil {
		return "", "", 0, err
	}
	setManifestHeaders(req)
	if p.config.Username != "" || p.config.Password != "" {
		req.SetBasicAuth(p.config.Username, p.config.Password)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("resolve OCI digest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", "", resp.StatusCode, fmt.Errorf("registry authorization failed or credentials are required")
	}
	if resp.StatusCode == http.StatusMethodNotAllowed {
		return "", "", resp.StatusCode, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", resp.StatusCode, fmt.Errorf("registry manifest request failed with status %d", resp.StatusCode)
	}
	return resp.Header.Get("Docker-Content-Digest"), resp.Header.Get("Content-Type"), resp.StatusCode, nil
}

func (p *Provider) manifestRequestWithBody(ctx context.Context, manifestURL string) (string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return "", "", err
	}
	setManifestHeaders(req)
	if p.config.Username != "" || p.config.Password != "" {
		req.SetBasicAuth(p.config.Username, p.config.Password)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("resolve OCI digest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", "", fmt.Errorf("registry authorization failed or credentials are required")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("registry manifest request failed with status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	mediaType := resp.Header.Get("Content-Type")
	if mediaType == "" {
		var payload struct {
			MediaType string `json:"mediaType"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			mediaType = payload.MediaType
		}
	}
	digest := resp.Header.Get("Docker-Content-Digest")
	if digest == "" {
		return "", "", fmt.Errorf("registry manifest response did not include Docker-Content-Digest")
	}
	return digest, mediaType, nil
}

func setManifestHeaders(req *http.Request) {
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", "))
}
