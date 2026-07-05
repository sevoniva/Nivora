package workflow

import (
	"strings"
	"testing"

	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func TestWorkflowPlanValidMatrixDAG(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: go-ci
  labels:
    team: platform
on:
  - manual
  - push
env:
  GOFLAGS: -mod=mod
jobs:
  test:
    runsOn: [self-hosted, shell]
    strategy:
      matrix:
        go: ["1.22", "1.23"]
        os: [linux]
    steps:
      - name: test
        run: go test ./...
  build:
    needs: [test]
    runsOn: [self-hosted, shell]
    steps:
      - name: build
        run: go build ./...
artifacts:
  - name: binaries
    path: dist/
cache:
  - key: gomod
    path: [go.sum]
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	plan, err := PlanDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("plan workflow: %v", err)
	}
	if !plan.ConversionReady {
		t.Fatalf("expected conversion ready: %#v", plan)
	}
	if len(plan.MatrixExpansions) != 2 {
		t.Fatalf("matrix expansions = %#v", plan.MatrixExpansions)
	}
	if len(plan.Edges) != 1 || plan.Edges[0].From != "test" || plan.Edges[0].To != "build" {
		t.Fatalf("edges = %#v", plan.Edges)
	}
	if len(plan.ArtifactOutputs) != 1 || plan.ArtifactOutputs[0].Name != "binaries" {
		t.Fatalf("artifacts = %#v", plan.ArtifactOutputs)
	}
	conversion, err := ToPipelineDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("convert workflow: %v", err)
	}
	if conversion.Definition.Kind != "Pipeline" {
		t.Fatalf("converted kind = %q", conversion.Definition.Kind)
	}
	if len(conversion.Definition.Spec.Stages) != 2 {
		t.Fatalf("converted stages = %#v", conversion.Definition.Spec.Stages)
	}
}

func TestWorkflowPlanRejectsDependencyCycle(t *testing.T) {
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Workflow",
		Metadata:   Metadata{Name: "cycle"},
		Jobs: map[string]Job{
			"a": {Needs: []string{"b"}, Steps: []Step{{Run: "echo a"}}},
			"b": {Needs: []string{"a"}, Steps: []Step{{Run: "echo b"}}},
		},
	}
	_, err := PlanDefinition(def, PlanOptions{})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestWorkflowPlanRejectsMissingNeedsTarget(t *testing.T) {
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Workflow",
		Metadata:   Metadata{Name: "missing"},
		Jobs: map[string]Job{
			"a": {Needs: []string{"b"}, Steps: []Step{{Run: "echo a"}}},
		},
	}
	_, err := PlanDefinition(def, PlanOptions{})
	if err == nil || !strings.Contains(err.Error(), "unknown job") {
		t.Fatalf("expected missing needs error, got %v", err)
	}
}

func TestWorkflowPlanRejectsMatrixLimit(t *testing.T) {
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Workflow",
		Metadata:   Metadata{Name: "matrix-limit"},
		Jobs: map[string]Job{
			"test": {
				Strategy: Strategy{Matrix: Matrix{Values: map[string][]string{
					"go": {"1.21", "1.22", "1.23"},
					"os": {"linux", "darwin"},
				}}},
				Steps: []Step{{Run: "go test ./..."}},
			},
		},
	}
	_, err := PlanDefinition(def, PlanOptions{MaxMatrixSize: 4})
	if err == nil || !strings.Contains(err.Error(), "matrix expands") {
		t.Fatalf("expected matrix limit error, got %v", err)
	}
}

func TestWorkflowPlanWarnsUnsupportedUsesAndBlocksConversion(t *testing.T) {
	def := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Workflow",
		Metadata:   Metadata{Name: "uses"},
		Jobs: map[string]Job{
			"test": {Steps: []Step{{Name: "external", Uses: "actions/checkout@v4"}}},
		},
	}
	plan, err := PlanDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("plan uses workflow: %v", err)
	}
	if plan.ConversionReady {
		t.Fatal("expected conversion not ready for uses step")
	}
	if len(plan.UnsupportedFeatures) == 0 {
		t.Fatalf("expected unsupported features warning: %#v", plan)
	}
	if _, err := ToPipelineDefinition(def, PlanOptions{}); err == nil {
		t.Fatal("expected conversion to fail for uses step")
	}
}

