package argocd

import (
	"context"
	"testing"

	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
)

func TestNoopProviderStatusAndGuardedSync(t *testing.T) {
	provider := NoopProvider{}
	status, err := provider.GetApplicationStatus(context.Background(), "demo")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if status.ApplicationName != "demo" {
		t.Fatalf("status = %#v", status)
	}
	result, err := provider.SyncApplication(context.Background(), portargocd.SyncRequest{ApplicationName: "demo"})
	if err != nil {
		t.Fatalf("sync skipped: %v", err)
	}
	if result.Requested {
		t.Fatal("sync should be skipped by default")
	}
	allowed, err := (NoopProvider{AllowSync: true}).SyncApplication(context.Background(), portargocd.SyncRequest{ApplicationName: "demo", AllowSync: true, Confirmed: true})
	if err != nil {
		t.Fatalf("sync allowed: %v", err)
	}
	if !allowed.Requested {
		t.Fatal("expected guarded noop sync request")
	}
}
