package releaseorchestration

import (
	"context"
	"errors"
	"testing"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domaindeployment "github.com/sevoniva/nivora/internal/domain/deployment"
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

func TestParseDefinition(t *testing.T) {
	def, err := LoadDefinitionFile("../../../examples/releases/sequential-release.yaml")
	if err != nil {
		t.Fatalf("LoadDefinitionFile() error = %v", err)
	}
	if def.Metadata.Name == "" || len(def.Spec.Targets) != 2 {
		t.Fatalf("unexpected definition: %#v", def)
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
