package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	apimcp "github.com/sevoniva/nivora/internal/api/mcp"
	appmcp "github.com/sevoniva/nivora/internal/app/mcp"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/spf13/cobra"
)

func newMCPCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve or inspect the local MCP control-plane foundation",
	}
	cmd.AddCommand(newMCPServeCommand())
	cmd.AddCommand(newMCPListToolsCommand())
	cmd.AddCommand(newMCPListResourcesCommand())
	return cmd
}

func newMCPServeCommand() *cobra.Command {
	var configPath string
	var stdio bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve MCP over stdio",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !stdio {
				return errUnsupportedMCPMode()
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
			return appmcp.RunStdio(ctx, configPath, cmd.InOrStdin(), cmd.OutOrStdout(), logger)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "configs/server.yaml", "config file path")
	cmd.Flags().BoolVar(&stdio, "stdio", true, "serve MCP over stdio")
	return cmd
}

func newMCPListToolsCommand() *cobra.Command {
	var configPath string
	var local bool
	cmd := &cobra.Command{
		Use:   "list-tools",
		Short: "List safe MCP tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, cleanup, err := buildMCPForCLI(cmd.Context(), configPath, local)
			if err != nil {
				return err
			}
			defer cleanup()
			tools, err := server.ListTools(cmd.Context())
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), tools)
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "configs/server.yaml", "config file path")
	cmd.Flags().BoolVar(&local, "local", false, "force local memory-backed MCP runtime")
	return cmd
}

func newMCPListResourcesCommand() *cobra.Command {
	var configPath string
	var local bool
	cmd := &cobra.Command{
		Use:   "list-resources",
		Short: "List MCP resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, cleanup, err := buildMCPForCLI(cmd.Context(), configPath, local)
			if err != nil {
				return err
			}
			defer cleanup()
			resources, err := server.ListResources(cmd.Context())
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), resources)
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "configs/server.yaml", "config file path")
	cmd.Flags().BoolVar(&local, "local", false, "force local memory-backed MCP runtime")
	return cmd
}

func buildMCPForCLI(ctx context.Context, configPath string, local bool) (*apimcp.Server, func(), error) {
	cfg, err := loadMCPConfig(configPath, local)
	if err != nil {
		return nil, nil, err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	return appmcp.BuildServer(ctx, cfg, logger)
}

func errUnsupportedMCPMode() error {
	return errors.New("only --stdio mode is supported in this MCP foundation")
}

func loadMCPConfig(configPath string, local bool) (config.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return config.Config{}, err
	}
	if local {
		cfg.Env = "local"
		cfg.Database.RuntimeStore = "memory"
		cfg.Auth.Enabled = false
		cfg.Auth.Mode = "dev"
		cfg.MCP.SubjectRole = "viewer"
	}
	return cfg, nil
}
