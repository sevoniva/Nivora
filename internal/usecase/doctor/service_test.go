package doctor

import (
	"path/filepath"
	"testing"

	"github.com/sevoniva/nivora/internal/infra/config"
)

func TestCheckConfigAcceptsSafeProductionProfile(t *testing.T) {
	cfg := safeProductionConfig()
	report := CheckConfig(cfg)
	if report.Status != StatusPass {
		t.Fatalf("report = %#v", report)
	}
}

func TestCheckConfigFailsUnsafeProductionDefaults(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*config.Config)
		wantID string
	}{
		{name: "memory store", wantID: "database.runtime_store", mutate: func(c *config.Config) { c.Database.RuntimeStore = "memory" }},
		{name: "auth disabled", wantID: "auth.enabled", mutate: func(c *config.Config) { c.Auth.Enabled = false }},
		{name: "dev auth", wantID: "auth.mode", mutate: func(c *config.Config) { c.Auth.Mode = "dev" }},
		{name: "local shell", wantID: "runtime.allow_local_shell_executor", mutate: func(c *config.Config) { c.Runtime.AllowLocalShellExecutor = true }},
		{name: "kubernetes apply", wantID: "runtime.allow_kubernetes_apply", mutate: func(c *config.Config) { c.Runtime.AllowKubernetesApply = true }},
		{name: "argo sync", wantID: "runtime.allow_argo_sync", mutate: func(c *config.Config) { c.Runtime.AllowArgoSync = true }},
		{name: "remote host", wantID: "runtime.allow_remote_host_deploy", mutate: func(c *config.Config) { c.Runtime.AllowRemoteHostDeploy = true }},
		{name: "insecure registry", wantID: "runtime.allow_insecure_registry", mutate: func(c *config.Config) { c.Runtime.AllowInsecureRegistry = true }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := safeProductionConfig()
			tc.mutate(&cfg)
			report := CheckConfig(cfg)
			if report.Status != StatusFail {
				t.Fatalf("expected fail report, got %#v", report)
			}
			if !hasCheckStatus(report, tc.wantID, StatusFail) {
				t.Fatalf("missing failed check %s in %#v", tc.wantID, report)
			}
		})
	}
}

func TestCheckConfigKeepsLocalDevAsWarning(t *testing.T) {
	report := CheckConfig(config.Default())
	if report.Status != StatusWarn {
		t.Fatalf("dev report status = %s, want WARN: %#v", report.Status, report)
	}
}

func TestCheckConfigFileRedactsSecretLikeEvidence(t *testing.T) {
	path := filepath.Join("..", "..", "..", "configs", "production.example.yaml")
	report, err := CheckConfigFile(path)
	if err != nil {
		t.Fatalf("check config file: %v", err)
	}
	for _, check := range report.Checks {
		if check.Evidence == "NIVORA_AUTH_TOKEN" {
			t.Fatalf("token-like evidence was not redacted: %#v", check)
		}
	}
}

func TestCheckConfigIncludesEnterpriseClosureChecks(t *testing.T) {
	report := CheckConfig(safeProductionConfig())
	for _, id := range []string{
		"audit.persistence",
		"event_outbox.persistence",
		"runner.identity_config",
		"secret_provider.posture",
		"runner.token_storage",
		"api.openapi_route_contract",
		"security.secret_scan",
	} {
		if !hasCheck(report, id) {
			t.Fatalf("missing doctor check %s in %#v", id, report)
		}
	}
	if !hasCheckStatus(report, "secret_provider.posture", StatusNotChecked) {
		t.Fatalf("secret provider posture should be explicit but not overclaimed: %#v", report)
	}
	if !hasCheckStatus(report, "api.openapi_route_contract", StatusNotChecked) {
		t.Fatalf("OpenAPI route contract should point to test verification: %#v", report)
	}
	if report.Status != StatusPass {
		t.Fatalf("not-checked live checks should not make safe profile fail: %#v", report)
	}
}

func safeProductionConfig() config.Config {
	cfg := config.Default()
	cfg.Env = "production"
	cfg.Database.RuntimeStore = "postgres"
	cfg.Database.URL = "postgres://nivora@postgres.example.internal:5432/nivora?sslmode=require"
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_AUTH_TOKEN"
	cfg.Runtime.AllowLocalShellExecutor = false
	cfg.Runtime.RunnerIsolationProfile = config.RunnerProfileContainer
	return cfg
}

func hasCheckStatus(report Report, id string, status string) bool {
	for _, check := range report.Checks {
		if check.ID == id && check.Status == status {
			return true
		}
	}
	return false
}

func hasCheck(report Report, id string) bool {
	for _, check := range report.Checks {
		if check.ID == id {
			return true
		}
	}
	return false
}
