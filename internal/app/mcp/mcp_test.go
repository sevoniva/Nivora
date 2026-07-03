package mcp

import (
	"context"
	"strings"
	"testing"

	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

func TestResolveSubjectLocalDefaultsToViewer(t *testing.T) {
	cfg := config.Default()
	subject, err := ResolveSubject(context.Background(), cfg, authusecase.NewService(authusecase.NewMemoryStore(), nil))
	if err != nil {
		t.Fatalf("ResolveSubject: %v", err)
	}
	if subject.AuthMode != "mcp-local" || len(subject.Roles) != 1 || subject.Roles[0] != domainauth.RoleViewer {
		t.Fatalf("subject = %#v", subject)
	}
}

func TestResolveSubjectProductionRequiresToken(t *testing.T) {
	cfg := productionMCPConfig()
	_, err := ResolveSubject(context.Background(), cfg, authusecase.NewService(authusecase.NewMemoryStore(), nil))
	if err == nil || !strings.Contains(err.Error(), "required in production") {
		t.Fatalf("expected production token error, got %v", err)
	}
}

func TestResolveSubjectRejectsRunnerToken(t *testing.T) {
	cfg := config.Default()
	cfg.MCP.TokenEnv = "NIVORA_TEST_MCP_TOKEN"
	t.Setenv("NIVORA_TEST_MCP_TOKEN", "nvr_runner_deadbeef")
	_, err := ResolveSubject(context.Background(), cfg, authusecase.NewService(authusecase.NewMemoryStore(), nil))
	if err == nil || !strings.Contains(err.Error(), "runner tokens cannot authenticate") {
		t.Fatalf("expected runner token denial, got %v", err)
	}
}

func TestResolveSubjectUsesConfiguredStaticToken(t *testing.T) {
	cfg := config.Default()
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_TEST_AUTH_TOKEN"
	cfg.MCP.TokenEnv = "NIVORA_TEST_MCP_TOKEN"
	t.Setenv("NIVORA_TEST_AUTH_TOKEN", "local-static-token")
	t.Setenv("NIVORA_TEST_MCP_TOKEN", "local-static-token")
	subject, err := ResolveSubject(context.Background(), cfg, authusecase.NewService(authusecase.NewMemoryStore(), nil))
	if err != nil {
		t.Fatalf("ResolveSubject: %v", err)
	}
	if subject.AuthMode != "token" || subject.ID != "service-account" || len(subject.Roles) == 0 || subject.Roles[0] != domainauth.RoleOwner {
		t.Fatalf("subject = %#v", subject)
	}
}

func TestBuildServerLocalConfig(t *testing.T) {
	cfg := config.Default()
	server, cleanup, err := BuildServer(context.Background(), cfg, nil)
	if err != nil {
		t.Fatalf("BuildServer: %v", err)
	}
	defer cleanup()
	tools, err := server.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatalf("expected tools")
	}
}

func productionMCPConfig() config.Config {
	cfg := config.Default()
	cfg.Env = "production"
	cfg.Auth.Enabled = true
	cfg.Auth.Mode = "token"
	cfg.Auth.StaticTokenEnv = "NIVORA_TEST_AUTH_TOKEN"
	cfg.Database.RuntimeStore = "postgres"
	cfg.Runtime.AllowLocalShellExecutor = false
	cfg.Runtime.RunnerIsolationProfile = config.RunnerProfileContainer
	cfg.MCP.Enabled = true
	cfg.MCP.TokenEnv = "NIVORA_TEST_MCP_TOKEN"
	return cfg
}
