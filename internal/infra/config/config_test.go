package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server.yaml")
	body := []byte(`
app:
  name: nivora-server
environment: test
http:
  bind_address: ":18080"
database:
  url: "postgres://nivora:nivora@localhost:5432/nivora?sslmode=disable"
event_bus:
  type: memory
object_store:
  type: local
  path: /tmp/nivora
log:
  level: debug
telemetry:
  enabled: false
auth:
  enabled: false
  mode: dev
  dev_user: local-admin
  static_token_env: NIVORA_AUTH_TOKEN
`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.App.Name != "nivora-server" {
		t.Fatalf("app name = %q", cfg.App.Name)
	}
	if cfg.HTTP.BindAddress != ":18080" {
		t.Fatalf("bind address = %q", cfg.HTTP.BindAddress)
	}
}

func TestRepositoryConfigExamplesValidate(t *testing.T) {
	paths := []string{
		"../../../configs/server.yaml",
		"../../../configs/worker.yaml",
		"../../../configs/runner.yaml",
		"../../../configs/docker-compose.server.yaml",
		"../../../configs/docker-compose.worker.yaml",
		"../../../configs/docker-compose.runner.yaml",
		"../../../configs/production.example.yaml",
	}
	for _, path := range paths {
		if _, err := Load(path); err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
	}
}

func TestProductionRejectsMemoryRuntimeStore(t *testing.T) {
	cfg := Default()
	cfg.Env = "production"
	cfg.Auth.Enabled = true
	cfg.Runtime.AllowLocalShellExecutor = false
	cfg.Database.RuntimeStore = "memory"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production memory runtime store to be rejected")
	}
}

func TestProductionRejectsUnsafeSecurityDefaults(t *testing.T) {
	base := Default()
	base.Env = "production"
	base.Database.RuntimeStore = "postgres"
	base.Auth.Enabled = true
	base.Runtime.AllowLocalShellExecutor = false

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"auth disabled", func(c *Config) { c.Auth.Enabled = false }},
		{"local shell", func(c *Config) { c.Runtime.AllowLocalShellExecutor = true }},
		{"privileged executor", func(c *Config) { c.Runtime.AllowPrivilegedExecutor = true }},
		{"remote host deploy", func(c *Config) { c.Runtime.AllowRemoteHostDeploy = true }},
		{"kubernetes apply", func(c *Config) { c.Runtime.AllowKubernetesApply = true }},
		{"argo sync", func(c *Config) { c.Runtime.AllowArgoSync = true }},
		{"insecure registry", func(c *Config) { c.Runtime.AllowInsecureRegistry = true }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected %s to be rejected", tc.name)
			}
		})
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("safe production config rejected: %v", err)
	}
}
