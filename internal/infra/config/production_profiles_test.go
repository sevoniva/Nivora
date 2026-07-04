package config

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHelmProductionValuesAvoidUnsafeDefaults(t *testing.T) {
	body, err := os.ReadFile("../../../deployments/helm/values-production.yaml")
	if err != nil {
		t.Fatalf("read production values: %v", err)
	}
	var values map[string]any
	if err := yaml.Unmarshal(body, &values); err != nil {
		t.Fatalf("decode production values: %v", err)
	}
	configValues := nestedMap(t, values, "config")
	if got := stringValue(configValues, "environment"); got != "production" {
		t.Fatalf("environment = %q, want production", got)
	}
	if got := stringValue(configValues, "runtimeStore"); got != "postgres" {
		t.Fatalf("runtimeStore = %q, want postgres", got)
	}
	authValues := nestedMap(t, configValues, "auth")
	if got := boolValue(authValues, "enabled"); !got {
		t.Fatal("auth must be enabled in production values")
	}
	mcpValues := nestedMap(t, configValues, "mcp")
	if boolValue(mcpValues, "enabled") {
		t.Fatal("MCP should be disabled by default in production values")
	}
	if boolValue(mcpValues, "allowActionTools") {
		t.Fatal("MCP action tools must be disabled in production values")
	}
	if got := stringValue(mcpValues, "mode"); got != "stdio" {
		t.Fatalf("MCP mode = %q, want stdio", got)
	}
	if got := stringValue(mcpValues, "requestTimeout"); got == "" {
		t.Fatal("MCP requestTimeout must be set in production values")
	}
	if got := intValue(mcpValues, "maxResponseBytes"); got <= 0 {
		t.Fatalf("MCP maxResponseBytes = %d, want positive", got)
	}
	if got := intValue(mcpValues, "maxRequestBytes"); got <= 0 {
		t.Fatalf("MCP maxRequestBytes = %d, want positive", got)
	}
	if got := intValue(mcpValues, "maxRequestsPerMinute"); got <= 0 {
		t.Fatalf("MCP maxRequestsPerMinute = %d, want positive", got)
	}
	runtimeValues := nestedMap(t, configValues, "runtime")
	for _, key := range []string{
		"allowLocalShellExecutor",
		"allowPrivilegedExecutor",
		"allowRemoteHostDeploy",
		"allowKubernetesApply",
		"allowArgoSync",
		"allowInsecureRegistry",
	} {
		if boolValue(runtimeValues, key) {
			t.Fatalf("%s must be false in production values", key)
		}
	}
	secretValues := nestedMap(t, values, "secret")
	if boolValue(secretValues, "create") {
		t.Fatal("production values should reference an existing secret instead of rendering a placeholder secret")
	}
	if got := stringValue(secretValues, "existingName"); got == "" {
		t.Fatal("production values must name an existing secret")
	}
}

func TestComposeProductionProfileAvoidsUnsafeDefaults(t *testing.T) {
	body, err := os.ReadFile("../../../deployments/docker-compose/docker-compose.production.example.yaml")
	if err != nil {
		t.Fatalf("read production compose profile: %v", err)
	}
	text := string(body)
	for _, required := range []string{"NIVORA_PRODUCTION_CONFIG", "NIVORA_AUTH_TOKEN", "NIVORA_POSTGRES_PASSWORD"} {
		if !strings.Contains(text, required) {
			t.Fatalf("production compose profile missing %s", required)
		}
	}
	for _, forbidden := range []string{"POSTGRES_HOST_AUTH_METHOD: trust", "runtime_store: memory", "auth: disabled"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("production compose profile contains unsafe value %q", forbidden)
		}
	}
}

func TestProductionConfigRejectsMemoryStore(t *testing.T) {
	cfg := Default()
	cfg.Env = "production"
	cfg.Database.RuntimeStore = "memory"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected production to reject memory runtime store")
	}
}

func TestProductionConfigRejectsDevAuth(t *testing.T) {
	cfg := Default()
	cfg.Env = "production"
	cfg.Database.RuntimeStore = "postgres"
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "dev"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected production to reject dev auth mode")
	}
}

func TestOIDCConfigRequiresIssuerAndClientID(t *testing.T) {
	cfg := Default()
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "oidc"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected OIDC to require issuer/client_id")
	}
}

func TestDefaultConfigIsSafeForDev(t *testing.T) {
	cfg := Default()
	if cfg.Env != "local" {
		t.Fatalf("expected env=local, got %s", cfg.Env)
	}
	if cfg.Database.RuntimeStore != "memory" {
		t.Fatalf("expected runtime_store=memory, got %s", cfg.Database.RuntimeStore)
	}
	if cfg.Auth.Enabled {
		t.Fatal("expected auth disabled by default")
	}
	if !cfg.Runtime.AllowLocalShellExecutor {
		t.Fatal("expected local shell allowed in default dev config")
	}
}

func nestedMap(t *testing.T, values map[string]any, key string) map[string]any {
	t.Helper()
	raw, ok := values[key]
	if !ok {
		t.Fatalf("missing key %s", key)
	}
	out, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("key %s has type %T, want map", key, raw)
	}
	return out
}

func stringValue(values map[string]any, key string) string {
	raw, _ := values[key]
	value, _ := raw.(string)
	return value
}

func boolValue(values map[string]any, key string) bool {
	raw, _ := values[key]
	value, _ := raw.(bool)
	return value
}

func intValue(values map[string]any, key string) int {
	raw, _ := values[key]
	switch value := raw.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}
