package deployment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	domainapp "github.com/sevoniva/nivora/internal/domain/application"
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/event"
	portargocd "github.com/sevoniva/nivora/internal/ports/argocd"
	portexecutor "github.com/sevoniva/nivora/internal/ports/executor"
	portgitops "github.com/sevoniva/nivora/internal/ports/gitops"
	"github.com/sevoniva/nivora/internal/ports/policy"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
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

func TestServicePersistsProjectScope(t *testing.T) {
	service, def := newTestService(t, true, nil)
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{
		Definition: def,
		ProjectID:  " project-a ",
	})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	if result.Record.Environment.ProjectID != "project-a" || result.Record.Target.ProjectID != "project-a" {
		t.Fatalf("project scope not persisted: environment=%q target=%q", result.Record.Environment.ProjectID, result.Record.Target.ProjectID)
	}
	own, err := service.ListFiltered(context.Background(), "project", "project-a")
	if err != nil {
		t.Fatalf("list own project: %v", err)
	}
	if len(own) != 1 || own[0].Run.ID != result.Record.Run.ID {
		t.Fatalf("own project list = %#v", own)
	}
	other, err := service.ListFiltered(context.Background(), "project", "project-b")
	if err != nil {
		t.Fatalf("list other project: %v", err)
	}
	if len(other) != 0 {
		t.Fatalf("other project list should be empty, got %#v", other)
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

func TestServiceDeploymentPlanWarnsForKubernetesSafetyViolation(t *testing.T) {
	service, def := newTestService(t, true, nil)
	writeDeploymentManifest(t, def, `
apiVersion: v1
kind: Pod
metadata:
  name: unsafe
spec:
  containers:
    - name: app
      image: example.local/demo:dev
      securityContext:
        privileged: true
`)

	result, err := service.Plan(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	assertPlanWarningContains(t, result.Record.Plan.Warnings, "kubernetes safety policy denied")
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

func TestServiceRejectsKubernetesSafetyViolationBeforeDryRun(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{})
	writeDeploymentManifest(t, def, `
apiVersion: v1
kind: Pod
metadata:
  name: unsafe
spec:
  containers:
    - name: app
      image: example.local/demo:dev
      securityContext:
        privileged: true
`)

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("safety denial should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if !strings.Contains(result.Record.Run.Reason, "kubernetes safety policy denied") || !strings.Contains(result.Record.Run.Reason, "privileged") {
		t.Fatalf("reason = %q", result.Record.Run.Reason)
	}
	if result.Record.DryRun.Message != "" {
		t.Fatalf("dry-run should not execute after safety denial: %#v", result.Record.DryRun)
	}
	assertPlanWarningContains(t, result.Record.Plan.Warnings, "kubernetes safety policy denied")
}

func TestServiceRejectsTooManyKubernetesResourcesBeforeDryRun(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{})
	writeDeploymentManifest(t, def, manyConfigMapsManifest(DefaultK8sSafetyPolicy().MaxResourceCount+1))

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("resource-count denial should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if !strings.Contains(result.Record.Run.Reason, "resource count") || !strings.Contains(result.Record.Run.Reason, "exceeds max") {
		t.Fatalf("reason = %q", result.Record.Run.Reason)
	}
	if result.Record.DryRun.Message != "" {
		t.Fatalf("dry-run should not execute after resource-count denial: %#v", result.Record.DryRun)
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

func TestServiceApprovalRequiredBlocksDeployment(t *testing.T) {
	service, def := newTestService(t, true, nil)
	service.WithGovernance(testGovernance{})
	def.Spec.Options.ApprovalRequired = true

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunWaitingApproval {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if result.Record.Approval.Status != domainapproval.StatusPending || result.Record.Approval.SubjectID != result.Record.Run.ID {
		t.Fatalf("approval = %#v", result.Record.Approval)
	}
}

func TestSavedPolicyAttachmentRequiresDeploymentApproval(t *testing.T) {
	service, def := newTestService(t, true, nil)
	catalog := policyusecase.NewService(policyusecase.NewMemoryStore())
	policyDef, err := catalog.Create(context.Background(), policyusecase.CreateInput{
		ID:            "policy-deploy-approval",
		Name:          "Deployment digest approval",
		Mode:          "require_approval",
		RequireDigest: true,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if _, err := catalog.Attach(context.Background(), policyDef.ID, policyusecase.AttachInput{ScopeType: "environment", ScopeID: "dev"}); err != nil {
		t.Fatalf("attach policy: %v", err)
	}
	service.WithSecurity(securityusecase.NewService(securityusecase.NewMemoryStore(), testSecurityScanner{}, nil, nil)).
		WithPolicyCatalog(catalog).
		WithGovernance(testGovernance{})

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ProjectID: "project-a", ActorID: "tester"})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunWaitingApproval {
		t.Fatalf("status = %s, want WaitingApproval", result.Record.Run.Status)
	}
	if result.Record.Security.Policy.PolicyID != policyDef.ID {
		t.Fatalf("policy id = %q, want %q", result.Record.Security.Policy.PolicyID, policyDef.ID)
	}
	if result.Record.Approval.Status != domainapproval.StatusPending || result.Record.Approval.Reason != "artifact digest is required" {
		t.Fatalf("approval = %#v", result.Record.Approval)
	}
}

func TestSavedPolicyAttachmentDeniesDeployment(t *testing.T) {
	service, def := newTestService(t, true, nil)
	catalog := policyusecase.NewService(policyusecase.NewMemoryStore())
	policyDef, err := catalog.Create(context.Background(), policyusecase.CreateInput{
		ID:            "policy-deploy-deny",
		Name:          "Deployment digest deny",
		Mode:          "deny",
		RequireDigest: true,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if _, err := catalog.Attach(context.Background(), policyDef.ID, policyusecase.AttachInput{ScopeType: "environment", ScopeID: "dev"}); err != nil {
		t.Fatalf("attach policy: %v", err)
	}
	service.WithSecurity(securityusecase.NewService(securityusecase.NewMemoryStore(), testSecurityScanner{}, nil, nil)).
		WithPolicyCatalog(catalog)

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ProjectID: "project-a", ActorID: "tester"})
	if err != nil {
		t.Fatalf("create and run should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s, want Failed", result.Record.Run.Status)
	}
	if result.Record.Security.Policy.PolicyID != policyDef.ID {
		t.Fatalf("policy id = %q, want %q", result.Record.Security.Policy.PolicyID, policyDef.ID)
	}
}

func TestServiceApprovalApprovedResumesDeployment(t *testing.T) {
	service, def := newTestService(t, true, nil)
	service.WithGovernance(testGovernance{})
	def.Spec.Options.ApprovalRequired = true

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	resumed, err := service.ApplyApprovalDecision(context.Background(), result.Record.Run.ID, domainapproval.ApprovalRequest{
		ID:          result.Record.Approval.ID,
		SubjectType: domainapproval.SubjectDeployment,
		SubjectID:   result.Record.Run.ID,
		Status:      domainapproval.StatusApproved,
	}, "reviewer")
	if err != nil {
		t.Fatalf("resume after approval: %v", err)
	}
	if resumed.Run.Status != domaindeployment.DeploymentRunSucceeded {
		t.Fatalf("status = %s", resumed.Run.Status)
	}
	assertHasDeploymentEvent(t, resumed.Events, EventDeploymentSucceeded)
}

func TestServiceApprovalRejectedStopsDeployment(t *testing.T) {
	service, def := newTestService(t, true, nil)
	service.WithGovernance(testGovernance{})
	def.Spec.Options.ApprovalRequired = true

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("create and run: %v", err)
	}
	stopped, err := service.ApplyApprovalDecision(context.Background(), result.Record.Run.ID, domainapproval.ApprovalRequest{
		ID:          result.Record.Approval.ID,
		SubjectType: domainapproval.SubjectDeployment,
		SubjectID:   result.Record.Run.ID,
		Status:      domainapproval.StatusRejected,
	}, "reviewer")
	if err != nil {
		t.Fatalf("reject after approval: %v", err)
	}
	if stopped.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", stopped.Run.Status)
	}
}

func TestServiceChangeWindowDeniedBlocksDeployment(t *testing.T) {
	service, def := newTestService(t, true, nil)
	service.WithGovernance(testGovernance{changeWindowAllowed: false})
	def.Spec.Options.ChangeWindowRequired = true

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("create and run should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if result.Record.ChangeWindow.Allowed {
		t.Fatalf("change window = %#v", result.Record.ChangeWindow)
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
	_, err = service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true})
	if err == nil {
		t.Fatal("expected apply confirm error")
	}
}

func TestServiceApplySuccessWithRollout(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{})
	def.Spec.Options = Options{Apply: true, DryRun: false, Wait: true, TimeoutSeconds: 30}

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true, Confirm: true})
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

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true, Confirm: true})
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

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true, Confirm: true})
	if err != nil {
		t.Fatalf("rollout failure should persist failed run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventDeploymentVerifyFailed)
}

func TestServiceRollbackRequiresConfirmation(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{})
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}

	_, err = service.Rollback(context.Background(), RollbackInput{DeploymentRunID: result.Record.Run.ID})
	if err == nil {
		t.Fatal("expected rollback confirmation error")
	}
}

