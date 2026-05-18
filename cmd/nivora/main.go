package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sevoniva/nivora/internal/app/runner"
	"github.com/sevoniva/nivora/internal/app/server"
	"github.com/sevoniva/nivora/internal/app/worker"
	"github.com/sevoniva/nivora/internal/infra/config"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	"github.com/sevoniva/nivora/internal/version"
	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "nivora",
		Short: "Nivora DevOps delivery control plane",
	}

	root.AddCommand(newVersionCommand())
	root.AddCommand(newRunCommand("server", "configs/server.yaml", server.Run))
	root.AddCommand(newRunCommand("worker", "configs/worker.yaml", worker.Run))
	root.AddCommand(newRunCommand("runner", "configs/runner.yaml", runner.Run))
	root.AddCommand(newConfigCommand())
	root.AddCommand(newPipelineCommand())
	return root
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.Current()
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s (commit %s, date %s)\n", info.Name, info.Version, info.Commit, info.Date)
		},
	}
}

func newRunCommand(name string, defaultConfig string, run func(context.Context, string) error) *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   name,
		Short: "Run nivora-" + name,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return run(ctx, configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", defaultConfig, "config file path")
	return cmd
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Config utilities",
	}

	var file string
	validate := &cobra.Command{
		Use:   "validate",
		Short: "Validate a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := config.Load(file); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "config %s is valid\n", file)
			return nil
		},
	}
	validate.Flags().StringVar(&file, "file", "configs/server.yaml", "config file to validate")
	cmd.AddCommand(validate)
	return cmd
}

func newPipelineCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Pipeline utilities",
	}
	cmd.AddCommand(newPipelineRunCommand())
	return cmd
}

func newPipelineRunCommand() *cobra.Command {
	var local bool
	var printLogs bool
	cmd := &cobra.Command{
		Use:   "run --local <pipeline.yaml>",
		Short: "Run a pipeline definition locally with the Phase 1 shell runtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !local {
				return fmt.Errorf("only --local pipeline execution is supported in Phase 1")
			}
			def, err := pipelineusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			result, err := server.NewPipelineService().CreateAndRun(cmd.Context(), pipelineusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "PipelineRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			if result.Record.Run.FailureReason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Failure: %s\n", result.Record.Run.FailureReason)
			}
			if printLogs {
				for _, log := range result.Record.Logs {
					fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s", log.Stream, log.Content)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "run with the in-process Phase 1 local runtime")
	cmd.Flags().BoolVar(&printLogs, "logs", true, "print captured logs")
	return cmd
}
