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
