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
