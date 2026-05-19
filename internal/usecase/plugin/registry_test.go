package plugin

import (
	"context"
	"errors"
	"testing"
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
			if item.Protocol != "builtin" || len(item.Capabilities) == 0 {
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
