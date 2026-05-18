package deployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/event"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	portgitops "github.com/sevoniva/nivora/internal/ports/gitops"
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
	if record.Snapshot.ContentHash == "" || record.RollbackPlan.Executable {
		t.Fatalf("snapshot/rollback = %#v %#v", record.Snapshot, record.RollbackPlan)
	}
	if record.Health.ResourcesChecked == 0 || record.Diff.Summary == "" {
		t.Fatalf("health/diff = %#v %#v", record.Health, record.Diff)
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
	assertHasDeploymentEvent(t, events, EventDeploymentInventoryCreated)
	assertHasDeploymentEvent(t, events, EventDeploymentSnapshotCreated)
	assertHasDeploymentEvent(t, events, EventDeploymentHealthCompleted)
	assertHasDeploymentEvent(t, events, EventDeploymentRollbackPlanCreated)
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

func TestServiceDeploymentPlanWarnsForUnboundManifestImage(t *testing.T) {
	service, def := newTestService(t, true, nil)
	def.Spec.Artifacts = nil
	result, err := service.Plan(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(result.Record.Plan.ManifestImages) == 0 {
		t.Fatal("expected manifest image inventory")
	}
	if len(result.Record.Plan.Warnings) == 0 {
		t.Fatal("expected manifest image warning")
	}
}

func TestServiceDeploymentPlanBindsManifestImage(t *testing.T) {
	service, def := newTestService(t, true, nil)
	def.Spec.Artifacts[0].Target.ImageName = "demo-app"
	result, err := service.Plan(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(result.Record.Plan.ArtifactDetails) != 1 {
		t.Fatalf("artifact details = %d", len(result.Record.Plan.ArtifactDetails))
	}
	if len(result.Record.Plan.ManifestImages) != 1 {
		t.Fatalf("manifest images = %d", len(result.Record.Plan.ManifestImages))
	}
	for _, warning := range result.Record.Plan.Warnings {
		if strings.Contains(warning, "differs from bound artifact") || strings.Contains(warning, "not bound to a release artifact") {
			t.Fatalf("unexpected binding warning: %s", warning)
		}
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
	service, def := newTestServiceWithClient(t, true, testManifestClient{dryRunErr: errors.New("dry-run failed")})

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

func TestServiceRejectsApplyWithoutExplicitConfirmation(t *testing.T) {
	service, def := newTestService(t, true, nil)
	def.Spec.Options = Options{Apply: true, DryRun: false}

	_, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err == nil {
		t.Fatal("expected apply confirmation error")
	}
}

func TestServiceApplySuccessWithRollout(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{})
	def.Spec.Options = Options{Apply: true, DryRun: false, Wait: true, TimeoutSeconds: 30}

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true})
	if err != nil {
		t.Fatalf("apply run: %v", err)
	}
	record := result.Record
	if record.Run.Status != domaindeployment.DeploymentRunSucceeded {
		t.Fatalf("status = %s", record.Run.Status)
	}
	if !record.Plan.Apply || !record.Plan.Wait {
		t.Fatalf("plan = %#v", record.Plan)
	}
	if record.Apply.Message == "" {
		t.Fatal("expected apply result")
	}
	if record.Rollout.Message == "" {
		t.Fatal("expected rollout result")
	}
	if record.Rollback == nil || len(record.Rollback.ResourceRefs) == 0 {
		t.Fatalf("rollback baseline = %#v", record.Rollback)
	}
	assertHasDeploymentEvent(t, record.Events, EventDeploymentApplyStarted)
	assertHasDeploymentEvent(t, record.Events, EventDeploymentApplyCompleted)
	assertHasDeploymentEvent(t, record.Events, EventDeploymentVerifyCompleted)
}

func TestServiceApplyFailure(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{applyErr: errors.New("apply failed")})
	def.Spec.Options = Options{Apply: true, DryRun: false}

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true})
	if err != nil {
		t.Fatalf("apply failure should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventDeploymentApplyFailed)
}

func TestServiceRolloutFailure(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{rolloutErr: errors.New("rollout timeout")})
	def.Spec.Options = Options{Apply: true, DryRun: false, Wait: true}

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true})
	if err != nil {
		t.Fatalf("rollout failure should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventDeploymentVerifyFailed)
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

func TestServicePlansGitOpsDeployment(t *testing.T) {
	service, def := newGitOpsTestService(t)
	result, err := service.Plan(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("plan gitops: %v", err)
	}
	plan := result.Record.GitOpsPlan
	if plan.ApplicationName != "demo-springboot" || len(plan.ArtifactChanges) != 1 {
		t.Fatalf("gitops plan = %#v", plan)
	}
	if !plan.DryRun || plan.SyncRequested {
		t.Fatalf("plan flags = %#v", plan)
	}
}

func TestServiceGitOpsRunSkipsSyncByDefault(t *testing.T) {
	service, def := newGitOpsTestService(t)
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("run gitops: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunSucceeded {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventGitOpsPlanCreated)
	assertHasDeploymentEvent(t, result.Record.Events, EventArgoCDSyncSkipped)
}

func TestServiceGitOpsWorkingTreeUpdate(t *testing.T) {
	service, def := newGitOpsTestService(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "apps/demo-springboot/dev/deployment.yaml")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(file, []byte("containers:\n  - name: app\n    image: old.example/demo:old\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	def.Spec.GitOps.WriteToWorkingTree = true
	def.Spec.GitOps.WorkingTree = dir
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("run gitops write: %v", err)
	}
	if len(result.Record.GitOpsDiff.Files) != 1 || !result.Record.GitOpsDiff.Files[0].Changed {
		t.Fatalf("diff = %#v", result.Record.GitOpsDiff)
	}
	body, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(body), "registry.example.com/demo/demo-springboot@sha256:example") {
		t.Fatalf("updated body = %s", string(body))
	}
}

func TestServiceGitOpsSyncRequiresConfirmation(t *testing.T) {
	service, def := newGitOpsTestService(t)
	def.Spec.GitOps.Sync = true
	_, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err == nil {
		t.Fatal("expected sync confirmation error")
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
	return newTestServiceWithClient(t, policyAllowed, testManifestClient{dryRunErr: clientErr})
}

func newTestServiceWithClient(t *testing.T, policyAllowed bool, client ManifestClient) (*Service, Definition) {
	t.Helper()
	dir := t.TempDir()
	manifest := filepath.Join(dir, "resources.yaml")
	if err := os.WriteFile(manifest, []byte(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo
spec:
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
    spec:
      containers:
        - name: demo-app
          image: example.local/demo:dev
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
	service := NewService(NewMemoryStore(), StaticManifestRenderer{}, client, testPolicy{allowed: policyAllowed}, testEventBus{})
	return service, def
}

func newGitOpsTestService(t *testing.T) (*Service, Definition) {
	t.Helper()
	service := NewService(NewMemoryStore(), StaticManifestRenderer{}, testManifestClient{}, testPolicy{allowed: true}, testEventBus{}).
		WithGitOps(fakeWorkingTree{}, fakeArgoCDProvider{})
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Deployment",
		Metadata:   Metadata{Name: "demo-gitops"},
		Spec: Spec{
			Application: "demo-springboot",
			Environment: "dev",
			Target: Target{
				Type:            "argocd",
				Name:            "demo-argocd",
				ApplicationName: "demo-springboot",
				RepoURL:         "https://example.com/gitops/demo.git",
				Path:            "apps/demo-springboot/dev",
				Revision:        "main",
			},
			Artifacts: []Artifact{{
				Name:      "demo-springboot",
				Type:      "image",
				Reference: "registry.example.com/demo/demo-springboot@sha256:example",
				Target:    ArtifactTarget{ImageName: "app"},
			}},
			GitOps: GitOps{Mode: "plan", Files: []string{"apps/demo-springboot/dev/deployment.yaml"}},
		},
	}
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

type testManifestClient struct {
	dryRunErr  error
	applyErr   error
	rolloutErr error
}

func (c testManifestClient) ServerDryRun(ctx context.Context, request ManifestRequest) (KubernetesDryRunResult, error) {
	if c.dryRunErr != nil {
		return KubernetesDryRunResult{}, c.dryRunErr
	}
	return KubernetesDryRunResult{Mode: "test", Message: "dry-run ok", Resources: request.Plan.Resources}, nil
}

func (c testManifestClient) Apply(ctx context.Context, request ManifestRequest) (KubernetesApplyResult, error) {
	if c.applyErr != nil {
		return KubernetesApplyResult{}, c.applyErr
	}
	return KubernetesApplyResult{Mode: "test", Message: "apply ok", Resources: request.Plan.Resources}, nil
}

func (c testManifestClient) WatchRollout(ctx context.Context, request ManifestRequest) (RolloutResult, error) {
	if c.rolloutErr != nil {
		return RolloutResult{}, c.rolloutErr
	}
	return RolloutResult{Mode: "test", Message: "rollout ok", Resources: request.Plan.Resources}, nil
}

type testEventBus struct{}

func (testEventBus) Publish(ctx context.Context, evt event.Event) error { return nil }
func (testEventBus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event)
	close(ch)
	return ch, nil
}

type fakeWorkingTree struct{}

func (fakeWorkingTree) ReadFile(ctx context.Context, root string, path string) (string, error) {
	body, err := os.ReadFile(filepath.Join(root, path))
	return string(body), err
}

func (fakeWorkingTree) WriteFile(ctx context.Context, root string, path string, content string) error {
	return os.WriteFile(filepath.Join(root, path), []byte(content), 0o600)
}

func (fakeWorkingTree) Diff(ctx context.Context, root string, path string, before string, after string) (string, error) {
	return before + "\n---\n" + after, nil
}

var _ portgitops.WorkingTree = fakeWorkingTree{}

type fakeArgoCDProvider struct{}

func (fakeArgoCDProvider) ValidateCredential(ctx context.Context, credential portargocd.CredentialRef) error {
	return nil
}

func (fakeArgoCDProvider) GetApplicationStatus(ctx context.Context, applicationName string) (portargocd.ApplicationStatus, error) {
	return portargocd.ApplicationStatus{ApplicationName: applicationName, SyncStatus: "Synced", HealthStatus: "Healthy", Message: "test status"}, nil
}

func (fakeArgoCDProvider) SyncApplication(ctx context.Context, request portargocd.SyncRequest) (portargocd.SyncResult, error) {
	return portargocd.SyncResult{ApplicationName: request.ApplicationName, Requested: request.AllowSync && request.Confirmed, Message: "test sync"}, nil
}
