package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
)

func TestServiceListBuildsIntegrationIndex(t *testing.T) {
	service := NewService(pluginusecase.NewDefaultRegistry())

	result, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list integrations: %v", err)
	}
	if result.Count == 0 || len(result.Integrations) == 0 {
		t.Fatalf("expected integration entries")
	}
	var foundArtifact bool
	for _, item := range result.Integrations {
		if item.Name == "artifact-oci" {
			foundArtifact = true
			if item.Type != "artifact" || item.Protocol != "builtin" || item.Maturity == "" {
				t.Fatalf("unexpected artifact integration = %#v", item)
			}
			if item.AdapterKind == "" || item.Boundary == "" || item.CredentialMode == "" || item.NetworkAccess == "" {
				t.Fatalf("artifact integration missing boundary fields = %#v", item)
			}
			if item.CredentialMode != "credential_ref_only" || item.NetworkAccess != "optional" {
				t.Fatalf("artifact integration credential/network boundary = %#v", item)
			}
			if !item.SafeByDefault || item.MutatesExternalSystems {
				t.Fatalf("artifact integration safety flags = %#v", item)
			}
		}
	}
	if !foundArtifact {
		t.Fatalf("artifact-oci integration not found: %#v", result.Integrations)
	}
}

func TestServiceListMarksSkeletonsHonestly(t *testing.T) {
	now := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	registry := pluginusecase.NewRegistry([]domainplugin.Manifest{{
		APIVersion: domainplugin.ManifestAPIVersion,
		Name:       "example-scm",
		Type:       domainplugin.TypeSCM,
		Version:    "0.1.0",
		Protocol:   string(domainplugin.ProtocolBuiltIn),
		Status:     domainplugin.StatusBuiltIn,
		Capabilities: []domainplugin.Capability{{
			Name:        "scm.placeholder",
			Description: "SCM provider adapter skeleton",
		}},
		Compatibility: domainplugin.Compatibility{PluginAPIVersion: domainplugin.PluginAPIVersion},
		CreatedAt:     now,
		UpdatedAt:     now,
	}})
	service := NewService(registry)

	result, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list integrations: %v", err)
	}
	if result.Integrations[0].Maturity != "placeholder" {
		t.Fatalf("maturity = %q", result.Integrations[0].Maturity)
	}
	if len(result.Integrations[0].Notes) < 2 {
		t.Fatalf("expected skeleton notes, got %#v", result.Integrations[0].Notes)
	}
}

func TestServiceListDefaultIntegrationsHaveExplicitSafeBoundaries(t *testing.T) {
	service := NewService(pluginusecase.NewDefaultRegistry())

	result, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list integrations: %v", err)
	}
	if result.Count < 10 {
		t.Fatalf("expected default integration catalog, got %#v", result)
	}
	for _, item := range result.Integrations {
		if item.AdapterKind == "" || item.Boundary == "" || item.CredentialMode == "" || item.NetworkAccess == "" {
			t.Fatalf("%s missing explicit boundary fields: %#v", item.Name, item)
		}
		if !item.SafeByDefault {
			t.Fatalf("%s must remain safe by default: %#v", item.Name, item)
		}
		if item.DefaultMutation {
			t.Fatalf("%s must not advertise default mutation: %#v", item.Name, item)
		}
		if item.MutatesExternalSystems {
			t.Fatalf("%s must not expose external mutation through built-in integration index: %#v", item.Name, item)
		}
		if item.NetworkAccess == "required" {
			t.Fatalf("%s must not require network access in baseline integration index", item.Name)
		}
		if leaksSensitiveBoundaryText(item) {
			t.Fatalf("%s integration metadata contains sensitive-looking text: %#v", item.Name, item)
		}
	}
}

func TestServiceListHighRiskIntegrationsRemainGuarded(t *testing.T) {
	service := NewService(pluginusecase.NewDefaultRegistry())
	result, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list integrations: %v", err)
	}
	items := map[string]Integration{}
	for _, item := range result.Integrations {
		items[item.Name] = item
	}

	for name, want := range map[string]struct {
		maturity       string
		adapterKind    string
		boundary       string
		credentialMode string
		networkAccess  string
	}{
		"artifact-oci":        {"partial", "foundation", "read-only", "credential_ref_only", "optional"},
		"executor-argocd":     {"experimental", "noop", "guarded-action", "credential_ref_only", "optional"},
		"executor-yaml-apply": {"experimental", "foundation", "guarded-action", "credential_ref_only", "optional"},
		"cloud-aws":           {"foundation", "skeleton", "metadata-only", "credential_ref_only", "none"},
		"scm-generic":         {"foundation", "skeleton", "metadata-only", "credential_ref_only", "none"},
		"scanner-noop":        {"foundation", "noop", "noop", "none", "none"},
	} {
		got, ok := items[name]
		if !ok {
			t.Fatalf("%s missing from integration catalog", name)
		}
		if got.Maturity != want.maturity || got.AdapterKind != want.adapterKind || got.Boundary != want.boundary || got.CredentialMode != want.credentialMode || got.NetworkAccess != want.networkAccess {
			t.Fatalf("%s boundary = %#v, want %#v", name, got, want)
		}
	}
}

func leaksSensitiveBoundaryText(item Integration) bool {
	text := item.Name + " " + item.Type + " " + item.Status + " " + item.Protocol + " " + item.Maturity + " " + item.AdapterKind + " " + item.Boundary + " " + item.CredentialMode + " " + item.NetworkAccess + " " + strings.Join(item.Notes, " ")
	for _, capability := range item.Capabilities {
		text += " " + capability.Name + " " + capability.Description
	}
	text = strings.ToLower(text)
	for _, marker := range []string{"password=", "token=", "authorization:", "private_key", "kubeconfig:", "secret=", "access_key="} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
