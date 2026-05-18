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
	resources, err := provider.GetApplicationResources(context.Background(), "demo")
	if err != nil || len(resources) != 1 {
		t.Fatalf("resources=%#v err=%v", resources, err)
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
	watch, err := (NoopProvider{AllowSync: true}).WatchApplicationStatus(context.Background(), "demo", 1)
	if err != nil || len(watch) != 2 || watch[1].HealthStatus != "Healthy" {
		t.Fatalf("watch=%#v err=%v", watch, err)
	}
	_, err = (NoopProvider{AllowSync: true}).SyncApplication(context.Background(), portargocd.SyncRequest{ApplicationName: "demo", AllowSync: true, Confirmed: true, Force: true})
	if err == nil {
		t.Fatal("expected force sync rejection")
	}
}
