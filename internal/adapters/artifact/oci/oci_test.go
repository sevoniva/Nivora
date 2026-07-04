package oci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domaincredential "github.com/sevoniva/nivora/internal/domain/credential"
	"github.com/sevoniva/nivora/internal/ports/artifact"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

func TestResolveDigestFromRegistryHEAD(t *testing.T) {
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/v2/team/app/manifests/1.0.0" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := New(WithConfig(Config{Endpoint: server.URL, Insecure: true}))
	resolution, err := provider.ResolveDigest(context.Background(), "app", "registry.example.com/team/app:1.0.0")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !resolution.Resolved || resolution.Digest != digest || resolution.MediaType == "" {
		t.Fatalf("resolution = %#v", resolution)
	}
	if resolution.DigestQualifiedReference != "registry.example.com/team/app:1.0.0@"+digest {
		t.Fatalf("digest reference = %q", resolution.DigestQualifiedReference)
	}
	if len(resolution.Warnings) == 0 || resolution.Warnings[len(resolution.Warnings)-1].Code != "insecure_registry" {
		t.Fatalf("warnings = %#v", resolution.Warnings)
	}
}

func TestResolveDigestFallsBackToGET(t *testing.T) {
	const digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.WriteHeader(http.StatusMethodNotAllowed)
		case http.MethodGet:
			w.Header().Set("Docker-Content-Digest", digest)
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			_, _ = w.Write([]byte(`{"schemaVersion":2}`))
		default:
			t.Fatalf("method = %s", r.Method)
		}
	}))
	defer server.Close()

	provider := New(WithConfig(Config{Endpoint: server.URL, Insecure: true}))
	resolution, err := provider.ResolveDigest(context.Background(), "app", "registry.example.com/team/app:1.0.0")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if resolution.Digest != digest || resolution.SizeBytes == 0 || resolution.ManifestSchema != "schemaVersion:2" {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestResolveDigestUsesSecretProviderCredential(t *testing.T) {
	const digest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "registry-user" || pass != "registry-pass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Docker-Content-Digest", digest)
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := New(
		WithConfig(Config{
			Endpoint:      server.URL,
			Insecure:      true,
			CredentialRef: CredentialRefForTest(),
		}),
		WithSecretProvider(fakeSecretProvider{value: []byte(`{"username":"registry-user","password":"registry-pass"}`)}),
	)
	resolution, err := provider.ResolveDigest(context.Background(), "app", "registry.example.com/team/app:1.0.0")
	if err != nil {
		t.Fatalf("resolve with secret provider: %v", err)
	}
	if resolution.Digest != digest {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestResolveDigestRequiresExplicitInsecure(t *testing.T) {
	provider := New(WithConfig(Config{Endpoint: "http://registry.example.com"}))
	_, err := provider.ResolveDigest(context.Background(), "app", "registry.example.com/team/app:1.0.0")
	if err == nil || !strings.Contains(err.Error(), "insecure") {
		t.Fatalf("expected insecure error, got %v", err)
	}
}

func TestResolveDigestAuthErrorDoesNotLeakCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	provider := New(WithConfig(Config{
		Endpoint: server.URL,
		Insecure: true,
		Username: "example-user",
		Password: "example-pass",
	}))
	_, err := provider.ResolveDigest(context.Background(), "app", "registry.example.com/team/app:1.0.0")
	if err == nil {
		t.Fatal("expected auth error")
	}
	if strings.Contains(err.Error(), "example-user") || strings.Contains(err.Error(), "example-pass") {
		t.Fatalf("credential leaked in error: %v", err)
	}
}

func TestResolveDigestPinnedReferenceDoesNotCallRegistry(t *testing.T) {
	provider := New()
	resolution, err := provider.ResolveDigest(context.Background(), "app", "registry.example.com/team/app@sha256:abcdef")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !resolution.Resolved || resolution.Digest != "sha256:abcdef" {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestListArtifactsFromRegistryTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/v2/team/app/tags/list" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"team/app","tags":["2.0.0","latest","1.0.0"]}`))
	}))
	defer server.Close()

	provider := New(WithConfig(Config{Endpoint: server.URL, Insecure: true}))
	artifacts, err := provider.ListArtifacts(context.Background(), "registry.example.com/team/app")
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 3 {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	if artifacts[0].Version != "1.0.0" || artifacts[1].Version != "2.0.0" || artifacts[2].Version != "latest" {
		t.Fatalf("artifacts not sorted by tag: %#v", artifacts)
	}
	if artifacts[0].Name != "app" || artifacts[0].Repository != "team/app" || artifacts[0].Reference != "registry.example.com/team/app:1.0.0" {
		t.Fatalf("artifact metadata = %#v", artifacts[0])
	}
}

func TestListArtifactsAuthErrorDoesNotLeakCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	provider := New(WithConfig(Config{
		Endpoint: server.URL,
		Insecure: true,
		Username: "example-user",
		Password: "example-pass",
	}))
	_, err := provider.ListArtifacts(context.Background(), "registry.example.com/team/app")
	if err == nil {
		t.Fatal("expected auth error")
	}
	if strings.Contains(err.Error(), "example-user") || strings.Contains(err.Error(), "example-pass") {
		t.Fatalf("credential leaked in error: %v", err)
	}
}

func TestInspectReferenceUsesDomainParser(t *testing.T) {
	inspection, err := New().InspectReference(context.Background(), "localhost:30500/team/app:dev", domainartifact.ArtifactTypeImage)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if inspection.Reference.Registry != "localhost:30500" {
		t.Fatalf("inspection = %#v", inspection)
	}
}

func CredentialRefForTest() artifact.CredentialRef {
	return artifact.CredentialRef{ID: "secret-registry", SecretKey: "registry"}
}

type fakeSecretProvider struct {
	value []byte
}

func (f fakeSecretProvider) PutSecret(ctx context.Context, request portsecret.PutRequest) (domaincredential.SecretRef, error) {
	return request.Ref, ctx.Err()
}

func (f fakeSecretProvider) ValidateProvider(ctx context.Context) (portsecret.ProviderStatus, error) {
	return portsecret.ProviderStatus{Provider: "fake", Configured: true, Reachable: true}, ctx.Err()
}

func (f fakeSecretProvider) GetSecret(ctx context.Context, ref domaincredential.SecretRef) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]byte(nil), f.value...), nil
}

func (f fakeSecretProvider) DeleteSecret(ctx context.Context, ref domaincredential.SecretRef) error {
	return ctx.Err()
}

func (f fakeSecretProvider) RotateSecret(ctx context.Context, ref domaincredential.SecretRef, newValue []byte) (domaincredential.SecretRef, error) {
	return ref, ctx.Err()
}

func (f fakeSecretProvider) ListSecretRefs(ctx context.Context, scope portsecret.Scope) ([]domaincredential.SecretRef, error) {
	return nil, ctx.Err()
}

func (f fakeSecretProvider) RecordUsage(ctx context.Context, usage domaincredential.SecretUsage) error {
	return ctx.Err()
}