func TestWorkflowPlanRejectsSecretLikeEnvAndRedactsSafeRefs(t *testing.T) {
	unsafe := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Workflow",
		Metadata:   Metadata{Name: "unsafe-env"},
		Jobs: map[string]Job{
			"test": {Env: map[string]string{"NIVORA_TOKEN": "plain-value"}, Steps: []Step{{Run: "echo test"}}},
		},
	}
	if _, err := PlanDefinition(unsafe, PlanOptions{}); err == nil {
		t.Fatal("expected secret-like env to be rejected")
	}

	safe := Definition{
		APIVersion: "nivora.io/v1alpha1",
		Kind:       "Workflow",
		Metadata:   Metadata{Name: "safe-env"},
		Jobs: map[string]Job{
			"test": {Env: map[string]string{"NIVORA_TOKEN": "secretRef:nivora-token"}, Steps: []Step{{Run: "echo test"}}},
		},
	}
	plan, err := PlanDefinition(safe, PlanOptions{})
	if err != nil {
		t.Fatalf("plan safe env: %v", err)
	}
	if got := plan.Steps[0].Env["NIVORA_TOKEN"]; got != "[REDACTED]" {
		t.Fatalf("secret-like env not redacted: %q", got)
	}
}

func TestWorkflowParseOnMap(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: triggers
on:
  manual: {}
  pull_request:
    branches: [main]
jobs:
  test:
    steps:
      - run: echo ok
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	plan, err := PlanDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("plan workflow: %v", err)
	}
	if strings.Join(plan.Triggers, ",") != "manual,pull_request" {
		t.Fatalf("triggers = %#v", plan.Triggers)
	}
}

func TestWorkflowPlanIncludesPlanOnlyIntents(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: intent-ci
on:
  - manual
  - schedule
jobs:
  test:
    runsOn: [self-hosted]
    steps:
      - run: go test ./...
artifacts:
  - name: binary
    path: dist/app
    type: binary
    retentionDays: 7
cache:
  - key: gomod
    path: [go.sum, go.mod]
    restoreKeys: [gomod-main]
    scope: project
security:
  scanners: [noop]
  required: true
release:
  name: demo
  environment: staging
  artifacts: [binary]
  requireDigest: true
deployment:
  targetType: kubernetes-yaml
  targetName: staging-yaml
  environment: staging
  apply: true
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	plan, err := PlanDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanDefinition: %v", err)
	}
	if len(plan.ArtifactOutputs) != 1 || plan.ArtifactOutputs[0].RetentionDays != 7 {
		t.Fatalf("artifact outputs = %#v", plan.ArtifactOutputs)
	}
	if len(plan.CacheHints) != 1 || len(plan.CacheHints[0].RestoreKeys) != 1 {
		t.Fatalf("cache hints = %#v", plan.CacheHints)
	}
	if plan.SecurityIntent == nil || !plan.SecurityIntent.Required || !plan.SecurityIntent.PlanOnly {
		t.Fatalf("security intent = %#v", plan.SecurityIntent)
	}
	if plan.ReleaseIntent == nil || !plan.ReleaseIntent.RequireDigest || !plan.ReleaseIntent.PlanOnly {
		t.Fatalf("release intent = %#v", plan.ReleaseIntent)
	}
	if plan.DeploymentIntent == nil || !plan.DeploymentIntent.ApplyRequested || !plan.DeploymentIntent.PlanOnly {
		t.Fatalf("deployment intent = %#v", plan.DeploymentIntent)
	}
	joinedWarnings := strings.Join(append(plan.Warnings, plan.SecurityWarnings...), "\n")
	for _, want := range []string{"schedule trigger is a placeholder", "security intent is plan-only", "release intent is plan-only", "apply=true was requested"} {
		if !strings.Contains(joinedWarnings, want) {
			t.Fatalf("missing warning %q in %q", want, joinedWarnings)
		}
	}
}

func TestWorkflowPlanIncludesPermissionRequestsAndWarnings(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: permissions
on: manual
permissions:
  contents: read
  deployments: write
  id-token: write
jobs:
  test:
    runsOn: [self-hosted]
    steps:
      - run: echo ok
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	plan, err := PlanDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanDefinition: %v", err)
	}
	if len(plan.PermissionRequests) != 3 {
		t.Fatalf("permission requests = %#v", plan.PermissionRequests)
	}
	if plan.PermissionRequests[0].Scope != "contents" || plan.PermissionRequests[0].Access != "read" || !plan.PermissionRequests[0].PlanOnly {
		t.Fatalf("first permission request = %#v", plan.PermissionRequests[0])
	}
	joined := strings.Join(plan.SecurityWarnings, "\n")
	for _, want := range []string{"deployments requests write access", "id-token permission is foundation-only"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing permission warning %q in %q", want, joined)
		}
	}
}

