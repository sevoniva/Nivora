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
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
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
	if record.Approval.Status != domainapproval.StatusPending || record.Approval.SubjectID != record.Execution.ID {
		t.Fatalf("approval = %#v", record.Approval)
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
		Status:        "Created",
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

func (fakeDeploymentService) Timeline(ctx context.Context, id string) ([]deploymentusecase.TimelineEntry, error) {
	return []deploymentusecase.TimelineEntry{{Type: "devops.deployment.succeeded", Time: time.Now(), Subject: id, Status: "Succeeded"}}, nil
}
