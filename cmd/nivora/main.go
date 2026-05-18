package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sevoniva/nivora/internal/app/runner"
	"github.com/sevoniva/nivora/internal/app/server"
	"github.com/sevoniva/nivora/internal/app/worker"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	"github.com/sevoniva/nivora/internal/infra/config"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
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
	root.AddCommand(newConfigCommand())
	root.AddCommand(newPipelineCommand())
	root.AddCommand(newArtifactCommand())
	root.AddCommand(newReleaseCommand())
	root.AddCommand(newDeploymentCommand())
	root.AddCommand(newGitOpsCommand())
	root.AddCommand(newArgoCDCommand())
	root.AddCommand(newRunnerCommand())
	return root
}

func newArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "artifact", Short: "Artifact utilities"}
	cmd.AddCommand(newArtifactInspectCommand())
	cmd.AddCommand(newArtifactResolveCommand())
	return cmd
}

func newArtifactInspectCommand() *cobra.Command {
	var artifactType string
	cmd := &cobra.Command{
		Use:   "inspect <reference>",
		Short: "Inspect and normalize an artifact reference locally",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := server.NewArtifactService().Inspect(cmd.Context(), args[0], domainartifact.ArtifactType(artifactType))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().StringVar(&artifactType, "type", "image", "artifact type")
	return cmd
}

func newArtifactResolveCommand() *cobra.Command {
	var artifactType string
	cmd := &cobra.Command{
		Use:   "resolve <reference>",
		Short: "Resolve artifact identity locally when already immutable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := server.NewArtifactService().Resolve(cmd.Context(), args[0], domainartifact.ArtifactType(artifactType))
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().StringVar(&artifactType, "type", "image", "artifact type")
	return cmd
}

func newReleaseCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "release", Short: "Release utilities"}
	cmd.AddCommand(newReleaseCreateCommand())
	cmd.AddCommand(newReleaseGetCommand())
	cmd.AddCommand(newReleaseArtifactsCommand())
	return cmd
}

func newReleaseCreateCommand() *cobra.Command {
	var file string
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "create --file <release.yaml>",
		Short: "Create a release and bind artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			def, err := artifactusecase.LoadReleaseDefinitionFile(file)
			if err != nil {
				return err
			}
			if !local {
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/releases", body)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), payload)
				return nil
			}
			record, err := server.NewArtifactService().CreateRelease(cmd.Context(), artifactusecase.CreateReleaseInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Release: %s\n", record.Release.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", record.Release.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Artifacts: %d\n", len(record.Bindings))
			if len(record.Warnings) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Warnings: %d\n", len(record.Warnings))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "release definition file")
	cmd.Flags().BoolVar(&local, "local", true, "create release in the in-process local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL for --local=false")
	return cmd
}

func newReleaseGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <release-id>",
		Short: "Get a release from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/releases/"+args[0], nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newReleaseArtifactsCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "artifacts <release-id>",
		Short: "List release artifacts from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/releases/"+args[0]+"/artifacts", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
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
	cmd.AddCommand(newPipelineGetCommand())
	cmd.AddCommand(newPipelineInspectCommand("logs", "Get PipelineRun logs", "/logs"))
	cmd.AddCommand(newPipelineInspectCommand("events", "Get PipelineRun events", "/events"))
	cmd.AddCommand(newPipelineInspectCommand("timeline", "Get PipelineRun timeline", "/timeline"))
	cmd.AddCommand(newPipelineCancelCommand())
	return cmd
}

func newPipelineRunCommand() *cobra.Command {
	var local bool
	var printLogs bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "run --local <pipeline.yaml>",
		Short: "Run a pipeline definition locally or against a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := pipelineusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			if !local {
				if serverURL == "" {
					return fmt.Errorf("--server is required when --local=false")
				}
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/pipeline-runs", body)
				if err != nil {
					return err
				}
				printPipelineRunSummary(cmd.OutOrStdout(), payload)
				if printLogs {
					printLogSummary(cmd.OutOrStdout(), payload)
				}
				return nil
			}
			started := time.Now()
			result, err := server.NewPipelineService().CreateAndRun(cmd.Context(), pipelineusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "PipelineRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			fmt.Fprintf(cmd.OutOrStdout(), "Logs: %d chunk(s)\n", len(result.Record.Logs))
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
	cmd.Flags().StringVar(&serverURL, "server", "", "Nivora server URL for --local=false")
	return cmd
}

func newPipelineGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <pipeline-run-id>",
		Short: "Get a PipelineRun from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/pipeline-runs/"+args[0], nil)
			if err != nil {
				return err
			}
			printPipelineRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPipelineInspectCommand(name string, short string, suffix string) *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   name + " <pipeline-run-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/pipeline-runs/"+args[0]+suffix, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newPipelineCancelCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "cancel <pipeline-run-id>",
		Short: "Cancel a PipelineRun on a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/pipeline-runs/"+args[0]+"/cancel", nil)
			if err != nil {
				return err
			}
			printPipelineRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployment",
		Short: "DeploymentRun utilities",
	}
	cmd.AddCommand(newDeploymentPlanCommand())
	cmd.AddCommand(newDeploymentRunCommand())
	cmd.AddCommand(newDeploymentDryRunCommand())
	cmd.AddCommand(newDeploymentApplyCommand())
	cmd.AddCommand(newDeploymentGetCommand())
	cmd.AddCommand(newDeploymentLocalInspectCommand("health", "Get DeploymentRun health", "/health", func(record deploymentusecase.RunRecord) any { return record.Health }))
	cmd.AddCommand(newDeploymentLocalInspectCommand("diff", "Get DeploymentRun diff", "/diff", func(record deploymentusecase.RunRecord) any { return record.Diff }))
	cmd.AddCommand(newDeploymentLocalInspectCommand("snapshot", "Get DeploymentRun manifest snapshot", "/manifest-snapshot", func(record deploymentusecase.RunRecord) any { return record.Snapshot }))
	cmd.AddCommand(newDeploymentLocalInspectCommand("rollback-plan", "Get DeploymentRun rollback plan", "/rollback-plan", func(record deploymentusecase.RunRecord) any { return record.RollbackPlan }))
	cmd.AddCommand(newDeploymentInspectCommand("resources", "Get DeploymentRun resources", "/resources"))
	cmd.AddCommand(newDeploymentInspectCommand("logs", "Get DeploymentRun logs", "/logs"))
	cmd.AddCommand(newDeploymentInspectCommand("events", "Get DeploymentRun events", "/events"))
	cmd.AddCommand(newDeploymentInspectCommand("timeline", "Get DeploymentRun timeline", "/timeline"))
	cmd.AddCommand(newDeploymentCancelCommand())
	return cmd
}

func newDeploymentLocalInspectCommand(name string, short string, suffix string, selector func(deploymentusecase.RunRecord) any) *cobra.Command {
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   name + " [--local <deployment.yaml> | <deployment-run-id>]",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if local {
				def, err := deploymentusecase.LoadDefinitionFile(args[0])
				if err != nil {
					return err
				}
				result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), selector(result.Record))
				return nil
			}
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/deployments/"+args[0]+suffix, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", false, "evaluate a local deployment definition instead of querying a server")
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentPlanCommand() *cobra.Command {
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "plan --local <deployment.yaml>",
		Short: "Render and plan a YAML DeploymentRun locally or against a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			if !local {
				if serverURL == "" {
					return fmt.Errorf("--server is required when --local=false")
				}
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/plan", body)
				if err != nil {
					return err
				}
				printJSON(cmd.OutOrStdout(), payload)
				return nil
			}
			result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printDeploymentPlanSummary(cmd.OutOrStdout(), result.Record.Plan)
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "plan with the in-process Phase 2.2 local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "", "Nivora server URL for --local=false")
	return cmd
}

