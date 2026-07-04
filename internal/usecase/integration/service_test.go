package integration

import (
	"context"
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