func TestServiceRollbackManifestRestore(t *testing.T) {
	service, def := newTestServiceWithClient(t, true, testManifestClient{})
	def.Spec.Options = Options{Apply: true, DryRun: false}
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true, Confirm: true})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	rolledBack, err := service.Rollback(context.Background(), RollbackInput{DeploymentRunID: result.Record.Run.ID, Confirm: true, ActorID: "tester"})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if rolledBack.Run.Status != domaindeployment.DeploymentRunRolledBack {
		t.Fatalf("status = %s", rolledBack.Run.Status)
	}
	if rolledBack.Rollback == nil || rolledBack.Rollback.Status != "succeeded" {
		t.Fatalf("rollback = %#v", rolledBack.Rollback)
	}
	assertHasDeploymentEvent(t, rolledBack.Events, EventDeploymentRollbackStarted)
	assertHasDeploymentEvent(t, rolledBack.Events, EventDeploymentRollbackSucceeded)
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

func TestServicePlansGitOpsDeploymentFromRepositoryCatalog(t *testing.T) {
	service, def := newGitOpsTestService(t)
	def.Spec.Target.RepositoryID = "repo-1"
	def.Spec.Target.RepoURL = ""
	def.Spec.Target.Revision = ""
	service.WithRepositoryCatalog(fakeRepositoryCatalog{repos: map[string]domainapp.Repository{
		"repo-1": {
			ID:            "repo-1",
			ProjectID:     "project-a",
			Name:          "platform-gitops",
			URL:           "https://example.com/platform/gitops.git",
			Provider:      "gitlab",
			DefaultBranch: "release-main",
			Enabled:       true,
		},
	}})

	result, err := service.Plan(context.Background(), CreateRunInput{Definition: def, ProjectID: "project-a"})
	if err != nil {
		t.Fatalf("plan gitops from repository catalog: %v", err)
	}
	plan := result.Record.GitOpsPlan
	if plan.RepositoryID != "repo-1" || plan.RepositoryName != "platform-gitops" || plan.RepositoryProvider != "gitlab" {
		t.Fatalf("repository metadata not attached to plan: %#v", plan)
	}
	if plan.RepoURL != "https://example.com/platform/gitops.git" || plan.Revision != "release-main" || result.Record.Plan.TargetContext != plan.RepoURL {
		t.Fatalf("repository resolution failed: plan=%#v deploymentPlan=%#v", plan, result.Record.Plan)
	}
	if !strings.Contains(strings.Join(plan.Warnings, "\n"), "repositoryId") {
		t.Fatalf("repositoryId warning missing: %#v", plan.Warnings)
	}
}

