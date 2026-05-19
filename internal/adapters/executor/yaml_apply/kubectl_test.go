package yamlapply

import (
	"context"
	"errors"
	"strings"
	"testing"

	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

func TestKubectlManifestClientCommandConstruction(t *testing.T) {
	runner := &recordingRunner{}
	client := NewKubectlManifestClient("kubectl-test", runner)
	request := kubectlTestRequest()

	if _, err := client.ServerDryRun(context.Background(), request); err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if runner.calls[0].name != "kubectl-test" {
		t.Fatalf("binary = %s", runner.calls[0].name)
	}
	assertArgs(t, runner.calls[0].args, "--context", "kind-dev", "--namespace", "default", "apply", "--server-side", "--dry-run=server", "-f", "-")
	if !strings.Contains(runner.calls[0].stdin, "kind: Deployment") {
		t.Fatalf("stdin missing manifest: %q", runner.calls[0].stdin)
	}

	if _, err := client.Apply(context.Background(), request); err != nil {
		t.Fatalf("apply: %v", err)
	}
	assertArgs(t, runner.calls[1].args, "--context", "kind-dev", "--namespace", "default", "apply", "-f", "-")

	if _, err := client.Rollback(context.Background(), request); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	assertArgs(t, runner.calls[2].args, "--context", "kind-dev", "--namespace", "default", "apply", "-f", "-")
}

func TestKubectlManifestClientRolloutCommands(t *testing.T) {
	runner := &recordingRunner{}
	client := NewKubectlManifestClient("kubectl", runner)
	request := kubectlTestRequest()
	request.Plan.Resources = append(request.Plan.Resources, deploymentusecase.ManifestResourceSummary{Kind: "Service", Name: "demo", Namespace: "default"})
	request.TimeoutSeconds = 45

	result, err := client.WatchRollout(context.Background(), request)
	if err != nil {
		t.Fatalf("rollout: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one supported rollout command, got %d", len(runner.calls))
	}
	assertArgs(t, runner.calls[0].args, "--context", "kind-dev", "--namespace", "default", "rollout", "status", "deployment/demo", "--timeout", "45s")
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "Service/demo") {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
}

func TestKubectlManifestClientRequiresExplicitContextAndNamespace(t *testing.T) {
	client := NewKubectlManifestClient("kubectl", &recordingRunner{})
	request := kubectlTestRequest()
	request.Plan.TargetContext = ""
	if _, err := client.ServerDryRun(context.Background(), request); err == nil {
		t.Fatal("expected missing context error")
	}
	request = kubectlTestRequest()
	request.Plan.Namespace = ""
	if _, err := client.Apply(context.Background(), request); err == nil {
		t.Fatal("expected missing namespace error")
	}
}

func TestKubectlManifestClientReturnsCommandErrors(t *testing.T) {
	client := NewKubectlManifestClient("kubectl", &recordingRunner{err: errors.New("boom"), stderr: "cluster denied"})
	_, err := client.Apply(context.Background(), kubectlTestRequest())
	if err == nil || !strings.Contains(err.Error(), "cluster denied") {
		t.Fatalf("err = %v", err)
	}
}

func kubectlTestRequest() deploymentusecase.ManifestRequest {
	return deploymentusecase.ManifestRequest{
		Plan: deploymentusecase.DeploymentPlan{
			DeploymentRunID: "drun-test",
			TargetContext:   "kind-dev",
			Namespace:       "default",
			Resources: []deploymentusecase.ManifestResourceSummary{{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "demo",
				Namespace:  "default",
			}},
		},
		Documents: []deploymentusecase.ManifestDocument{{
			Content: "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: demo\n",
		}},
	}
}

type recordingRunner struct {
	calls  []recordedCall
	stdout string
	stderr string
	err    error
}

type recordedCall struct {
	name  string
	args  []string
	stdin string
}

func (r *recordingRunner) Run(ctx context.Context, name string, args []string, stdin string) (string, string, error) {
	r.calls = append(r.calls, recordedCall{name: name, args: append([]string(nil), args...), stdin: stdin})
	return r.stdout, r.stderr, r.err
}

func assertArgs(t *testing.T, got []string, want ...string) {
	t.Helper()
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}
