package plugin

import (
	"context"
	"errors"
	"testing"

	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
)

func TestDefaultRegistryListsBuiltInPlugins(t *testing.T) {
	registry := NewDefaultRegistry()
	plugins, err := registry.List(context.Background())
	if err != nil {
		t.Fatalf("list plugins: %v", err)
	}
	if len(plugins) == 0 {
		t.Fatal("expected built-in plugins")
	}
	found := false
	for _, item := range plugins {
		if item.Name == "executor-shell" {
			found = true
			if item.APIVersion != domainplugin.ManifestAPIVersion || item.Protocol != "builtin" || item.Compatibility.PluginAPIVersion != domainplugin.PluginAPIVersion || len(item.Capabilities) == 0 {
				t.Fatalf("unexpected shell plugin: %#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("executor-shell not found in %#v", plugins)
	}
}

func TestRegistryCapabilityMatching(t *testing.T) {
	registry := NewDefaultRegistry()
	matches, err := registry.MatchCapability(context.Background(), "artifact.resolve_digest")
	if err != nil {
		t.Fatalf("match capability: %v", err)
	}
	if len(matches) != 1 || matches[0].Plugin.Name != "artifact-oci" {
		t.Fatalf("matches = %#v", matches)
	}
}

func TestRegistryGetMissingPlugin(t *testing.T) {
	registry := NewDefaultRegistry()
	_, err := registry.Get(context.Background(), "missing")
	if !errors.Is(err, ErrPluginNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateManifestAcceptsCompatibleExternalPlugin(t *testing.T) {
	registry := NewDefaultRegistry()
	result, err := registry.Validate(context.Background(), domainplugin.Manifest{
		APIVersion: domainplugin.ManifestAPIVersion,
		Name:       "example-scanner",
		Type:       domainplugin.TypeScanner,
		Version:    "0.1.0",
		Protocol:   string(domainplugin.ProtocolHTTP),
		Endpoint:   "https://plugins.example.invalid/scanner",
		Capabilities: []domainplugin.Capability{{
			Name:        "security.scan",
			Description: "example scanner capability",
		}},
		Compatibility: domainplugin.Compatibility{
			PluginAPIVersion: domainplugin.PluginAPIVersion,
			NivoraMinVersion: "0.1.0-alpha.1",
			Protocols:        []string{string(domainplugin.ProtocolHTTP)},
		},
		Lifecycle: domainplugin.Lifecycle{Health: true, Capabilities: true, ValidateConfig: true, Execute: true},
	})
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !result.Valid || len(result.Errors) != 0 {
		t.Fatalf("result = %#v", result)
	}
}

func TestValidateManifestRejectsIncompatiblePluginAPI(t *testing.T) {
	result := ValidateManifest(domainplugin.Manifest{
		APIVersion: domainplugin.ManifestAPIVersion,
		Name:       "future",
		Type:       domainplugin.TypeArtifact,
		Version:    "1.0.0",
		Protocol:   string(domainplugin.ProtocolGRPC),
		Endpoint:   "dns:///future",
		Capabilities: []domainplugin.Capability{{
			Name: "artifact.inspect",
		}},
		Compatibility: domainplugin.Compatibility{PluginAPIVersion: "v9"},
	})
	if result.Valid {
		t.Fatalf("expected incompatible manifest to be invalid: %#v", result)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected validation errors: %#v", result)
	}
}

func TestValidateManifestRejectsFutureNivoraMinimum(t *testing.T) {
	result := ValidateManifest(domainplugin.Manifest{
		APIVersion: domainplugin.ManifestAPIVersion,
		Name:       "future",
		Type:       domainplugin.TypeExecutor,
		Version:    "1.0.0",
		Protocol:   string(domainplugin.ProtocolHTTP),
		Endpoint:   "https://plugins.example.invalid/future",
		Capabilities: []domainplugin.Capability{{
			Name: "executor.future",
		}},
		Compatibility: domainplugin.Compatibility{
			PluginAPIVersion: domainplugin.PluginAPIVersion,
			NivoraMinVersion: "99.0.0",
		},
	})
	if result.Valid {
		t.Fatalf("expected future minimum to be invalid: %#v", result)
	}
}