func TestServiceRejectsInaccessibleGitOpsRepositoryCatalogRecord(t *testing.T) {
	service, def := newGitOpsTestService(t)
	def.Spec.Target.RepositoryID = "repo-1"
	def.Spec.Target.RepoURL = ""
	service.WithRepositoryCatalog(fakeRepositoryCatalog{repos: map[string]domainapp.Repository{
		"repo-1": {ID: "repo-1", ProjectID: "project-b", Name: "other", URL: "https://example.com/other.git", DefaultBranch: "main", Enabled: true},
	}})
	if _, err := service.Plan(context.Background(), CreateRunInput{Definition: def, ProjectID: "project-a"}); err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("cross-project repository error = %v, want ErrInvalidInput", err)
	}

	def.Spec.Target.RepositoryID = "repo-disabled"
	service.WithRepositoryCatalog(fakeRepositoryCatalog{repos: map[string]domainapp.Repository{
		"repo-disabled": {ID: "repo-disabled", ProjectID: "project-a", Name: "disabled", URL: "https://example.com/disabled.git", DefaultBranch: "main", Enabled: false},
	}})
	if _, err := service.Plan(context.Background(), CreateRunInput{Definition: def, ProjectID: "project-a"}); err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("disabled repository error = %v, want ErrInvalidInput", err)
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

