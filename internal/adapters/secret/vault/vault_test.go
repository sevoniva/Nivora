package vault

import (
	"context"
	"testing"
)

func TestValidateProviderDoesNotRequireVault(t *testing.T) {
	provider := New(Config{Address: "https://vault.example", Mount: "secret"})
	status, err := provider.ValidateProvider(context.Background())
	if err != nil {
		t.Fatalf("validate provider: %v", err)
	}
	if status.Provider != "vault" || !status.Configured || status.Reachable {
		t.Fatalf("unexpected vault status: %#v", status)
	}
}
