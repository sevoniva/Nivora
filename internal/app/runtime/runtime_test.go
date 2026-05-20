package runtime

import (
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestMemoryStoreSelection(t *testing.T) {
	cfg := config.Default()
	// Default config uses memory runtime store
	if cfg.Database.RuntimeStore != "memory" {
		t.Fatalf("expected memory, got %s", cfg.Database.RuntimeStore)
	}
}

func TestProductionRejectsMemoryRuntimeStore(t *testing.T) {
	cfg := config.Default()
	cfg.Env = "production"
	cfg.Database.RuntimeStore = "memory"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for production with memory runtime store")
	}
}

func TestProductionRequiresAuth(t *testing.T) {
	cfg := config.Default()
	cfg.Env = "production"
	cfg.Database.RuntimeStore = "postgres"
	cfg.Auth.Enabled = false
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for production with auth disabled")
	}
}

func TestProductionRejectsUnsafeExecutors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
	}{
		{"local shell", func(c *config.Config) { c.Runtime.AllowLocalShellExecutor = true }},
		{"privileged", func(c *config.Config) { c.Runtime.AllowPrivilegedExecutor = true }},
		{"remote host", func(c *config.Config) { c.Runtime.AllowRemoteHostDeploy = true }},
		{"kubernetes apply", func(c *config.Config) { c.Runtime.AllowKubernetesApply = true }},
		{"argo sync", func(c *config.Config) { c.Runtime.AllowArgoSync = true }},
		{"insecure registry", func(c *config.Config) { c.Runtime.AllowInsecureRegistry = true }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Env = "production"
			cfg.Database.RuntimeStore = "postgres"
			cfg.Auth.Enabled = true
			cfg.Auth.Mode = "token"
			cfg.Auth.StaticTokenEnv = "NIVORA_AUTH_TOKEN"
			tt.mutate(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected error for %s=true in production", tt.name)
			}
		})
	}
}

func TestProductionRejectsInlineDBPassword(t *testing.T) {
	cfg := config.Default()
	cfg.Env = "production"
	cfg.Database.RuntimeStore = "postgres"
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_AUTH_TOKEN"
	cfg.Database.URL = "postgres://user:password123@localhost:5432/db?sslmode=require"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for inline database password in production")
	}
}

func TestProductionValidConfigPasses(t *testing.T) {
	cfg := config.Default()
	cfg.Env = "production"
	cfg.Database.RuntimeStore = "postgres"
	cfg.Database.URL = "postgres://nivora@localhost:5432/nivora?sslmode=require"
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_AUTH_TOKEN"
	cfg.Runtime.AllowLocalShellExecutor = false
	cfg.Runtime.AllowPrivilegedExecutor = false
	cfg.Runtime.AllowRemoteHostDeploy = false
	cfg.Runtime.AllowKubernetesApply = false
	cfg.Runtime.AllowArgoSync = false
	cfg.Runtime.AllowInsecureRegistry = false
	cfg.Runtime.RunnerIsolationProfile = "container-isolated"
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected valid production config to pass, got: %v", err)
	}
}

func TestDevConfigWithMemoryStorePasses(t *testing.T) {
	cfg := config.Default()
	cfg.Env = "development"
	cfg.Database.RuntimeStore = "memory"
	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected dev config to pass, got: %v", err)
	}
}

func TestInvalidRuntimeStoreRejected(t *testing.T) {
	cfg := config.Default()
	cfg.Database.RuntimeStore = "mongodb"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for invalid runtime store")
	}
}