func TestServiceGitOpsDigestSubstitutionRequiresExplicitTargetFlag(t *testing.T) {
	service, def := newGitOpsTestService(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "apps/demo-springboot/dev/deployment.yaml")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(file, []byte("containers:\n  - name: app\n    image: old.example/demo:old\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	def.Spec.Artifacts[0].Reference = "registry.example.com/demo/demo-springboot:1.0.0"
	def.Spec.Artifacts[0].Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	def.Spec.Artifacts[0].Target.Substitute = true
	def.Spec.GitOps.WriteToWorkingTree = true
	def.Spec.GitOps.WorkingTree = dir
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("run gitops digest substitute: %v", err)
	}
	if len(result.Record.GitOpsDiff.Files) != 1 || !strings.Contains(result.Record.GitOpsDiff.Files[0].After, "@sha256:aaaaaaaa") {
		t.Fatalf("diff = %#v", result.Record.GitOpsDiff)
	}
}

func TestServiceGitOpsCommitAndPushGuard(t *testing.T) {
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
	def.Spec.GitOps.Commit = true
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("run gitops commit: %v", err)
	}
	if !result.Record.GitOpsCommit.Committed || result.Record.GitOpsCommit.Revision == "" {
		t.Fatalf("commit = %#v", result.Record.GitOpsCommit)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventGitOpsCommitCreated)

	def.Spec.GitOps.Push = true
	result, err = service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("push guard should fail run cleanly: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("unguarded push status = %s", result.Record.Run.Status)
	}
}

func TestServiceGitOpsRollbackRequiresConfirmation(t *testing.T) {
	service, def := newGitOpsTestService(t)
	def.Spec.GitOps.Rollback = true
	def.Spec.GitOps.RollbackRevision = "fake-previous"
	def.Spec.GitOps.WorkingTree = t.TempDir()
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("rollback guard should fail run cleanly: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunFailed {
		t.Fatalf("unguarded rollback status = %s", result.Record.Run.Status)
	}
	result, err = service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, Confirm: true})
	if err != nil {
		t.Fatalf("confirmed rollback: %v", err)
	}
	if result.Record.GitOpsRollback.Revision != "fake-previous" {
		t.Fatalf("rollback = %#v", result.Record.GitOpsRollback)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventGitOpsRollbackCompleted)
}

func TestServiceGitOpsSyncRequiresConfirmation(t *testing.T) {
	service, def := newGitOpsTestService(t)
	def.Spec.GitOps.Sync = true
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("sync should be skipped, not fail: %v", err)
	}
	if result.Record.ArgoCDSync.Requested {
		t.Fatalf("sync should be skipped without allow/confirm: %#v", result.Record.ArgoCDSync)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventArgoCDSyncSkipped)
}

func TestServiceGitOpsGuardedSync(t *testing.T) {
	service, def := newGitOpsTestService(t)
	def.Spec.GitOps.Sync = true
	def.Spec.GitOps.AllowSync = true
	def.Spec.GitOps.Wait = true
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowSync: true, Confirm: true})
	if err != nil {
		t.Fatalf("guarded sync: %v", err)
	}
	if !result.Record.ArgoCDSync.Requested || len(result.Record.ArgoCDWatch) == 0 {
		t.Fatalf("sync/watch = %#v %#v", result.Record.ArgoCDSync, result.Record.ArgoCDWatch)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventArgoCDSyncRequested)
	assertHasDeploymentEvent(t, result.Record.Events, EventArgoCDSyncCompleted)
	assertHasDeploymentEvent(t, result.Record.Events, EventArgoCDHealthChanged)
}

