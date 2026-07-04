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
	base.Database.URL = "postgres://nivora@postgres.example.internal:5432/nivora?sslmode=require"
	base.Auth.Enabled = true
	base.Auth.Mode = "token"
	base.Runtime.AllowLocalShellExecutor = false
	base.Runtime.RunnerIsolationProfile = RunnerProfileContainer

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"auth disabled", func(c *Config) { c.Auth.Enabled = false }},
		{"dev auth mode", func(c *Config) { c.Auth.Mode = "dev" }},
		{"missing token env", func(c *Config) { c.Auth.StaticTokenEnv = "" }},
		{"inline database password", func(c *Config) {
			c.Database.URL = "postgres://nivora:secret@postgres.example.internal:5432/nivora?sslmode=require"
		}},
		{"local shell", func(c *Config) { c.Runtime.AllowLocalShellExecutor = true }},
		{"privileged executor", func(c *Config) { c.Runtime.AllowPrivilegedExecutor = true }},
		{"remote host deploy", func(c *Config) { c.Runtime.AllowRemoteHostDeploy = true }},
		{"kubernetes apply", func(c *Config) { c.Runtime.AllowKubernetesApply = true }},
		{"argo sync", func(c *Config) { c.Runtime.AllowArgoSync = true }},
		{"insecure registry", func(c *Config) { c.Runtime.AllowInsecureRegistry = true }},
		{"mcp action tools", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.AllowActionTools = true
		}},
		{"mcp not readonly", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.ReadOnly = false
		}},
		{"mcp unsupported mode", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.Mode = "http"
		}},
		{"mcp missing token env", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.TokenEnv = ""
		}},
		{"mcp missing request timeout", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.RequestTimeout = ""
		}},
		{"mcp missing response cap", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.MaxResponseBytes = 0
		}},
		{"mcp missing request cap", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.MaxRequestBytes = 0
		}},
		{"mcp missing rate limit", func(c *Config) {
			c.MCP.Enabled = true
			c.MCP.MaxRequestsPerMinute = 0
		}},
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

func TestMCPConfigRejectsInvalidLimits(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"negative max response bytes", func(c *Config) { c.MCP.MaxResponseBytes = -1 }},
		{"negative max request bytes", func(c *Config) { c.MCP.MaxRequestBytes = -1 }},
		{"negative max requests per minute", func(c *Config) { c.MCP.MaxRequestsPerMinute = -1 }},
		{"invalid request timeout", func(c *Config) { c.MCP.RequestTimeout = "soon" }},
		{"zero request timeout", func(c *Config) { c.MCP.RequestTimeout = "0s" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			tc.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected %s to be rejected", tc.name)
			}
		})
	}
}
