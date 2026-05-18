package oci

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
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
	if resolution.Digest != digest {
		t.Fatalf("digest = %q", resolution.Digest)
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

func TestInspectReferenceUsesDomainParser(t *testing.T) {
	inspection, err := New().InspectReference(context.Background(), "localhost:30500/team/app:dev", domainartifact.ArtifactTypeImage)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if inspection.Reference.Registry != "localhost:30500" {
		t.Fatalf("inspection = %#v", inspection)
	}
}
