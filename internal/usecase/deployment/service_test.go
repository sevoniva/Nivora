package deployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/ports/policy"
)

func TestServiceCreateAndRunDryRunDeployment(t *testing.T) {
	service, def := newTestService(t, true, nil)

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	record := result.Record
	if record.Run.Status != domaindeployment.DeploymentRunSucceeded {
		t.Fatalf("status = %s", record.Run.Status)
	}
	if record.Plan.ManifestCount != 2 {
		t.Fatalf("manifest count = %d", record.Plan.ManifestCount)
	}
	if len(record.Logs) == 0 {
		t.Fatal("expected logs")
	}
	events, err := service.Events(context.Background(), record.Run.ID)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	assertHasDeploymentEvent(t, events, EventDeploymentCreated)
	assertHasDeploymentEvent(t, events, EventDeploymentPrecheckCompleted)
	assertHasDeploymentEvent(t, events, EventDeploymentDryRunCompleted)
	assertHasDeploymentEvent(t, events, EventDeploymentSucceeded)

	timeline, err := service.Timeline(context.Background(), record.Run.ID)
	if err != nil {
		t.Fatalf("timeline: %v", err)
	}
	if len(timeline) != len(events) {
		t.Fatalf("timeline len = %d events len = %d", len(timeline), len(events))
	}
	audits, err := service.store.Audits(context.Background(), record.Run.ID)
	if err != nil {
		t.Fatalf("audits: %v", err)
	}
	if len(audits) < 4 {
		t.Fatalf("audit count = %d", len(audits))
	}
}

func TestServiceFailsInvalidManifest(t *testing.T) {
	service, def := newTestService(t, true, nil)
	invalid := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(invalid, []byte("kind: Service\nmetadata:\n  name: bad\n"), 0o600); err != nil {
		t.Fatalf("write invalid manifest: %v", err)
	}
	def.Spec.Manifests = []string{invalid}

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("create and run should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if result.Record.Run.Reason == "" {
		t.Fatal("expected failure reason")
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventDeploymentFailed)
}

func TestServiceFailsWhenPolicyDenies(t *testing.T) {
	service, def := newTestService(t, false, nil)

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("create and run should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if result.Record.Policy.Allowed {
		t.Fatal("expected denied policy result")
	}
}

func TestServiceFailsWhenDryRunClientFails(t *testing.T) {
	service, def := newTestService(t, true, errors.New("dry-run failed"))

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("create and run should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if result.Record.Run.Reason != "dry-run failed" {
		t.Fatalf("reason = %q", result.Record.Run.Reason)
	}
}

func TestServiceCancelCreatedRun(t *testing.T) {
	service, def := newTestService(t, true, nil)
	record := service.newRecord(def)
	if err := service.store.Save(context.Background(), record); err != nil {
		t.Fatalf("save record: %v", err)
	}

	canceled, err := service.Cancel(context.Background(), record.Run.ID, "tester")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if canceled.Run.Status != domaindeployment.DeploymentRunCanceled {
		t.Fatalf("status = %s", canceled.Run.Status)
	}
	_, err = service.Cancel(context.Background(), record.Run.ID, "tester")
	if !errors.Is(err, ErrRunTerminal) {
		t.Fatalf("expected terminal error, got %v", err)
	}
}

func TestMemoryStoreOrdersLogs(t *testing.T) {
	store := NewMemoryStore()
	record := RunRecord{Run: domaindeployment.DeploymentRun{ID: "run-logs"}}
	if err := store.Save(context.Background(), record); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := store.AppendLog(context.Background(), "run-logs", event.LogChunk{ID: "later", Content: "later"}); err != nil {
		t.Fatalf("append log: %v", err)
	}
	if err := store.AppendLog(context.Background(), "run-logs", event.LogChunk{ID: "earlier", Content: "earlier"}); err != nil {
		t.Fatalf("append log: %v", err)
	}
	logs, err := store.Logs(context.Background(), "run-logs")
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if logs[0].Sequence != 1 || logs[1].Sequence != 2 {
		t.Fatalf("sequences = %d,%d", logs[0].Sequence, logs[1].Sequence)
	}
}

func newTestService(t *testing.T, policyAllowed bool, clientErr error) (*Service, Definition) {
	t.Helper()
	dir := t.TempDir()
	manifest := filepath.Join(dir, "resources.yaml")
	if err := os.WriteFile(manifest, []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
---
apiVersion: v1
kind: Service
metadata:
  name: demo
`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Deployment",
		Metadata:   Metadata{Name: "demo-yaml"},
		Spec: Spec{
			Application: "demo-app",
			Environment: "dev",
			Target:      Target{Type: "kubernetes-yaml", Name: "dev-kind", Namespace: "default"},
			Artifacts: []Artifact{{
				Name:      "demo-app",
				Type:      "image",
				Reference: "example.local/demo:dev",
			}},
			Manifests: []string{manifest},
			Options:   Options{DryRun: true, Apply: false},
		},
	}
	service := NewService(NewMemoryStore(), StaticManifestRenderer{}, failingManifestClient{err: clientErr}, testPolicy{allowed: policyAllowed}, testEventBus{})
	return service, def
}

func assertHasDeploymentEvent(t *testing.T, events []event.Event, eventType string) {
	t.Helper()
	for _, evt := range events {
		if evt.Type == eventType {
			return
		}
	}
	t.Fatalf("missing event %s in %#v", eventType, events)
}

type testPolicy struct {
	allowed bool
}

func (p testPolicy) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	if p.allowed {
		return policy.Result{Allowed: true}, nil
	}
	return policy.Result{Allowed: false, Reasons: []string{"denied by test policy"}}, nil
}

type failingManifestClient struct {
	err error
}

func (c failingManifestClient) DryRun(ctx context.Context, plan DeploymentPlan, documents []ManifestDocument) error {
	return c.err
}

type testEventBus struct{}

func (testEventBus) Publish(ctx context.Context, evt event.Event) error { return nil }
func (testEventBus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event)
	close(ch)
	return ch, nil
}
