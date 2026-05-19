package yamlapply

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, stdin string) (stdout string, stderr string, err error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, name string, args []string, stdin string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

type KubectlManifestClient struct {
	Binary string
	Runner CommandRunner
}

func NewKubectlManifestClient(binary string, runner CommandRunner) KubectlManifestClient {
	if binary == "" {
		binary = "kubectl"
	}
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	return KubectlManifestClient{Binary: binary, Runner: runner}
}

func (c KubectlManifestClient) ServerDryRun(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesDryRunResult, error) {
	if err := validateKubectlRequest(request); err != nil {
		return deploymentusecase.KubernetesDryRunResult{}, err
	}
	stdout, stderr, err := c.run(ctx, request, []string{"apply", "--server-side", "--dry-run=server", "-f", "-"}, manifestInput(request))
	if err != nil {
		return deploymentusecase.KubernetesDryRunResult{}, fmt.Errorf("kubectl server-side dry-run failed: %w: %s", err, strings.TrimSpace(stderr))
	}
	return deploymentusecase.KubernetesDryRunResult{Mode: "kubectl", Message: "server-side dry-run completed", Resources: request.Plan.Resources, Stdout: stdout, Stderr: stderr}, nil
}

func (c KubectlManifestClient) Apply(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesApplyResult, error) {
	if err := validateKubectlRequest(request); err != nil {
		return deploymentusecase.KubernetesApplyResult{}, err
	}
	stdout, stderr, err := c.run(ctx, request, []string{"apply", "-f", "-"}, manifestInput(request))
	if err != nil {
		return deploymentusecase.KubernetesApplyResult{}, fmt.Errorf("kubectl apply failed: %w: %s", err, strings.TrimSpace(stderr))
	}
	return deploymentusecase.KubernetesApplyResult{Mode: "kubectl", Message: "apply completed", Resources: request.Plan.Resources, Stdout: stdout, Stderr: stderr}, nil
}

func (c KubectlManifestClient) WatchRollout(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.RolloutResult, error) {
	if err := validateKubectlRequest(request); err != nil {
		return deploymentusecase.RolloutResult{}, err
	}
	timeout := timeoutArg(request.TimeoutSeconds)
	checked := make([]deploymentusecase.ManifestResourceSummary, 0, len(request.Plan.Resources))
	warnings := []string{}
	var stdout strings.Builder
	var stderr strings.Builder
	for _, resource := range request.Plan.Resources {
		args, ok := rolloutCommand(resource, timeout)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("rollout watch does not support %s/%s", resource.Kind, resource.Name))
			continue
		}
		out, errOut, err := c.run(ctx, request, args, "")
		stdout.WriteString(out)
		stderr.WriteString(errOut)
		if err != nil {
			return deploymentusecase.RolloutResult{}, fmt.Errorf("kubectl rollout watch failed for %s/%s: %w: %s", resource.Kind, resource.Name, err, strings.TrimSpace(errOut))
		}
		checked = append(checked, resource)
	}
	return deploymentusecase.RolloutResult{
		Mode:      "kubectl",
		Message:   fmt.Sprintf("rollout verification completed for %d resource(s)", len(checked)),
		Resources: checked,
		Warnings:  warnings,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
	}, nil
}

func (c KubectlManifestClient) Rollback(ctx context.Context, request deploymentusecase.ManifestRequest) (deploymentusecase.KubernetesRollbackResult, error) {
	if err := validateKubectlRequest(request); err != nil {
		return deploymentusecase.KubernetesRollbackResult{}, err
	}
	stdout, stderr, err := c.run(ctx, request, []string{"apply", "-f", "-"}, manifestInput(request))
	if err != nil {
		return deploymentusecase.KubernetesRollbackResult{}, fmt.Errorf("kubectl rollback manifest restore failed: %w: %s", err, strings.TrimSpace(stderr))
	}
	return deploymentusecase.KubernetesRollbackResult{Mode: "kubectl", Message: "rollback manifest restore applied", Resources: request.Plan.Resources, Stdout: stdout, Stderr: stderr}, nil
}

func (c KubectlManifestClient) run(ctx context.Context, request deploymentusecase.ManifestRequest, operation []string, stdin string) (string, string, error) {
	binary := c.Binary
	if binary == "" {
		binary = "kubectl"
	}
	runner := c.Runner
	if runner == nil {
		runner = ExecCommandRunner{}
	}
	args := kubectlBaseArgs(request)
	args = append(args, operation...)
	return runner.Run(ctx, binary, args, stdin)
}

func validateKubectlRequest(request deploymentusecase.ManifestRequest) error {
	if err := validateRequest(request); err != nil {
		return err
	}
	if request.Plan.TargetContext == "" {
		return fmt.Errorf("kubernetes target.context is required for the kubectl adapter")
	}
	if request.Plan.Namespace == "" {
		return fmt.Errorf("kubernetes target.namespace is required for the kubectl adapter")
	}
	return nil
}

func kubectlBaseArgs(request deploymentusecase.ManifestRequest) []string {
	args := []string{"--context", request.Plan.TargetContext, "--namespace", request.Plan.Namespace}
	return args
}

func manifestInput(request deploymentusecase.ManifestRequest) string {
	var input strings.Builder
	for _, doc := range request.Documents {
		input.WriteString("---\n")
		input.WriteString(doc.Content)
		if !strings.HasSuffix(doc.Content, "\n") {
			input.WriteString("\n")
		}
	}
	return input.String()
}

func rolloutCommand(resource deploymentusecase.ManifestResourceSummary, timeout string) ([]string, bool) {
	name := strings.ToLower(resource.Kind) + "/" + resource.Name
	switch resource.Kind {
	case "Deployment", "StatefulSet", "DaemonSet":
		return []string{"rollout", "status", name, "--timeout", timeout}, true
	case "Job":
		return []string{"wait", "--for=condition=complete", name, "--timeout", timeout}, true
	default:
		return nil, false
	}
}

func timeoutArg(seconds int) string {
	if seconds <= 0 {
		return (2 * time.Minute).String()
	}
	return strconv.Itoa(seconds) + "s"
}