func TestServicePlansHostDeployment(t *testing.T) {
	service, def := newHostTestService()
	def.Spec.Host.BatchSize = 1
	def.Spec.Host.RestartCommand = "systemctl restart demo"
	def.Spec.Host.HealthChecks = []HostHealthCheck{{Type: "http", Target: "http://localhost:8080/healthz", TimeoutSeconds: 5}}
	def.Spec.Host.Hosts = append(def.Spec.Host.Hosts, Host{ID: "local-noop-host-2", Name: "local-noop-host-2", Address: "127.0.0.2", EnvironmentID: "dev"})
	result, err := service.Plan(context.Background(), CreateRunInput{Definition: def})
	if err != nil {
		t.Fatalf("plan host: %v", err)
	}
	if result.Record.HostPlan.DeployPath != "/opt/nivora/apps/demo" {
		t.Fatalf("host plan = %#v", result.Record.HostPlan)
	}
	if len(result.Record.HostPlan.Hosts) != 2 {
		t.Fatalf("host count = %d", len(result.Record.HostPlan.Hosts))
	}
	if result.Record.HostPlan.Hosts[1].BatchIndex != 2 || result.Record.HostPlan.BatchSize != 1 {
		t.Fatalf("batch plan = %#v", result.Record.HostPlan)
	}
	if len(result.Record.HostPlan.HealthChecks) != 1 || result.Record.HostPlan.RestartCommand == "" {
		t.Fatalf("health/restart plan = %#v", result.Record.HostPlan)
	}
	if result.Record.HostPlan.RollbackPlan.Executable {
		t.Fatalf("rollback plan should be non-destructive: %#v", result.Record.HostPlan.RollbackPlan)
	}
}

func TestServiceRunsHostDryRunNoop(t *testing.T) {
	service, def := newHostTestService()
	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("run host dry-run: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunSucceeded {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if len(result.Record.HostDetails) != 1 || result.Record.HostDetails[0].Status != "Succeeded" {
		t.Fatalf("host details = %#v", result.Record.HostDetails)
	}
	assertHasDeploymentEvent(t, result.Record.Events, EventHostDeploymentPlanCreated)
	assertHasDeploymentEvent(t, result.Record.Events, EventHostDeploymentStarted)
	assertHasDeploymentEvent(t, result.Record.Events, EventHostDeploymentHostCompleted)
	assertHasDeploymentEvent(t, result.Record.Events, EventHostDeploymentHealthCompleted)
	assertHasDeploymentEvent(t, result.Record.Events, EventHostRollbackPlanCreated)
}

func TestServiceRejectsHostRemoteWithoutConfirmation(t *testing.T) {
	service, def := newHostTestService()
	def.Spec.Options = Options{Apply: true, DryRun: false}
	def.Spec.Host.AllowRemoteHostDeploy = true
	def.Spec.Host.CredentialRef = "cred-host"

	_, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true})
	if err == nil {
		t.Fatal("expected remote confirmation error")
	}
}

func TestServiceRejectsHostRemoteWithoutCredential(t *testing.T) {
	service, def := newHostTestService()
	def.Spec.Options = Options{Apply: true, DryRun: false}
	def.Spec.Host.AllowRemoteHostDeploy = true

	_, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true, Confirm: true})
	if err == nil {
		t.Fatal("expected credential error")
	}
}

