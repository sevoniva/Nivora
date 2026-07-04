package releaseorchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
	"github.com/sevoniva/nivora/internal/domain/environment"
	"github.com/sevoniva/nivora/internal/domain/release"
	"github.com/sevoniva/nivora/internal/ports/policy"
	portsecurity "github.com/sevoniva/nivora/internal/ports/security"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	policyusecase "github.com/sevoniva/nivora/internal/usecase/policy"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

func TestPlanMultipleTargets(t *testing.T) {
	service := newTestService(allowPolicy{})
	record, err := service.Plan(context.Background(), PlanInput{Definition: testDefinition(false, StrategySequential)})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(record.Plan.Targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(record.Plan.Targets))
	}
	if len(record.Plan.DeploymentPlans) != 2 {
		t.Fatalf("deployment plans = %d, want 2", len(record.Plan.DeploymentPlans))
	}
	if record.Plan.Ordering[0] != "dev-yaml" || record.Plan.Ordering[1] != "audit-only" {
		t.Fatalf("ordering = %#v", record.Plan.Ordering)
	}
	if record.Release.Status != string(release.ReleaseStatusPlanning) {
		t.Fatalf("release status = %q, want %q", record.Release.Status, release.ReleaseStatusPlanning)
	}
}

func TestPlanAndDeployPersistProjectScope(t *testing.T) {
	service := newTestService(allowPolicy{})
	plan, err := service.Plan(context.Background(), PlanInput{Definition: testDefinition(false, StrategySequential), ProjectID: " project-a "})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	for _, target := range plan.Plan.Targets {
		if target.ProjectID != "project-a" {
			t.Fatalf("target %s project id = %q", target.Name, target.ProjectID)
		}
	}
	record, err := service.Deploy(context.Background(), DeployInput{Definition: testDefinition(false, StrategySequential), ProjectID: "project-a"})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	for _, target := range record.Plan.Targets {
		if target.ProjectID != "project-a" {
			t.Fatalf("execution target %s project id = %q", target.Name, target.ProjectID)
		}
	}
	for _, deployment := range record.Deployments {
		if deployment.Environment.ProjectID != "project-a" || deployment.Target.ProjectID != "project-a" {
			t.Fatalf("deployment scope not propagated: environment=%q target=%q", deployment.Environment.ProjectID, deployment.Target.ProjectID)
		}
	}
}

func TestSequentialExecutionSuccess(t *testing.T) {
	service := newTestService(allowPolicy{})
	record, err := service.Deploy(context.Background(), DeployInput{Definition: testDefinition(false, StrategySequential)})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if record.Execution.Status != ExecutionSucceeded {
		t.Fatalf("status = %s, want %s", record.Execution.Status, ExecutionSucceeded)
	}
	if record.Release.Status != string(release.ReleaseStatusSucceeded) {
		t.Fatalf("release status = %q, want %q", record.Release.Status, release.ReleaseStatusSucceeded)
	}
	if len(record.Execution.DeploymentRunIDs) != 1 {
		t.Fatalf("deployment run ids = %d, want 1", len(record.Execution.DeploymentRunIDs))
	}
	if len(record.Events) == 0 {
		t.Fatal("expected release execution events")
	}
	timeline, err := service.Timeline(context.Background(), record.Execution.ID)
	if err != nil {
		t.Fatalf("Timeline() error = %v", err)
	}
	if len(timeline) == 0 {
		t.Fatal("expected aggregate timeline")
	}
}

func TestSequentialExecutionPartialFailure(t *testing.T) {
	def := testDefinition(true, StrategySequential)
	def.Spec.ContinueOnFailure = true
	service := newTestService(allowPolicy{})
	record, err := service.Deploy(context.Background(), DeployInput{Definition: def})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if record.Execution.Status != ExecutionPartiallySucceeded {
		t.Fatalf("status = %s, want %s", record.Execution.Status, ExecutionPartiallySucceeded)
	}
	if record.Release.Status != string(release.ReleaseStatusFailed) {
		t.Fatalf("release status = %q, want %q", record.Release.Status, release.ReleaseStatusFailed)
	}
}

func TestPolicyDenied(t *testing.T) {
	service := newTestService(denyPolicy{})
	record, err := service.Deploy(context.Background(), DeployInput{Definition: testDefinition(false, StrategySequential)})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if record.Execution.Status != ExecutionFailed {
		t.Fatalf("status = %s, want %s", record.Execution.Status, ExecutionFailed)
	}
	if record.Execution.Reason == "" {
		t.Fatal("expected denial reason")
	}
	if record.Release.Status != string(release.ReleaseStatusFailed) {
		t.Fatalf("release status = %q, want %q", record.Release.Status, release.ReleaseStatusFailed)
	}
}