func TestWorkflowPlanPreservesJobLabelsForPipelineConversion(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: labeled-runner
on: manual
jobs:
  test:
    runsOn: [self-hosted, shell]
    labels:
      runtime: workflow
      tier: secure
    steps:
      - run: go test ./...
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	plan, err := PlanDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("PlanDefinition: %v", err)
	}
	if got := plan.Jobs[0].Labels["runtime"]; got != "workflow" {
		t.Fatalf("planned job labels = %#v", plan.Jobs[0].Labels)
	}
	direct, err := ToPipelineDefinition(def, PlanOptions{})
	if err != nil {
		t.Fatalf("direct conversion: %v", err)
	}
	if got := direct.Definition.Spec.Stages[0].Jobs[0].Labels["tier"]; got != "secure" {
		t.Fatalf("direct converted labels = %#v", direct.Definition.Spec.Stages[0].Jobs[0].Labels)
	}
	directJob := direct.Definition.Spec.Stages[0].Jobs[0]
	if directJob.Metadata[pipelineusecase.MetadataWorkflowJobID] == "" {
		t.Fatalf("direct converted job metadata = %#v", directJob.Metadata)
	}
	if directJob.Steps[0].Metadata[pipelineusecase.MetadataWorkflowStepID] == "" {
		t.Fatalf("direct converted step metadata = %#v", directJob.Steps[0].Metadata)
	}
	fromPlan, err := ToPipelineDefinitionFromPlan(plan)
	if err != nil {
		t.Fatalf("plan conversion: %v", err)
	}
	if got := fromPlan.Definition.Spec.Stages[0].Jobs[0].Labels["runtime"]; got != "workflow" {
		t.Fatalf("plan converted labels = %#v", fromPlan.Definition.Spec.Stages[0].Jobs[0].Labels)
	}
	planJob := fromPlan.Definition.Spec.Stages[0].Jobs[0]
	if planJob.Metadata[pipelineusecase.MetadataWorkflowJobID] != plan.Jobs[0].ID {
		t.Fatalf("plan converted job metadata = %#v plan job=%#v", planJob.Metadata, plan.Jobs[0])
	}
	if planJob.Steps[0].Metadata[pipelineusecase.MetadataWorkflowStepID] != plan.Steps[0].ID {
		t.Fatalf("plan converted step metadata = %#v plan step=%#v", planJob.Steps[0].Metadata, plan.Steps[0])
	}
}

func TestWorkflowPlanRejectsSecretLikeJobLabels(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: unsafe-label
on: manual
jobs:
  test:
    labels:
      token: runner-a
    steps:
      - run: echo ok
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	_, err = PlanDefinition(def, PlanOptions{})
	if err == nil || !strings.Contains(err.Error(), "label") {
		t.Fatalf("expected secret-like label rejection, got %v", err)
	}
}

func TestWorkflowPlanRejectsSecretLikePermissionScope(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: unsafe-permissions
on: manual
permissions:
  password: read
jobs:
  test:
    steps:
      - run: echo ok
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	_, err = PlanDefinition(def, PlanOptions{})
	if err == nil || !strings.Contains(err.Error(), "permission scope") {
		t.Fatalf("expected secret-like permission scope rejection, got %v", err)
	}
}

func TestWorkflowPlanRejectsSecretLikeIntentValues(t *testing.T) {
	def, err := ParseDefinition([]byte(`
apiVersion: nivora.io/v1alpha1
kind: Workflow
metadata:
  name: unsafe-intent
on: manual
jobs:
  test:
    steps:
      - run: echo ok
deployment:
  token: inline-token-value
`))
	if err != nil {
		t.Fatalf("parse workflow: %v", err)
	}
	_, err = PlanDefinition(def, PlanOptions{})
	if err == nil || !strings.Contains(err.Error(), "deployment.token") {
		t.Fatalf("expected secret-like deployment intent rejection, got %v", err)
	}
}