func TestServiceRunsGuardedHostApplyAndRollbackWithFakeExecutor(t *testing.T) {
	executor := &recordingHostExecutor{}
	service, def := newHostTestService()
	service.WithHostExecutor(executor)
	def.Spec.Options = Options{Apply: true, DryRun: false, Wait: true, TimeoutSeconds: 20}
	def.Spec.Host.AllowRemoteHostDeploy = true
	def.Spec.Host.CredentialRef = "cred-host"
	def.Spec.Host.RestartCommand = "systemctl restart demo"
	def.Spec.Host.BatchSize = 1
	def.Spec.Host.HealthChecks = []HostHealthCheck{{Type: "command", Command: "curl -fsS http://localhost:8080/healthz"}}

	result, err := service.CreateAndRun(context.Background(), CreateRunInput{Definition: def, AllowApply: true, Confirm: true, ActorID: "tester"})
	if err != nil {
		t.Fatalf("guarded host apply: %v", err)
	}
	if result.Record.Run.Status != domaindeployment.DeploymentRunSucceeded {
		t.Fatalf("status = %s", result.Record.Run.Status)
	}
	if executor.uploads != 1 || executor.executes != 1 || executor.healthChecks != 1 {
		t.Fatalf("executor calls = %#v", executor)
	}
	if !result.Record.HostDetails[0].RollbackReady || result.Record.HostDetails[0].HealthStatus != "Healthy" {
		t.Fatalf("host details = %#v", result.Record.HostDetails)
	}
	rolledBack, err := service.Rollback(context.Background(), RollbackInput{DeploymentRunID: result.Record.Run.ID, Confirm: true, ActorID: "tester"})
	if err != nil {
		t.Fatalf("host rollback: %v", err)
	}
	if rolledBack.Run.Status != domaindeployment.DeploymentRunRolledBack {
		t.Fatalf("rollback status = %s", rolledBack.Run.Status)
	}
	if executor.rollbacks != 1 {
		t.Fatalf("rollback calls = %d", executor.rollbacks)
	}
	assertHasDeploymentEvent(t, rolledBack.Events, EventHostRollbackCompleted)
}

