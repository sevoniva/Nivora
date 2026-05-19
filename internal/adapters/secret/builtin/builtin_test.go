package builtin

import (
	"context"
	"testing"

	"github.com/sevoniva/nivora/internal/domain/credential"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
)

func TestStoreReturnsRefsWithoutValues(t *testing.T) {
	store := New()
	ref, err := store.PutSecret(context.Background(), portsecret.PutRequest{
		Ref:   credential.SecretRef{Name: "registry token", Key: "examples/registry/token", ScopeType: credential.ScopeProject},
		Value: []byte("sample-value-for-test-only"),
	})
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	value, err := store.GetSecret(context.Background(), ref)
	if err != nil {
		t.Fatalf("get secret: %v", err)
	}
	if string(value) != "sample-value-for-test-only" {
		t.Fatalf("unexpected secret value")
	}
	refs, err := store.ListSecretRefs(context.Background(), portsecret.Scope{ScopeType: credential.ScopeProject})
	if err != nil {
		t.Fatalf("list refs: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("refs len = %d", len(refs))
	}
	if refs[0].ID == "" || refs[0].Key == "" {
		t.Fatalf("expected metadata ref, got %#v", refs[0])
	}
}

func TestStoreDeleteRemovesSecret(t *testing.T) {
	store := New()
	ref, err := store.PutSecret(context.Background(), portsecret.PutRequest{Ref: credential.SecretRef{Name: "token", Key: "examples/token"}, Value: []byte("placeholder")})
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	if err := store.DeleteSecret(context.Background(), ref); err != nil {
		t.Fatalf("delete secret: %v", err)
	}
	if _, err := store.GetSecret(context.Background(), ref); err == nil {
		t.Fatalf("expected deleted secret to be unavailable")
	}
}

func TestStoreRotateUpdatesVersion(t *testing.T) {
	store := New()
	ref, err := store.PutSecret(context.Background(), portsecret.PutRequest{Ref: credential.SecretRef{Name: "token", Key: "examples/token"}, Value: []byte("old")})
	if err != nil {
		t.Fatalf("put secret: %v", err)
	}
	rotated, err := store.RotateSecret(context.Background(), ref, []byte("new"))
	if err != nil {
		t.Fatalf("rotate secret: %v", err)
	}
	if rotated.Version == "" || rotated.Version == ref.Version {
		t.Fatalf("expected version to change: before=%q after=%q", ref.Version, rotated.Version)
	}
	value, err := store.GetSecret(context.Background(), rotated)
	if err != nil {
		t.Fatalf("get rotated secret: %v", err)
	}
	if string(value) != "new" {
		t.Fatalf("unexpected rotated value")
	}
}

func TestStoreValidateProvider(t *testing.T) {
	store := New()
	status, err := store.ValidateProvider(context.Background())
	if err != nil {
		t.Fatalf("validate provider: %v", err)
	}
	if status.Provider != "builtin" || !status.Configured || !status.Reachable {
		t.Fatalf("unexpected status: %#v", status)
	}
}
