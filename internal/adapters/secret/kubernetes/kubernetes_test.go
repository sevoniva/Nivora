package kubernetes

import (
	"context"
	"testing"
)

func TestValidateProviderDoesNotRequireCluster(t *testing.T) {
	provider := New(Config{Namespace: "default"})
	status, err := provider.ValidateProvider(context.Background())
	if err != nil {
		t.Fatalf("validate provider: %v", err)
	}
	if status.Provider != "kubernetes" || !status.Configured || status.Reachable {
		t.Fatalf("unexpected kubernetes status: %#v", status)
	}
}