func TestServiceStoresHostGroups(t *testing.T) {
	service, _ := newHostTestService()
	group, err := service.CreateHostGroup(context.Background(), HostGroup{
		Name:          "local-host-group",
		EnvironmentID: "dev",
		CredentialRef: "cred-host",
		Hosts:         []HostTarget{{Name: "local-noop-host", Address: "127.0.0.1"}},
	})
	if err != nil {
		t.Fatalf("create host group: %v", err)
	}
	if group.ID == "" || group.Hosts[0].CredentialRef != "cred-host" {
		t.Fatalf("group = %#v", group)
	}
	groups, err := service.ListHostGroups(context.Background())
	if err != nil {
		t.Fatalf("list host groups: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("groups = %#v", groups)
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

func newHostTestService() (*Service, Definition) {
	service := NewService(NewMemoryStore(), StaticManifestRenderer{}, testManifestClient{}, testPolicy{allowed: true}, testEventBus{}).
		WithHostExecutor(testHostExecutor{})
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Deployment",
		Metadata:   Metadata{Name: "demo-host-release"},
		Spec: Spec{
			Application: "demo",
			Environment: "dev",
			Target:      Target{Type: "host", Name: "local-host-group"},
			Artifact: Artifact{
				Name:      "demo",
				Type:      "binary",
				Reference: "./dist/demo.tar.gz",
			},
			Host: HostSpec{
				DeployPath:  "/opt/nivora/apps/demo",
				ServiceName: "demo",
				Strategy:    "symlink",
				HealthCheck: "http://localhost:8080/healthz",
				Hosts: []Host{{
					ID:            "local-noop-host",
					Name:          "local-noop-host",
					Address:       "127.0.0.1",
					EnvironmentID: "dev",
				}},
			},
			Options: Options{DryRun: true, Apply: false},
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

func assertPlanWarningContains(t *testing.T, warnings []string, needle string) {
	t.Helper()
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return
		}
	}
	t.Fatalf("missing warning containing %q in %#v", needle, warnings)
}

func writeDeploymentManifest(t *testing.T, def Definition, body string) {
	t.Helper()
	if len(def.Spec.Manifests) == 0 {
		t.Fatal("test definition has no manifest path")
	}
	if err := os.WriteFile(def.Spec.Manifests[0], []byte(body), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func manyConfigMapsManifest(count int) string {
	var b strings.Builder
	for i := 0; i < count; i++ {
		if i > 0 {
			b.WriteString("---\n")
		}
		b.WriteString(fmt.Sprintf("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm-%03d\n", i))
	}
	return b.String()
}

type fakeRepositoryCatalog struct {
	repos map[string]domainapp.Repository
	err   error
}

func (f fakeRepositoryCatalog) GetRepository(ctx context.Context, id string) (domainapp.Repository, error) {
	if err := ctx.Err(); err != nil {
		return domainapp.Repository{}, err
	}
	if f.err != nil {
		return domainapp.Repository{}, f.err
	}
	repository, ok := f.repos[id]
	if !ok {
		return domainapp.Repository{}, fmt.Errorf("repository %q not found", id)
	}
	return repository, nil
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
	dryRunErr   error
	applyErr    error
	rolloutErr  error
	rollbackErr error
}

type testSecurityScanner struct{}

func (testSecurityScanner) ScanArtifact(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "test"}, nil
}

func (testSecurityScanner) ScanManifest(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "test"}, nil
}

func (testSecurityScanner) ScanDeploymentPlan(ctx context.Context, request portsecurity.ScanRequest) (portsecurity.ScanResult, error) {
	return portsecurity.ScanResult{Scanner: "test"}, nil
}

func (testSecurityScanner) GetCapabilities(ctx context.Context) ([]portsecurity.Capability, error) {
	return nil, nil
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

func (c testManifestClient) Rollback(ctx context.Context, request ManifestRequest) (KubernetesRollbackResult, error) {
	if c.rollbackErr != nil {
		return KubernetesRollbackResult{}, c.rollbackErr
	}
	return KubernetesRollbackResult{Mode: "test", Message: "rollback ok", Resources: request.Plan.Resources}, nil
}

type testEventBus struct{}

func (testEventBus) Publish(ctx context.Context, evt event.Event) error { return nil }
func (testEventBus) Subscribe(ctx context.Context, eventType string) (<-chan event.Event, error) {
	ch := make(chan event.Event)
	close(ch)
	return ch, nil
}

type testGovernance struct {
	changeWindowAllowed bool
}

func (g testGovernance) RequestApproval(ctx context.Context, subjectType string, subjectID string, environmentID string, requestedBy string, reason string) (domainapproval.ApprovalRequest, error) {
	return domainapproval.ApprovalRequest{ID: "appr-test", SubjectType: subjectType, SubjectID: subjectID, EnvironmentID: environmentID, RequiredByPolicy: true, Status: domainapproval.StatusPending, RequestedBy: requestedBy, Reason: reason}, nil
}

func (g testGovernance) EvaluateChangeWindow(ctx context.Context, environmentID string) (domainapproval.ChangeWindowResult, error) {
	allowed := g.changeWindowAllowed
	reason := "change window denied"
	if allowed {
		reason = "change window allowed"
	}
	return domainapproval.ChangeWindowResult{WindowID: "cwin-test", EnvironmentID: environmentID, Allowed: allowed, Reason: reason}, nil
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

func (fakeWorkingTree) CurrentRevision(ctx context.Context, root string) (string, error) {
	return "fake-revision", nil
}

func (fakeWorkingTree) Commit(ctx context.Context, root string, message string, files []string) (portgitops.CommitResult, error) {
	return portgitops.CommitResult{Message: message, Revision: "fake-commit", Files: append([]string(nil), files...), Committed: true}, nil
}

func (fakeWorkingTree) Push(ctx context.Context, root string, remote string, branch string, allowPush bool) (portgitops.CommitResult, error) {
	if !allowPush {
		return portgitops.CommitResult{}, fmt.Errorf("push disabled")
	}
	return portgitops.CommitResult{Revision: "fake-commit", Pushed: true}, nil
}

func (fakeWorkingTree) CheckoutRevision(ctx context.Context, root string, revision string, confirm bool) (portgitops.CommitResult, error) {
	if !confirm {
		return portgitops.CommitResult{}, fmt.Errorf("rollback requires confirmation")
	}
	return portgitops.CommitResult{Revision: revision}, nil
}

var _ portgitops.WorkingTree = fakeWorkingTree{}

type testHostExecutor struct{}

func (testHostExecutor) Prepare(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return testHostResult(request, "prepared"), nil
}

func (testHostExecutor) Upload(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return testHostResult(request, "uploaded"), nil
}

func (testHostExecutor) Execute(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return testHostResult(request, "executed"), nil
}

func (testHostExecutor) HealthCheck(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return testHostResult(request, "health check ok"), nil
}

func (testHostExecutor) Rollback(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return testHostResult(request, "rollback skipped"), nil
}

func testHostResult(request portexecutor.HostDeploymentRequest, message string) portexecutor.HostDeploymentResult {
	return portexecutor.HostDeploymentResult{HostID: request.HostID, HostName: request.HostName, Status: "Succeeded", Message: message}
}

var _ portexecutor.HostExecutor = testHostExecutor{}

type recordingHostExecutor struct {
	uploads      int
	executes     int
	healthChecks int
	rollbacks    int
}

func (e *recordingHostExecutor) Prepare(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	return testHostResult(request, "prepared"), nil
}

func (e *recordingHostExecutor) Upload(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	e.uploads++
	return testHostResult(request, "uploaded"), nil
}

func (e *recordingHostExecutor) Execute(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	e.executes++
	if request.RestartCommand == "" {
		return portexecutor.HostDeploymentResult{}, fmt.Errorf("restart command missing")
	}
	return testHostResult(request, "executed"), nil
}

func (e *recordingHostExecutor) HealthCheck(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	e.healthChecks++
	if request.HealthCheckType != "command" {
		return portexecutor.HostDeploymentResult{}, fmt.Errorf("health check type = %s", request.HealthCheckType)
	}
	return testHostResult(request, "health check ok"), nil
}

func (e *recordingHostExecutor) Rollback(ctx context.Context, request portexecutor.HostDeploymentRequest) (portexecutor.HostDeploymentResult, error) {
	e.rollbacks++
	if !request.Confirmed {
		return portexecutor.HostDeploymentResult{}, fmt.Errorf("rollback requires confirmation")
	}
	return testHostResult(request, "host rollback ok"), nil
}

var _ portexecutor.HostExecutor = (*recordingHostExecutor)(nil)

type fakeArgoCDProvider struct{}

func (fakeArgoCDProvider) ValidateCredential(ctx context.Context, credential portargocd.CredentialRef) error {
	return nil
}

func (fakeArgoCDProvider) GetApplicationStatus(ctx context.Context, applicationName string) (portargocd.ApplicationStatus, error) {
	return portargocd.ApplicationStatus{ApplicationName: applicationName, SyncStatus: "Synced", HealthStatus: "Healthy", Message: "test status", Resources: []portargocd.ResourceStatus{{Kind: "Deployment", Name: applicationName, Health: "Healthy", SyncStatus: "Synced"}}}, nil
}

func (fakeArgoCDProvider) GetApplicationResources(ctx context.Context, applicationName string) ([]portargocd.ResourceStatus, error) {
	return []portargocd.ResourceStatus{{Kind: "Deployment", Name: applicationName, Health: "Healthy", SyncStatus: "Synced"}}, nil
}

func (fakeArgoCDProvider) GetApplicationHistory(ctx context.Context, applicationName string) ([]portargocd.ApplicationStatus, error) {
	status, _ := fakeArgoCDProvider{}.GetApplicationStatus(ctx, applicationName)
	return []portargocd.ApplicationStatus{status}, nil
}

func (fakeArgoCDProvider) SyncApplication(ctx context.Context, request portargocd.SyncRequest) (portargocd.SyncResult, error) {
	return portargocd.SyncResult{ApplicationName: request.ApplicationName, Requested: request.AllowSync && request.Confirmed, Started: true, Completed: true, SyncStatus: "Synced", HealthStatus: "Healthy", Message: "test sync"}, nil
}

func (fakeArgoCDProvider) WatchApplicationStatus(ctx context.Context, applicationName string, timeoutSeconds int) ([]portargocd.ApplicationStatus, error) {
	status, _ := fakeArgoCDProvider{}.GetApplicationStatus(ctx, applicationName)
	return []portargocd.ApplicationStatus{status}, nil
}
