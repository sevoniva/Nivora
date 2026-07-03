package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	apimcp "github.com/sevoniva/nivora/internal/api/mcp"
	"github.com/sevoniva/nivora/internal/app/runtime"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

func RunStdio(ctx context.Context, configPath string, in io.Reader, out io.Writer, logger *slog.Logger) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	if isProduction(cfg) && !cfg.MCP.Enabled {
		return errors.New("mcp.enabled=true is required to run MCP in production")
	}
	server, cleanup, err := BuildServer(ctx, cfg, logger)
	if err != nil {
		return err
	}
	defer cleanup()
	return server.ServeStdio(ctx, in, out)
}

func BuildServer(ctx context.Context, cfg config.Config, logger *slog.Logger) (*apimcp.Server, func(), error) {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	var cleanups []func()
	cleanup := func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}
	add := func(closeFn func()) {
		if closeFn != nil {
			cleanups = append(cleanups, closeFn)
		}
	}

	pipelines, closePipelines, err := runtime.NewPipelineServiceWithConfig(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closePipelines)
	deployments, closeDeployments, err := runtime.NewDeploymentServiceWithConfig(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closeDeployments)
	artifacts, closeArtifacts, err := runtime.NewArtifactServiceWithConfig(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closeArtifacts)
	releases, closeReleases, err := runtime.NewReleaseOrchestrationServiceWithConfig(ctx, cfg, artifacts, deployments)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closeReleases)
	security, closeSecurity, err := runtime.NewSecurityServiceWithConfig(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closeSecurity)
	approval, closeApproval, err := runtime.NewApprovalServiceWithConfig(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closeApproval)
	compliance, closeCompliance, err := runtime.NewComplianceServiceWithConfig(ctx, cfg, pipelines, deployments, artifacts, releases, security, approval)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closeCompliance)
	authService, closeAuth, err := runtime.NewAuthServiceWithConfig(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	add(closeAuth)
	subject, err := ResolveSubject(ctx, cfg, authService)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	server := apimcp.NewServer(apimcp.Services{
		Config:      cfg,
		Subject:     subject,
		Auth:        authService,
		Pipelines:   pipelines,
		Deployments: deployments,
		Artifacts:   artifacts,
		Releases:    releases,
		Security:    security,
		Compliance:  compliance,
		Plugins:     runtime.NewPluginRegistry(),
		Audit:       &apimcp.MemoryAuditRecorder{},
	}, logger)
	return server, cleanup, nil
}

func ResolveSubject(ctx context.Context, cfg config.Config, authService *authusecase.Service) (domainauth.Subject, error) {
	token := ""
	if cfg.MCP.TokenEnv != "" {
		token = os.Getenv(cfg.MCP.TokenEnv)
	}
	if strings.HasPrefix(token, "nvr_runner_") {
		return domainauth.Subject{}, errors.New("runner tokens cannot authenticate to MCP")
	}
	if token != "" {
		mode := cfg.Auth.Mode
		if mode == "" || mode == "dev" || mode == "disabled" {
			mode = "token"
		}
		staticToken := ""
		if cfg.Auth.StaticTokenEnv != "" {
			staticToken = os.Getenv(cfg.Auth.StaticTokenEnv)
		}
		return authService.Authenticate(ctx, authusecase.AuthenticateInput{
			Mode:         mode,
			Token:        token,
			StaticToken:  staticToken,
			DevUser:      cfg.Auth.DevUser,
			OIDCIssuer:   firstNonEmpty(cfg.Auth.OIDC.Issuer, cfg.Auth.Issuer),
			OIDCAudience: cfg.Auth.OIDC.ClientID,
		})
	}
	if isProduction(cfg) {
		return domainauth.Subject{}, fmt.Errorf("MCP token env %s is required in production", cfg.MCP.TokenEnv)
	}
	role := firstNonEmpty(cfg.MCP.SubjectRole, domainauth.RoleViewer)
	id := firstNonEmpty(cfg.MCP.SubjectID, "mcp-local-"+role)
	return domainauth.Subject{
		ID:          id,
		Username:    id,
		DisplayName: id,
		Roles:       []string{role},
		AuthMode:    "mcp-local",
	}, nil
}

func isProduction(cfg config.Config) bool {
	return cfg.Env == "production" || cfg.Env == "prod"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