func newDeploymentRunCommand() *cobra.Command {
	var local bool
	var serverURL string
	cmd := &cobra.Command{
		Use:   "run --local <deployment.yaml>",
		Short: "Run a non-destructive YAML DeploymentRun locally or against a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			if !local {
				if serverURL == "" {
					return fmt.Errorf("--server is required when --local=false")
				}
				body, err := json.Marshal(def)
				if err != nil {
					return err
				}
				payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments", body)
				if err != nil {
					return err
				}
				printDeploymentRunSummary(cmd.OutOrStdout(), payload)
				return nil
			}
			started := time.Now()
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			fmt.Fprintf(cmd.OutOrStdout(), "Manifests: %d\n", result.Record.Plan.ManifestCount)
			fmt.Fprintf(cmd.OutOrStdout(), "Logs: %d chunk(s)\n", len(result.Record.Logs))
			if result.Record.Run.Reason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Reason: %s\n", result.Record.Run.Reason)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "run with the in-process Phase 2.2 local runtime")
	cmd.Flags().StringVar(&serverURL, "server", "", "Nivora server URL for --local=false")
	return cmd
}

func newDeploymentDryRunCommand() *cobra.Command {
	cmd := newDeploymentRunCommand()
	cmd.Use = "dry-run --local <deployment.yaml>"
	cmd.Short = "Run a non-destructive YAML DeploymentRun dry-run"
	return cmd
}

func newDeploymentApplyCommand() *cobra.Command {
	var local bool
	var confirm bool
	cmd := &cobra.Command{
		Use:   "apply --local <deployment.yaml> --confirm",
		Short: "Run an explicit local YAML apply through the configured manifest client",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !local {
				return fmt.Errorf("server-backed deployment apply is not implemented; use --local for Phase 2.2")
			}
			if !confirm {
				return fmt.Errorf("deployment apply requires --confirm")
			}
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.Options.Apply = true
			def.Spec.Options.DryRun = false
			started := time.Now()
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, AllowApply: true})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Duration: %s\n", time.Since(started).Round(time.Millisecond))
			fmt.Fprintf(cmd.OutOrStdout(), "Apply: %s\n", result.Record.Apply.Message)
			if result.Record.Rollout.Message != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Rollout: %s\n", result.Record.Rollout.Message)
			}
			if result.Record.Run.Reason != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Reason: %s\n", result.Record.Run.Reason)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&local, "local", true, "apply with the in-process Phase 2.2 local runtime")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm explicit local apply")
	return cmd
}

func newDeploymentGetCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "get <deployment-run-id>",
		Short: "Get a DeploymentRun from a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/deployments/"+args[0], nil)
			if err != nil {
				return err
			}
			printDeploymentRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentInspectCommand(name string, short string, suffix string) *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   name + " <deployment-run-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/deployments/"+args[0]+suffix, nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newDeploymentCancelCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "cancel <deployment-run-id>",
		Short: "Cancel a DeploymentRun on a Nivora server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/deployments/"+args[0]+"/cancel", nil)
			if err != nil {
				return err
			}
			printDeploymentRunSummary(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newGitOpsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "gitops", Short: "GitOps planning utilities"}
	cmd.AddCommand(newGitOpsPlanCommand())
	cmd.AddCommand(newGitOpsDiffCommand())
	cmd.AddCommand(newGitOpsWriteCommand())
	return cmd
}

func newGitOpsPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan --local <deployment.yaml>",
		Short: "Build a local GitOps change plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printGitOpsPlanSummary(cmd.OutOrStdout(), result.Record.GitOpsPlan)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 2.3 local runtime")
	return cmd
}

func newGitOpsDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff --local <deployment.yaml>",
		Short: "Build a local GitOps diff plan without writing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.GitOps.WriteToWorkingTree = false
			result, err := server.NewDeploymentService().Plan(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printGitOpsPlanSummary(cmd.OutOrStdout(), result.Record.GitOpsPlan)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 2.3 local runtime")
	return cmd
}

func newGitOpsWriteCommand() *cobra.Command {
	var workingTree string
	var confirm bool
	cmd := &cobra.Command{
		Use:   "write --local <deployment.yaml> --working-tree <path> --confirm",
		Short: "Write GitOps changes to a local working tree",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirm {
				return fmt.Errorf("gitops write requires --confirm")
			}
			if workingTree == "" {
				return fmt.Errorf("--working-tree is required")
			}
			def, err := deploymentusecase.LoadDefinitionFile(args[0])
			if err != nil {
				return err
			}
			def.Spec.GitOps.WriteToWorkingTree = true
			def.Spec.GitOps.WorkingTree = workingTree
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			fmt.Fprintf(cmd.OutOrStdout(), "Diff: %s\n", result.Record.GitOpsDiff.Summary)
			return nil
		},
	}
	cmd.Flags().Bool("local", true, "use the in-process Phase 2.3 local runtime")
	cmd.Flags().StringVar(&workingTree, "working-tree", "", "local GitOps working tree root")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm local working tree writes")
	return cmd
}

func newArgoCDCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "argocd", Short: "Argo CD foundation utilities"}
	cmd.AddCommand(newArgoCDStatusCommand())
	cmd.AddCommand(newArgoCDSyncCommand())
	return cmd
}

func newArgoCDStatusCommand() *cobra.Command {
	var app string
	cmd := &cobra.Command{
		Use:   "status --app <name>",
		Short: "Read modeled Argo CD application status through the local noop provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == "" {
				return fmt.Errorf("--app is required")
			}
			def := gitOpsStatusDefinition(app)
			def.Spec.GitOps.StatusRead = true
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def})
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), result.Record.ArgoCD)
			return nil
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "Argo CD application name")
	cmd.Flags().String("server", "", "optional Argo CD URL for future adapters; ignored by the Phase 2.3 noop provider")
	return cmd
}

func newArgoCDSyncCommand() *cobra.Command {
	var app string
	var confirm bool
	var allowSync bool
	cmd := &cobra.Command{
		Use:   "sync --app <name> --confirm --allow-sync",
		Short: "Request Argo CD sync through the guarded local noop provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if app == "" {
				return fmt.Errorf("--app is required")
			}
			if !confirm || !allowSync {
				return fmt.Errorf("argocd sync requires --confirm and --allow-sync")
			}
			def := gitOpsStatusDefinition(app)
			def.Spec.GitOps.Sync = true
			result, err := server.NewDeploymentService().CreateAndRun(cmd.Context(), deploymentusecase.CreateRunInput{Definition: def, AllowSync: true, Confirm: true})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "DeploymentRun: %s\n", result.Record.Run.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", result.Record.Run.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&app, "app", "", "Argo CD application name")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm sync request")
	cmd.Flags().BoolVar(&allowSync, "allow-sync", false, "allow guarded sync request")
	return cmd
}

func gitOpsStatusDefinition(app string) deploymentusecase.Definition {
	return deploymentusecase.Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Deployment",
		Metadata:   deploymentusecase.Metadata{Name: app + "-argocd-status"},
		Spec: deploymentusecase.Spec{
			Application: "argocd-status",
			Environment: "local",
			Target: deploymentusecase.Target{
				Type:            "argocd",
				Name:            "local-noop",
				ApplicationName: app,
				RepoURL:         "placeholder://argocd-status",
				Path:            "apps/" + app,
				Revision:        "HEAD",
			},
			GitOps: deploymentusecase.GitOps{Mode: "plan"},
		},
	}
}

func newRunnerCommand() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "runner",
		Short: "Run nivora-runner or use runner utilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runner.Run(ctx, configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "configs/runner.yaml", "config file path")
	cmd.AddCommand(newRunnerListCommand())
	cmd.AddCommand(newRunnerHeartbeatCommand())
	return cmd
}