func TestApprovalRequiredCreatesApprovalRequest(t *testing.T) {
	def := testDefinition(false, StrategySequential)
	def.Spec.ApprovalRequired = true
	service := newTestService(allowPolicy{}).WithGovernance(testGovernance{})
	record, err := service.Deploy(context.Background(), DeployInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if record.Execution.Status != ExecutionWaitingApproval {
		t.Fatalf("status = %s", record.Execution.Status)
	}
	if record.Release.Status != string(release.ReleaseStatusWaitingApproval) {
		t.Fatalf("release status = %q, want %q", record.Release.Status, release.ReleaseStatusWaitingApproval)
	}
	if record.Approval.Status != domainapproval.StatusPending || record.Approval.SubjectID != record.Execution.ID {
		t.Fatalf("approval = %#v", record.Approval)
	}
}

func TestSavedPolicyAttachmentRequiresReleaseApproval(t *testing.T) {
	catalog := policyusecase.NewService(policyusecase.NewMemoryStore())
	policyDef, err := catalog.Create(context.Background(), policyusecase.CreateInput{
		ID:            "policy-release-approval",
		Name:          "Release digest approval",
		Mode:          "require_approval",
		RequireDigest: true,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if _, err := catalog.Attach(context.Background(), policyDef.ID, policyusecase.AttachInput{ScopeType: "environment", ScopeID: "dev"}); err != nil {
		t.Fatalf("attach policy: %v", err)
	}
	service := newTestService(allowPolicy{}).
		WithSecurity(securityusecase.NewService(securityusecase.NewMemoryStore(), testSecurityScanner{}, nil, nil)).
		WithPolicyCatalog(catalog).
		WithGovernance(testGovernance{})

	record, err := service.Deploy(context.Background(), DeployInput{Definition: testDefinition(false, StrategySequential), ProjectID: "project-a", ActorID: "tester"})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if record.Execution.Status != ExecutionWaitingApproval {
		t.Fatalf("status = %s, want WaitingApproval", record.Execution.Status)
	}
	if record.Security.Policy.PolicyID != policyDef.ID {
		t.Fatalf("policy id = %q, want %q", record.Security.Policy.PolicyID, policyDef.ID)
	}
	if record.Approval.Status != domainapproval.StatusPending || record.Approval.Reason != "artifact digest is required" {
		t.Fatalf("approval = %#v", record.Approval)
	}
}

func TestSavedPolicyAttachmentDeniesReleaseExecution(t *testing.T) {
	catalog := policyusecase.NewService(policyusecase.NewMemoryStore())
	policyDef, err := catalog.Create(context.Background(), policyusecase.CreateInput{
		ID:            "policy-release-deny",
		Name:          "Release digest deny",
		Mode:          "deny",
		RequireDigest: true,
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if _, err := catalog.Attach(context.Background(), policyDef.ID, policyusecase.AttachInput{ScopeType: "environment", ScopeID: "dev"}); err != nil {
		t.Fatalf("attach policy: %v", err)
	}
	service := newTestService(allowPolicy{}).
		WithSecurity(securityusecase.NewService(securityusecase.NewMemoryStore(), testSecurityScanner{}, nil, nil)).
		WithPolicyCatalog(catalog)

	record, err := service.Deploy(context.Background(), DeployInput{Definition: testDefinition(false, StrategySequential), ProjectID: "project-a", ActorID: "tester"})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if record.Execution.Status != ExecutionFailed {
		t.Fatalf("status = %s, want Failed", record.Execution.Status)
	}
	if record.Security.Policy.PolicyID != policyDef.ID {
		t.Fatalf("policy id = %q, want %q", record.Security.Policy.PolicyID, policyDef.ID)
	}
}

func TestApprovalApprovedResumesReleaseExecution(t *testing.T) {
	def := testDefinition(false, StrategySequential)
	def.Spec.ApprovalRequired = true
	service := newTestService(allowPolicy{}).WithGovernance(testGovernance{})
	record, err := service.Deploy(context.Background(), DeployInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	resumed, err := service.ApplyApprovalDecision(context.Background(), record.Execution.ID, domainapproval.ApprovalRequest{
		ID:          record.Approval.ID,
		SubjectType: domainapproval.SubjectRelease,
		SubjectID:   record.Execution.ID,
		Status:      domainapproval.StatusApproved,
	}, "reviewer")
	if err != nil {
		t.Fatalf("resume after approval: %v", err)
	}
	if resumed.Execution.Status != ExecutionSucceeded {
		t.Fatalf("status = %s", resumed.Execution.Status)
	}
	if resumed.Release.Status != string(release.ReleaseStatusSucceeded) {
		t.Fatalf("release status = %q, want %q", resumed.Release.Status, release.ReleaseStatusSucceeded)
	}
}

func TestApprovalRejectedStopsReleaseExecution(t *testing.T) {
	def := testDefinition(false, StrategySequential)
	def.Spec.ApprovalRequired = true
	service := newTestService(allowPolicy{}).WithGovernance(testGovernance{})
	record, err := service.Deploy(context.Background(), DeployInput{Definition: def, ActorID: "tester"})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	stopped, err := service.ApplyApprovalDecision(context.Background(), record.Execution.ID, domainapproval.ApprovalRequest{
		ID:          record.Approval.ID,
		SubjectType: domainapproval.SubjectRelease,
		SubjectID:   record.Execution.ID,
		Status:      domainapproval.StatusRejected,
	}, "reviewer")
	if err != nil {
		t.Fatalf("reject approval: %v", err)
	}
	if stopped.Execution.Status != ExecutionFailed {
		t.Fatalf("status = %s", stopped.Execution.Status)
	}
	if stopped.Release.Status != string(release.ReleaseStatusFailed) {
		t.Fatalf("release status = %q, want %q", stopped.Release.Status, release.ReleaseStatusFailed)
	}
}

func TestCancelExecution(t *testing.T) {
	service := newTestService(allowPolicy{})
	record, err := service.Deploy(context.Background(), DeployInput{Definition: testDefinition(false, StrategyPlanOnly)})
	if err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if _, err := service.Cancel(context.Background(), record.Execution.ID, "tester"); !errors.Is(err, ErrExecutionTerminal) {
		t.Fatalf("Cancel() error = %v, want terminal", err)
	}
}

func TestCancelExecutionCancelsLinkedNonTerminalDeploymentRuns(t *testing.T) {
	store := NewMemoryStore()
	deployments := &cancelSpyDeploymentService{
		records: map[string]deploymentusecase.RunRecord{
			"drun-running": {Run: domaindeployment.DeploymentRun{ID: "drun-running", Status: domaindeployment.DeploymentRunDeploying}},
			"drun-done":    {Run: domaindeployment.DeploymentRun{ID: "drun-done", Status: domaindeployment.DeploymentRunSucceeded}},
		},
	}
	service := NewService(store, fakeArtifactService{}, deployments, allowPolicy{}, nil)
	now := time.Now()
	record := ExecutionRecord{
		Release: release.Release{ID: "rel-cascade", Name: "cascade", Version: "1.0.0"},
		Plan: ReleasePlan{
			ID:        "rplan-cascade",
			ReleaseID: "rel-cascade",
		},
		Execution: ReleaseExecution{
			ID:               "rexec-cascade",
			ReleaseID:        "rel-cascade",
			Status:           ExecutionRunning,
			DeploymentRunIDs: []string{"drun-running", "drun-done"},
			Targets: []TargetExecution{
				{TargetName: "running", DeploymentRunID: "drun-running", Status: ExecutionRunning},
				{TargetName: "done", DeploymentRunID: "drun-done", Status: ExecutionSucceeded},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		Deployments: []deploymentusecase.RunRecord{
			{Run: domaindeployment.DeploymentRun{ID: "drun-running", Status: domaindeployment.DeploymentRunDeploying}},
			{Run: domaindeployment.DeploymentRun{ID: "drun-done", Status: domaindeployment.DeploymentRunSucceeded}},
		},
	}
	if err := store.SaveExecution(context.Background(), record); err != nil {
		t.Fatalf("save execution: %v", err)
	}

	canceled, err := service.Cancel(context.Background(), record.Execution.ID, "operator")
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if canceled.Execution.Status != ExecutionCanceled {
		t.Fatalf("execution status = %s, want %s", canceled.Execution.Status, ExecutionCanceled)
	}
	if canceled.Release.Status != string(release.ReleaseStatusCanceled) {
		t.Fatalf("release status = %q, want %q", canceled.Release.Status, release.ReleaseStatusCanceled)
	}
	if got := deployments.cancelCount("drun-running"); got != 1 {
		t.Fatalf("running deployment cancel count = %d, want 1", got)
	}
	if got := deployments.cancelCount("drun-done"); got != 0 {
		t.Fatalf("terminal deployment cancel count = %d, want 0", got)
	}
	for _, target := range canceled.Execution.Targets {
		switch target.DeploymentRunID {
		case "drun-running":
			if target.Status != ExecutionCanceled {
				t.Fatalf("running target status = %s, want %s", target.Status, ExecutionCanceled)
			}
		case "drun-done":
			if target.Status != ExecutionSucceeded {
				t.Fatalf("done target status = %s, want %s", target.Status, ExecutionSucceeded)
			}
		}
	}
}

func TestCancelExecutionsForReleaseCancelsOnlyNonTerminalRecords(t *testing.T) {
	service := newTestService(allowPolicy{}).WithGovernance(testGovernance{})
	waitingDef := testDefinition(false, StrategySequential)
	waitingDef.Spec.ApprovalRequired = true
	waiting, err := service.Deploy(context.Background(), DeployInput{Definition: waitingDef, ActorID: "tester"})
	if err != nil {
		t.Fatalf("create waiting execution: %v", err)
	}
	terminal, err := service.Deploy(context.Background(), DeployInput{Definition: testDefinition(false, StrategyPlanOnly), ActorID: "tester"})
	if err != nil {
		t.Fatalf("create terminal execution: %v", err)
	}

	canceled, err := service.CancelExecutionsForRelease(context.Background(), waiting.Execution.ReleaseID, "operator")
	if err != nil {
		t.Fatalf("CancelExecutionsForRelease() error = %v", err)
	}
	if len(canceled) != 1 {
		t.Fatalf("canceled executions = %d, want 1", len(canceled))
	}
	if canceled[0].Execution.ID != waiting.Execution.ID || canceled[0].Execution.Status != ExecutionCanceled {
		t.Fatalf("unexpected canceled execution: %#v", canceled[0].Execution)
	}
	for _, target := range canceled[0].Execution.Targets {
		if target.Status != ExecutionCanceled {
			t.Fatalf("target %s status = %s, want %s", target.TargetName, target.Status, ExecutionCanceled)
		}
	}
	loadedTerminal, err := service.GetExecution(context.Background(), terminal.Execution.ID)
	if err != nil {
		t.Fatalf("load terminal execution: %v", err)
	}
	if loadedTerminal.Execution.Status != ExecutionSucceeded {
		t.Fatalf("terminal execution status = %s, want %s", loadedTerminal.Execution.Status, ExecutionSucceeded)
	}
}

func TestParseDefinition(t *testing.T) {
	def, err := LoadDefinitionFile("../../../examples/releases/sequential-release.yaml")
	if err != nil {
		t.Fatalf("LoadDefinitionFile() error = %v", err)
	}
	if def.Metadata.Name == "" || len(def.Spec.Targets) != 2 {
		t.Fatalf("unexpected definition: %#v", def)
	}
}

func TestDefinitionAllowsHostTarget(t *testing.T) {
	def := testDefinition(false, StrategySequential)
	def.Spec.Targets = []TargetSpec{{
		Name:  "host-noop",
		Type:  "host",
		Order: 1,
		Deployment: deploymentusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Deployment",
			Metadata:   deploymentusecase.Metadata{Name: "host-noop"},
			Spec: deploymentusecase.Spec{
				Application: "demo",
				Environment: "dev",
				Target: deploymentusecase.Target{
					Type: "host",
					Name: "host-noop",
				},
				Artifact: deploymentusecase.Artifact{Name: "demo", Type: "binary", Reference: "./dist/demo.tar.gz"},
				Host: deploymentusecase.HostSpec{
					DeployPath: "/opt/nivora/apps/demo",
					DryRun:     true,
				},
			},
		},
	}}
	if err := def.Validate(); err != nil {
		t.Fatalf("host target should validate: %v", err)
	}
}

func newTestService(policyEngine policy.Engine) *Service {
	return NewService(NewMemoryStore(), fakeArtifactService{}, fakeDeploymentService{}, policyEngine, nil)
}

func testDefinition(fail bool, strategy ExecutionStrategy) Definition {
	targets := []TargetSpec{
		{
			Name:  "dev-yaml",
			Type:  "kubernetes-yaml",
			Order: 1,
			Deployment: deploymentusecase.Definition{
				APIVersion: "nivora.io/v1alpha1",
				Kind:       "Deployment",
				Metadata:   deploymentusecase.Metadata{Name: "test-yaml"},
				Spec: deploymentusecase.Spec{
					Application: "demo",
					Environment: "dev",
					Target: deploymentusecase.Target{
						Type:      "kubernetes-yaml",
						Name:      "dev-yaml",
						Namespace: "default",
					},
					Artifacts: []deploymentusecase.Artifact{{Name: "demo", Type: "image", Reference: "registry.example.com/demo/app:1.0.0"}},
					Manifests: []string{"../../../examples/yaml/deployment.yaml"},
					Options:   deploymentusecase.Options{DryRun: true, Apply: false},
				},
			},
		},
		{Name: "audit-only", Type: "noop", Order: 2},
	}
	if fail {
		targets = append(targets, TargetSpec{
			Name:  "bad-yaml",
			Type:  "kubernetes-yaml",
			Order: 2,
			Deployment: deploymentusecase.Definition{
				APIVersion: "nivora.io/v1alpha1",
				Kind:       "Deployment",
				Metadata:   deploymentusecase.Metadata{Name: "bad-yaml"},
				Spec: deploymentusecase.Spec{
					Application: "demo",
					Environment: "dev",
					Target: deploymentusecase.Target{
						Type:      "kubernetes-yaml",
						Name:      "bad-yaml",
						Namespace: "default",
					},
					Manifests: []string{"../../../examples/deployments/missing.yaml"},
					Options:   deploymentusecase.Options{DryRun: true, Apply: false},
				},
			},
		})
	}
	return Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "ReleaseOrchestration",
		Metadata:   Metadata{Name: "test-release-orchestration"},
		Spec: Spec{
			Environment:       "dev",
			Strategy:          strategy,
			ContinueOnFailure: fail,
			Release: artifactusecase.ReleaseDefinition{
				APIVersion: "nivora.io/v1alpha1",
				Kind:       "Release",
				Metadata:   artifactusecase.ReleaseMetadata{Name: "demo"},
				Spec: artifactusecase.ReleaseSpec{
					Version:     "1.0.0",
					Application: "demo",
					Environment: "dev",
					Artifacts: []artifactusecase.ReleaseArtifactSpec{{
						Name:      "demo",
						Type:      "image",
						Role:      "application",
						Required:  true,
						Reference: "registry.example.com/demo/app:1.0.0",
					}},
				},
			},
			Targets: targets,
		},
	}
}

type allowPolicy struct{}

func (allowPolicy) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	return policy.Result{Allowed: true, Reasons: []string{"allowed"}}, nil
}

type denyPolicy struct{}

func (denyPolicy) Evaluate(ctx context.Context, request policy.Request) (policy.Result, error) {
	return policy.Result{Allowed: false, Reasons: []string{"denied by test policy"}}, nil
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

type testGovernance struct{}

func (testGovernance) RequestApproval(ctx context.Context, subjectType string, subjectID string, environmentID string, requestedBy string, reason string) (domainapproval.ApprovalRequest, error) {
	return domainapproval.ApprovalRequest{ID: "appr-release-test", SubjectType: subjectType, SubjectID: subjectID, EnvironmentID: environmentID, RequiredByPolicy: true, Status: domainapproval.StatusPending, RequestedBy: requestedBy, Reason: reason}, nil
}

type fakeArtifactService struct{}

func (fakeArtifactService) CreateRelease(ctx context.Context, input artifactusecase.CreateReleaseInput) (artifactusecase.ReleaseRecord, error) {
	now := time.Now()
	rel := release.Release{
		ID:            "rel-test",
		Name:          input.Definition.Metadata.Name,
		Version:       input.Definition.Spec.Version,
		ApplicationID: input.Definition.Spec.Application,
		EnvironmentID: input.Definition.Spec.Environment,
		Status:        string(release.ReleaseStatusReady),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	record := artifactusecase.ReleaseRecord{Release: rel}
	for _, item := range input.Definition.Spec.Artifacts {
		record.Artifacts = append(record.Artifacts, domainartifact.Artifact{
			ID:        "artifact-" + item.Name,
			Type:      domainartifact.ArtifactType(item.Type),
			Name:      item.Name,
			Reference: item.Reference,
			CreatedAt: now,
		})
		record.Bindings = append(record.Bindings, release.ReleaseArtifact{
			ID:        "binding-" + item.Name,
			ReleaseID: rel.ID,
			Name:      item.Name,
			Type:      item.Type,
			Reference: item.Reference,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	return record, nil
}

func (fakeArtifactService) GetRelease(ctx context.Context, id string) (artifactusecase.ReleaseRecord, error) {
	return artifactusecase.ReleaseRecord{Release: release.Release{ID: id, Name: id, Version: "test"}}, nil
}

func (fakeArtifactService) UpdateReleaseStatus(ctx context.Context, id string, status release.ReleaseStatus, actorID string, reason string) (artifactusecase.ReleaseRecord, error) {
	return artifactusecase.ReleaseRecord{Release: release.Release{ID: id, Name: id, Version: "test", Status: string(status), UpdatedAt: time.Now()}}, nil
}

type fakeDeploymentService struct{}

func (fakeDeploymentService) Plan(ctx context.Context, input deploymentusecase.CreateRunInput) (deploymentusecase.CreateRunResult, error) {
	return deploymentusecase.CreateRunResult{Record: deploymentusecase.RunRecord{
		Plan: deploymentusecase.DeploymentPlan{
			DeploymentRunID: "plan-" + input.Definition.Metadata.Name,
			TargetType:      input.Definition.Spec.Target.Type,
			Namespace:       input.Definition.Spec.Target.Namespace,
			ManifestCount:   len(input.Definition.Spec.Manifests),
			Actions:         []string{"test plan"},
			DiffSummary:     "test diff",
		},
	}}, nil
}

func (fakeDeploymentService) CreateAndRun(ctx context.Context, input deploymentusecase.CreateRunInput) (deploymentusecase.CreateRunResult, error) {
	status := domaindeployment.DeploymentRunSucceeded
	reason := ""
	for _, manifest := range input.Definition.Spec.Manifests {
		if manifest == "../../../examples/deployments/missing.yaml" {
			status = domaindeployment.DeploymentRunFailed
			reason = "manifest missing"
		}
	}
	return deploymentusecase.CreateRunResult{Record: deploymentusecase.RunRecord{
		Environment: environment.Environment{ProjectID: input.ProjectID},
		Target:      environment.ReleaseTarget{ProjectID: input.ProjectID},
		Run: domaindeployment.DeploymentRun{
			ID:         "drun-" + input.Definition.Metadata.Name,
			TargetType: input.Definition.Spec.Target.Type,
			Status:     status,
			Reason:     reason,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		Plan: deploymentusecase.DeploymentPlan{
			DeploymentRunID: "drun-" + input.Definition.Metadata.Name,
			TargetType:      input.Definition.Spec.Target.Type,
		},
	}}, nil
}

func (fakeDeploymentService) Cancel(ctx context.Context, id string, actorID string) (deploymentusecase.RunRecord, error) {
	return deploymentusecase.RunRecord{Run: domaindeployment.DeploymentRun{ID: id, Status: domaindeployment.DeploymentRunCanceled}}, nil
}

func (fakeDeploymentService) Timeline(ctx context.Context, id string) ([]deploymentusecase.TimelineEntry, error) {
	return []deploymentusecase.TimelineEntry{{Type: "devops.deployment.succeeded", Time: time.Now(), Subject: id, Status: "Succeeded"}}, nil
}

type cancelSpyDeploymentService struct {
	fakeDeploymentService
	records  map[string]deploymentusecase.RunRecord
	canceled []string
}

func (s *cancelSpyDeploymentService) Cancel(ctx context.Context, id string, actorID string) (deploymentusecase.RunRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return deploymentusecase.RunRecord{}, deploymentusecase.ErrRunNotFound
	}
	if record.Run.Status == domaindeployment.DeploymentRunSucceeded || record.Run.Status == domaindeployment.DeploymentRunFailed ||
		record.Run.Status == domaindeployment.DeploymentRunCanceled || record.Run.Status == domaindeployment.DeploymentRunRolledBack {
		return record, deploymentusecase.ErrRunTerminal
	}
	s.canceled = append(s.canceled, id)
	record.Run.Status = domaindeployment.DeploymentRunCanceled
	s.records[id] = record
	return record, nil
}

func (s *cancelSpyDeploymentService) cancelCount(id string) int {
	count := 0
	for _, item := range s.canceled {
		if item == id {
			count++
		}
	}
	return count
}
