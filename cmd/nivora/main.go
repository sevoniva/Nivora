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
	root.AddCommand(newConfigCommand())
	root.AddCommand(newPipelineCommand())
	root.AddCommand(newRunnerCommand())
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

func printJSON(w io.Writer, payload any) {
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(w, "%v\n", payload)
		return
	}
	fmt.Fprintf(w, "%s\n", encoded)
}