func newRunnerListCommand() *cobra.Command {
	var serverURL string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List runners from a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := doJSON(cmd.Context(), http.MethodGet, serverURL, "/api/v1/runners", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	return cmd
}

func newRunnerHeartbeatCommand() *cobra.Command {
	var serverURL string
	var name string
	cmd := &cobra.Command{
		Use:   "heartbeat --name <runner-id>",
		Short: "Record a runner heartbeat on a Nivora server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			payload, err := doJSON(cmd.Context(), http.MethodPost, serverURL, "/api/v1/runners/"+name+"/heartbeat", nil)
			if err != nil {
				return err
			}
			printJSON(cmd.OutOrStdout(), payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&serverURL, "server", "http://localhost:8080", "Nivora server URL")
	cmd.Flags().StringVar(&name, "name", "local-runner", "runner ID")
	return cmd
}

func doJSON(ctx context.Context, method string, serverURL string, path string, body []byte) (any, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("server URL is required")
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(serverURL, "/")+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload any
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &payload); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
	}
	if resp.StatusCode >= 400 {
		return payload, fmt.Errorf("server returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	return payload, nil
}

func printPipelineRunSummary(w io.Writer, payload any) {
	record, _ := payload.(map[string]any)
	run, _ := record["run"].(map[string]any)
	if run == nil {
		printJSON(w, payload)
		return
	}
	fmt.Fprintf(w, "PipelineRun: %v\n", run["id"])
	fmt.Fprintf(w, "Status: %v\n", run["status"])
	if failure, _ := run["failureReason"].(string); failure != "" {
		fmt.Fprintf(w, "Failure: %s\n", failure)
	}
	if logs, _ := record["logs"].([]any); logs != nil {
		fmt.Fprintf(w, "Logs: %d chunk(s)\n", len(logs))
	}
}

func printLogSummary(w io.Writer, payload any) {
	record, _ := payload.(map[string]any)
	logs, _ := record["logs"].([]any)
	if logs == nil {
		return
	}
	for _, item := range logs {
		log, _ := item.(map[string]any)
		if log == nil {
			continue
		}
		fmt.Fprintf(w, "[%v] %v", log["stream"], log["content"])
	}
}

func printDeploymentPlanSummary(w io.Writer, plan deploymentusecase.DeploymentPlan) {
	fmt.Fprintf(w, "DeploymentRun: %s\n", plan.DeploymentRunID)
	fmt.Fprintf(w, "Target: %s\n", plan.TargetType)
	fmt.Fprintf(w, "Namespace: %s\n", plan.Namespace)
	fmt.Fprintf(w, "Manifests: %d\n", plan.ManifestCount)
	fmt.Fprintf(w, "DryRun: %t\n", plan.DryRun)
	fmt.Fprintf(w, "Apply: %t\n", plan.Apply)
	if plan.DiffSummary != "" {
		fmt.Fprintf(w, "Diff: %s\n", plan.DiffSummary)
	}
	if len(plan.Warnings) > 0 {
		fmt.Fprintf(w, "Warnings:\n")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(w, "- %s\n", warning)
		}
	}
}

func printGitOpsPlanSummary(w io.Writer, plan deploymentusecase.GitOpsChangePlan) {
	fmt.Fprintf(w, "Application: %s\n", plan.ApplicationName)
	fmt.Fprintf(w, "Repo: %s\n", plan.RepoURL)
	fmt.Fprintf(w, "Path: %s\n", plan.Path)
	fmt.Fprintf(w, "Revision: %s\n", plan.Revision)
	fmt.Fprintf(w, "Files: %d\n", len(plan.Files))
	fmt.Fprintf(w, "Artifacts: %d\n", len(plan.ArtifactChanges))
	fmt.Fprintf(w, "SyncRequested: %t\n", plan.SyncRequested)
	if len(plan.Warnings) > 0 {
		fmt.Fprintf(w, "Warnings:\n")
		for _, warning := range plan.Warnings {
			fmt.Fprintf(w, "- %s\n", warning)
		}
	}
}

func printDeploymentRunSummary(w io.Writer, payload any) {
	record, _ := payload.(map[string]any)
	run, _ := record["run"].(map[string]any)
	if run == nil {
		printJSON(w, payload)
		return
	}
	fmt.Fprintf(w, "DeploymentRun: %v\n", run["id"])
	fmt.Fprintf(w, "Status: %v\n", run["status"])
	if reason, _ := run["reason"].(string); reason != "" {
		fmt.Fprintf(w, "Reason: %s\n", reason)
	}
	if plan, _ := record["plan"].(map[string]any); plan != nil {
		fmt.Fprintf(w, "Manifests: %v\n", plan["manifestCount"])
	}
	if logs, _ := record["logs"].([]any); logs != nil {
		fmt.Fprintf(w, "Logs: %d chunk(s)\n", len(logs))
	}
}

func printJSON(w io.Writer, payload any) {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "%v\n", payload)
		return
	}
	fmt.Fprintf(w, "%s\n", encoded)
}
