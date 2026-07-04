package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sevoniva/nivora/internal/app/runtime"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/infra/config"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

func TestMCPInitializeJSONRPC(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	response := server.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if response.Error != nil {
		t.Fatalf("initialize error = %#v", response.Error)
	}
	body := mustMarshal(t, response.Result)
	if !strings.Contains(body, ProtocolVersion) || !strings.Contains(body, "nivora-mcp") {
		t.Fatalf("initialize result = %s", body)
	}
}

func TestMCPResourceToolAndPromptCatalogs(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	resources, err := server.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	for _, want := range []string{"nivora://capabilities/current", "nivora://system/runtime", "nivora://runtime/recovery", "nivora://events", "nivora://logs", "nivora://catalog/summary", "nivora://pipelines/definitions/{id}", "nivora://deployments/{id}/health", "nivora://artifacts/{id}/releases", "nivora://security/findings", "nivora://policy/results/summary", "nivora://evidence/bundles", "nivora://evidence/bundles/{id}", "nivora://plugins/capabilities"} {
		if !hasResource(resources, want) {
			t.Fatalf("resource %s missing from %#v", want, resources)
		}
	}
	tools, err := server.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	for _, blocked := range []string{"nivora_apply_deployment", "nivora_get_secret", "nivora_rotate_token"} {
		if hasTool(tools, blocked) {
			t.Fatalf("blocked action tool %s was exposed", blocked)
		}
	}
	for _, want := range []string{"nivora_status", "nivora_get_runtime_recovery_status", "nivora_search_events", "nivora_search_logs", "nivora_get_catalog_summary", "nivora_list_pipeline_definitions", "nivora_get_pipeline_definition", "nivora_get_deployment_health", "nivora_list_artifacts", "nivora_get_artifact_releases", "nivora_list_security_findings", "nivora_get_policy_result_summary", "nivora_list_evidence_bundles", "nivora_get_evidence_bundle", "nivora_plan_deployment_local"} {
		if !hasTool(tools, want) {
			t.Fatalf("tool %s missing from %#v", want, tools)
		}
	}
	prompts, err := server.ListPrompts(context.Background())
	if err != nil {
		t.Fatalf("ListPrompts: %v", err)
	}
	for _, want := range []string{"diagnose_pipeline_run", "release_readiness_review", "mcp_safe_operation_check"} {
		if !hasPrompt(prompts, want) {
			t.Fatalf("prompt %s missing from %#v", want, prompts)
		}
	}
}

func TestMCPBlockedActionToolDenied(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleAdmin, "mcp-local")
	for name := range blockedActionTools {
		result, err := server.CallTool(context.Background(), name, map[string]any{"id": "dep-1", "authorization": "Bearer should-not-leak"})
		if err != nil {
			t.Fatalf("%s: CallTool returned transport error: %v", name, err)
		}
		if !result.IsError {
			t.Fatalf("%s: expected blocked action error, got %#v", name, result)
		}
		body := result.Content[0].Text
		if !strings.Contains(body, "mcp_action_not_allowed") || !strings.Contains(body, "requiredFutureGate") {
			t.Fatalf("%s: blocked action body = %s", name, body)
		}
		if strings.Contains(body, "should-not-leak") {
			t.Fatalf("%s: blocked action leaked arguments: %s", name, body)
		}
	}
}

func TestMCPRunnerTokenCannotUseControlPlane(t *testing.T) {
	server := newTestMCPServerWithSubject(t, domainauth.Subject{ID: "runner:runner-a", Username: "runner-a", AuthMode: "runner_token"})
	if _, err := server.ListResources(context.Background()); err == nil || !strings.Contains(err.Error(), "runner tokens cannot use MCP") {
		t.Fatalf("expected runner token denial, got %v", err)
	}
	result, err := server.CallTool(context.Background(), "nivora_status", nil)
	if err != nil {
		t.Fatalf("CallTool transport error = %v", err)
	}
	if !result.IsError || !strings.Contains(result.Content[0].Text, "mcp_runner_token_denied") {
		t.Fatalf("runner token tool result = %#v", result)
	}
}

func TestMCPRBACReadAuditAndPlanBoundaries(t *testing.T) {
	ctx := context.Background()
	viewer := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	if _, err := viewer.ReadResource(ctx, "nivora://system/runtime"); err != nil {
		t.Fatalf("viewer read resource: %v", err)
	}
	planResult, err := viewer.CallTool(ctx, "nivora_plan_deployment_local", map[string]any{"content": deploymentDefinitionYAML()})
	if err != nil {
		t.Fatalf("viewer plan transport error: %v", err)
	}
	if !planResult.IsError || !strings.Contains(planResult.Content[0].Text, "mcp_forbidden") {
		t.Fatalf("viewer plan result = %#v", planResult)
	}
	if _, err := viewer.ReadResource(ctx, "nivora://audit/search"); err == nil {
		t.Fatalf("viewer read audit unexpectedly allowed")
	}

	auditor := newTestMCPServer(t, domainauth.RoleAuditor, "mcp-local")
	if _, err := auditor.ReadResource(ctx, "nivora://audit/search"); err != nil {
		t.Fatalf("auditor read audit: %v", err)
	}
	planResult, err = auditor.CallTool(ctx, "nivora_plan_deployment_local", map[string]any{"content": deploymentDefinitionYAML()})
	if err != nil {
		t.Fatalf("auditor plan transport error: %v", err)
	}
	if !planResult.IsError {
		t.Fatalf("auditor plan unexpectedly allowed: %#v", planResult)
	}

	developer := newTestMCPServer(t, domainauth.RoleDeveloper, "token")
	planResult, err = developer.CallTool(ctx, "nivora_plan_deployment_local", map[string]any{"content": deploymentDefinitionYAML()})
	if err != nil {
		t.Fatalf("developer plan transport error: %v", err)
	}
	if planResult.IsError || !strings.Contains(planResult.Content[0].Text, `"mutated": false`) {
		t.Fatalf("developer plan result = %#v", planResult)
	}
}

func TestMCPArtifactInventoryReadOnly(t *testing.T) {
	ctx := context.Background()
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	artifactID, releaseID := createMCPArtifactFixture(t, server)

	resource, err := server.ReadResource(ctx, "nivora://artifacts")
	if err != nil {
		t.Fatalf("read artifact inventory resource: %v", err)
	}
	if !strings.Contains(resource.Text, artifactID) || !strings.Contains(resource.Text, "registry.example.com/team/demo") {
		t.Fatalf("artifact inventory resource body = %s", resource.Text)
	}

	resource, err = server.ReadResource(ctx, "nivora://artifacts/"+artifactID+"/releases")
	if err != nil {
		t.Fatalf("read artifact releases resource: %v", err)
	}
	if !strings.Contains(resource.Text, releaseID) || !strings.Contains(resource.Text, artifactID) {
		t.Fatalf("artifact releases resource body = %s", resource.Text)
	}

	listResult, err := server.CallTool(ctx, "nivora_list_artifacts", map[string]any{"registry": "registry.example.com"})
	if err != nil {
		t.Fatalf("list artifact tool transport error: %v", err)
	}
	if listResult.IsError || !strings.Contains(listResult.Content[0].Text, artifactID) || !strings.Contains(listResult.Content[0].Text, `"mutated": false`) {
		t.Fatalf("list artifact tool result = %#v", listResult)
	}

	getResult, err := server.CallTool(ctx, "nivora_get_artifact", map[string]any{"id": artifactID})
	if err != nil {
		t.Fatalf("get artifact tool transport error: %v", err)
	}
	if getResult.IsError || !strings.Contains(getResult.Content[0].Text, artifactID) || !strings.Contains(getResult.Content[0].Text, `"mutated": false`) {
		t.Fatalf("get artifact tool result = %#v", getResult)
	}

	releasesResult, err := server.CallTool(ctx, "nivora_get_artifact_releases", map[string]any{"id": artifactID, "authorization": "Bearer should-not-leak"})
	if err != nil {
		t.Fatalf("get artifact releases tool transport error: %v", err)
	}
	body := releasesResult.Content[0].Text
	if releasesResult.IsError || !strings.Contains(body, releaseID) || !strings.Contains(body, `"mutated": false`) {
		t.Fatalf("get artifact releases tool result = %#v", releasesResult)
	}
	for _, forbidden := range []string{"should-not-leak", "tokenHash", "BEGIN PRIVATE KEY", "password"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("artifact MCP output leaked sensitive value %q: %s", forbidden, body)
		}
	}
}

func TestMCPCatalogAndPipelineDefinitionsReadOnly(t *testing.T) {
	ctx := context.Background()
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	projectID, pipelineID := createMCPCatalogAndPipelineFixture(t, server)

	resource, err := server.ReadResource(ctx, "nivora://catalog/summary")
	if err != nil {
		t.Fatalf("read catalog summary: %v", err)
	}
	if !strings.Contains(resource.Text, projectID) || !strings.Contains(resource.Text, "demo-api") || !strings.Contains(resource.Text, `"mutated": false`) {
		t.Fatalf("catalog summary body = %s", resource.Text)
	}

	resource, err = server.ReadResource(ctx, "nivora://pipelines/definitions")
	if err != nil {
		t.Fatalf("read pipeline definition catalog: %v", err)
	}
	if !strings.Contains(resource.Text, pipelineID) || !strings.Contains(resource.Text, "mcp-demo-pipeline") {
		t.Fatalf("pipeline definition catalog body = %s", resource.Text)
	}

	summaryResult, err := server.CallTool(ctx, "nivora_get_catalog_summary", map[string]any{"projectId": projectID})
	if err != nil {
		t.Fatalf("catalog summary tool transport error: %v", err)
	}
	if summaryResult.IsError || !strings.Contains(summaryResult.Content[0].Text, projectID) || !strings.Contains(summaryResult.Content[0].Text, `"mutated": false`) {
		t.Fatalf("catalog summary tool result = %#v", summaryResult)
	}

	listResult, err := server.CallTool(ctx, "nivora_list_pipeline_definitions", map[string]any{"projectId": projectID})
	if err != nil {
		t.Fatalf("list pipeline definitions tool transport error: %v", err)
	}
	if listResult.IsError || !strings.Contains(listResult.Content[0].Text, pipelineID) || !strings.Contains(listResult.Content[0].Text, `"mutated": false`) {
		t.Fatalf("list pipeline definitions result = %#v", listResult)
	}

	getResult, err := server.CallTool(ctx, "nivora_get_pipeline_definition", map[string]any{"id": pipelineID, "token": "should-not-leak"})
	if err != nil {
		t.Fatalf("get pipeline definition tool transport error: %v", err)
	}
	body := getResult.Content[0].Text
	if getResult.IsError || !strings.Contains(body, pipelineID) || !strings.Contains(body, `"mutated": false`) {
		t.Fatalf("get pipeline definition result = %#v", getResult)
	}
	for _, forbidden := range []string{"should-not-leak", "tokenHash", "BEGIN PRIVATE KEY"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("pipeline definition MCP output leaked sensitive value %q: %s", forbidden, body)
		}
	}
}

func TestMCPSecurityPolicyAndRuntimeReadOnly(t *testing.T) {
	ctx := context.Background()
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	record, err := server.services.Security.Scan(ctx, securityusecase.ScanInput{
		SubjectType: domainsecurity.SubjectManifest,
		SubjectID:   "mcp-manifest-risk",
		Content:     "containers:\n- name: api\n  image: registry.example.invalid/team/api:latest\n  imagePullPolicy: Always\n  securityContext:\n    privileged: true\n",
		ActorID:     "mcp-fixture",
	})
	if err != nil {
		t.Fatalf("create security scan fixture: %v", err)
	}

	summary, err := server.ReadResource(ctx, "nivora://security/summary")
	if err != nil {
		t.Fatalf("read security summary: %v", err)
	}
	if !strings.Contains(summary.Text, record.Scan.ID) || !strings.Contains(summary.Text, `"findingCount": 2`) || !strings.Contains(summary.Text, `"warn": 1`) {
		t.Fatalf("security summary body = %s", summary.Text)
	}

	findingsResource, err := server.ReadResource(ctx, "nivora://security/findings")
	if err != nil {
		t.Fatalf("read security findings: %v", err)
	}
	if !strings.Contains(findingsResource.Text, "Privileged container requested") || !strings.Contains(findingsResource.Text, `"mutated": false`) {
		t.Fatalf("security findings body = %s", findingsResource.Text)
	}

	findingsResult, err := server.CallTool(ctx, "nivora_list_security_findings", map[string]any{
		"severity":      "High",
		"authorization": "Bearer should-not-leak",
	})
	if err != nil {
		t.Fatalf("security findings tool transport error: %v", err)
	}
	findingsBody := findingsResult.Content[0].Text
	if findingsResult.IsError || !strings.Contains(findingsBody, "Privileged container requested") || strings.Contains(findingsBody, "latest image with Always pull policy") || !strings.Contains(findingsBody, `"mutated": false`) {
		t.Fatalf("security findings tool result = %#v", findingsResult)
	}
	if strings.Contains(findingsBody, "should-not-leak") || strings.Contains(findingsBody, "Authorization: Bearer") {
		t.Fatalf("security findings leaked sensitive value: %s", findingsBody)
	}

	policyResource, err := server.ReadResource(ctx, "nivora://policy/results/summary")
	if err != nil {
		t.Fatalf("read policy summary: %v", err)
	}
	if !strings.Contains(policyResource.Text, `"warn": 1`) || !strings.Contains(policyResource.Text, record.Scan.ID) || !strings.Contains(policyResource.Text, `"mutated": false`) {
		t.Fatalf("policy summary body = %s", policyResource.Text)
	}

	policyResult, err := server.CallTool(ctx, "nivora_get_policy_result_summary", map[string]any{"subjectType": "manifest", "subjectId": "mcp-manifest-risk"})
	if err != nil {
		t.Fatalf("policy summary tool transport error: %v", err)
	}
	if policyResult.IsError || !strings.Contains(policyResult.Content[0].Text, `"warn": 1`) || !strings.Contains(policyResult.Content[0].Text, `"mutated": false`) {
		t.Fatalf("policy summary tool result = %#v", policyResult)
	}

	runtimeResult, err := server.CallTool(ctx, "nivora_get_runtime_recovery_status", map[string]any{"token": "should-not-leak"})
	if err != nil {
		t.Fatalf("runtime recovery tool transport error: %v", err)
	}
	runtimeBody := runtimeResult.Content[0].Text
	if runtimeResult.IsError || !strings.Contains(runtimeBody, `"pipeline"`) || !strings.Contains(runtimeBody, `"deployment"`) || !strings.Contains(runtimeBody, `"release"`) || !strings.Contains(runtimeBody, `"mutated": false`) {
		t.Fatalf("runtime recovery tool result = %#v", runtimeResult)
	}
	if strings.Contains(runtimeBody, "should-not-leak") || strings.Contains(runtimeBody, "tokenHash") {
		t.Fatalf("runtime recovery leaked sensitive value: %s", runtimeBody)
	}
}

func TestMCPAggregateEventsAndLogsReadOnly(t *testing.T) {
	ctx := context.Background()
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	pipelineRunID, deploymentRunID := createMCPObservabilityFixture(t, server)

	eventsResource, err := server.ReadResource(ctx, "nivora://events")
	if err != nil {
		t.Fatalf("read events resource: %v", err)
	}
	if !strings.Contains(eventsResource.Text, pipelineRunID) || !strings.Contains(eventsResource.Text, deploymentRunID) || !strings.Contains(eventsResource.Text, `"mutated": false`) {
		t.Fatalf("events resource body = %s", eventsResource.Text)
	}

	logsResource, err := server.ReadResource(ctx, "nivora://logs")
	if err != nil {
		t.Fatalf("read logs resource: %v", err)
	}
	if !strings.Contains(logsResource.Text, "mcp-observe-log") || !strings.Contains(logsResource.Text, "dry-run validation completed") || !strings.Contains(logsResource.Text, `"mutated": false`) {
		t.Fatalf("logs resource body = %s", logsResource.Text)
	}

	eventResult, err := server.CallTool(ctx, "nivora_search_events", map[string]any{
		"pipelineRunId": pipelineRunID,
		"type":          "pipeline.run.completed",
		"authorization": "Bearer should-not-leak",
	})
	if err != nil {
		t.Fatalf("event search tool transport error: %v", err)
	}
	eventBody := eventResult.Content[0].Text
	if eventResult.IsError || !strings.Contains(eventBody, pipelineRunID) || !strings.Contains(eventBody, `"mutated": false`) {
		t.Fatalf("event search result = %#v", eventResult)
	}

	logResult, err := server.CallTool(ctx, "nivora_search_logs", map[string]any{
		"deploymentRunId": deploymentRunID,
		"contains":        "dry-run validation",
		"token":           "should-not-leak",
	})
	if err != nil {
		t.Fatalf("log search tool transport error: %v", err)
	}
	logBody := logResult.Content[0].Text
	if logResult.IsError || !strings.Contains(logBody, deploymentRunID) || !strings.Contains(logBody, "dry-run validation completed") || !strings.Contains(logBody, `"mutated": false`) {
		t.Fatalf("log search result = %#v", logResult)
	}

	for _, body := range []string{eventBody, logBody} {
		for _, forbidden := range []string{"should-not-leak", "tokenHash", "BEGIN PRIVATE KEY", "Authorization: Bearer"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("aggregate observability MCP output leaked sensitive value %q: %s", forbidden, body)
			}
		}
	}
}

func TestMCPTenantScopeFiltersDeploymentReadsAndAggregates(t *testing.T) {
	ctx := context.Background()
	store := deploymentusecase.NewMemoryStore()
	deployments := runtime.NewDeploymentServiceWithStore(store)

	projectA := newTestMCPServerWithServices(t, domainauth.Subject{
		ID:        "sa-project-a",
		Username:  "sa-project-a",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-a",
	}, deployments)
	projectB := newTestMCPServerWithServices(t, domainauth.Subject{
		ID:        "sa-project-b",
		Username:  "sa-project-b",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-b",
	}, deployments)

	runA := createScopedMCPDeploymentRun(t, store, deployments, "project-a")
	runB := createScopedMCPDeploymentRun(t, store, deployments, "project-b")

	if _, err := projectA.ReadResource(ctx, "nivora://deployments/"+runA); err != nil {
		t.Fatalf("project A read own deployment: %v", err)
	}
	if _, err := projectA.ReadResource(ctx, "nivora://deployments/"+runA+"/health"); err != nil {
		t.Fatalf("project A read own deployment health: %v", err)
	}
	_, err := projectA.ReadResource(ctx, "nivora://deployments/"+runB)
	var op OperationError
	if !errors.As(err, &op) || op.Code != "mcp_scope_denied" {
		t.Fatalf("expected scope denial for cross-project deployment, got %T %v", err, err)
	}

	result, err := projectA.CallTool(ctx, "nivora_search_events", map[string]any{"deploymentRunId": runB})
	if err != nil {
		t.Fatalf("cross-project event search transport error: %v", err)
	}
	if !resultCountIsZero(t, result) {
		t.Fatalf("cross-project event search returned data: %#v", result)
	}
	result, err = projectA.CallTool(ctx, "nivora_search_logs", map[string]any{"deploymentRunId": runB})
	if err != nil {
		t.Fatalf("cross-project log search transport error: %v", err)
	}
	if !resultCountIsZero(t, result) {
		t.Fatalf("cross-project log search returned data: %#v", result)
	}

	if _, err := projectB.ReadResource(ctx, "nivora://deployments/"+runB+"/diff"); err != nil {
		t.Fatalf("project B read own deployment diff: %v", err)
	}
}

func TestMCPTenantScopeFiltersPipelineReadsAndAggregates(t *testing.T) {
	ctx := context.Background()
	pipelines := runtime.NewPipelineService()
	projectA := newTestMCPServerWithSubject(t, domainauth.Subject{
		ID:        "sa-pipeline-project-a",
		Username:  "sa-pipeline-project-a",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-a",
	})
	projectA.services.Pipelines = pipelines
	projectB := newTestMCPServerWithSubject(t, domainauth.Subject{
		ID:        "sa-pipeline-project-b",
		Username:  "sa-pipeline-project-b",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-b",
	})
	projectB.services.Pipelines = pipelines

	runA := createScopedMCPPipelineRun(t, pipelines, "project-a")
	runB := createScopedMCPPipelineRun(t, pipelines, "project-b")

	if _, err := projectA.ReadResource(ctx, "nivora://pipelines/runs/"+runA); err != nil {
		t.Fatalf("project A read own pipeline run: %v", err)
	}
	if _, err := projectA.ReadResource(ctx, "nivora://pipelines/runs/"+runA+"/logs"); err != nil {
		t.Fatalf("project A read own pipeline logs: %v", err)
	}
	_, err := projectA.ReadResource(ctx, "nivora://pipelines/runs/"+runB)
	var op OperationError
	if !errors.As(err, &op) || op.Code != "mcp_scope_denied" {
		t.Fatalf("expected scope denial for cross-project pipeline run, got %T %v", err, err)
	}

	result, err := projectA.CallTool(ctx, "nivora_search_events", map[string]any{"pipelineRunId": runB})
	if err != nil {
		t.Fatalf("cross-project pipeline event search transport error: %v", err)
	}
	if !resultCountIsZero(t, result) {
		t.Fatalf("cross-project pipeline event search returned data: %#v", result)
	}
	result, err = projectA.CallTool(ctx, "nivora_search_logs", map[string]any{"pipelineRunId": runB})
	if err != nil {
		t.Fatalf("cross-project pipeline log search transport error: %v", err)
	}
	if !resultCountIsZero(t, result) {
		t.Fatalf("cross-project pipeline log search returned data: %#v", result)
	}

	if _, err := projectB.ReadResource(ctx, "nivora://pipelines/runs/"+runB+"/timeline"); err != nil {
		t.Fatalf("project B read own pipeline timeline: %v", err)
	}
}

func TestMCPTenantScopeFiltersReleaseExecutionReadsAndAggregates(t *testing.T) {
	ctx := context.Background()
	releaseStore := releaseusecase.NewMemoryStore()
	artifacts := runtime.NewArtifactService()
	deployments := runtime.NewDeploymentService()
	releases := runtime.NewReleaseOrchestrationServiceWithStore(releaseStore, artifacts, deployments)

	projectA := newTestMCPServerWithSubject(t, domainauth.Subject{
		ID:        "sa-release-project-a",
		Username:  "sa-release-project-a",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-a",
	})
	projectA.services.Releases = releases
	projectB := newTestMCPServerWithSubject(t, domainauth.Subject{
		ID:        "sa-release-project-b",
		Username:  "sa-release-project-b",
		Roles:     []string{domainauth.RoleDeveloper},
		AuthMode:  "service_account",
		ScopeType: "project",
		ScopeID:   "project-b",
	})
	projectB.services.Releases = releases

	execA := createScopedMCPReleaseExecution(t, releaseStore, releases, "project-a")
	execB := createScopedMCPReleaseExecution(t, releaseStore, releases, "project-b")

	if _, err := projectA.ReadResource(ctx, "nivora://releases/executions/"+execA); err != nil {
		t.Fatalf("project A read own release execution: %v", err)
	}
	if _, err := projectA.ReadResource(ctx, "nivora://releases/executions/"+execA+"/timeline"); err != nil {
		t.Fatalf("project A read own release execution timeline: %v", err)
	}
	_, err := projectA.ReadResource(ctx, "nivora://releases/executions/"+execB)
	var op OperationError
	if !errors.As(err, &op) || op.Code != "mcp_scope_denied" {
		t.Fatalf("expected scope denial for cross-project release execution, got %T %v", err, err)
	}

	result, err := projectA.CallTool(ctx, "nivora_search_events", map[string]any{"subject": execB})
	if err != nil {
		t.Fatalf("cross-project release event search transport error: %v", err)
	}
	if !resultCountIsZero(t, result) {
		t.Fatalf("cross-project release event search returned data: %#v", result)
	}

	result, err = projectB.CallTool(ctx, "nivora_get_release_execution", map[string]any{"id": execB})
	if err != nil {
		t.Fatalf("project B read own release execution through tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("project B release execution tool returned error: %#v", result)
	}
}

func TestMCPEvidenceBundleRequiresAuditRead(t *testing.T) {
	ctx := context.Background()
	viewer := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	if _, err := viewer.ReadResource(ctx, "nivora://evidence/bundles"); err == nil {
		t.Fatal("viewer read evidence list unexpectedly allowed")
	}
	if _, err := viewer.ReadResource(ctx, "nivora://evidence/bundles/evb-missing"); err == nil {
		t.Fatal("viewer read evidence unexpectedly allowed")
	}

	auditor := newTestMCPServer(t, domainauth.RoleAuditor, "token")
	bundle, err := auditor.services.Compliance.EvidenceBundle(ctx, complianceusecase.EvidenceInput{SubjectType: "generic", SubjectID: "mcp-evidence"})
	if err != nil {
		t.Fatalf("generate evidence bundle: %v", err)
	}
	listResource, err := auditor.ReadResource(ctx, "nivora://evidence/bundles")
	if err != nil {
		t.Fatalf("auditor read evidence list resource: %v", err)
	}
	if !strings.Contains(listResource.Text, bundle.ID) || !strings.Contains(listResource.Text, `"mutated": false`) {
		t.Fatalf("evidence list resource body = %s", listResource.Text)
	}
	resource, err := auditor.ReadResource(ctx, "nivora://evidence/bundles/"+bundle.ID)
	if err != nil {
		t.Fatalf("auditor read evidence resource: %v", err)
	}
	if !strings.Contains(resource.Text, bundle.ID) {
		t.Fatalf("evidence resource body = %s", resource.Text)
	}
	listResult, err := auditor.CallTool(ctx, "nivora_list_evidence_bundles", map[string]any{"subjectType": "generic", "subjectId": "mcp-evidence"})
	if err != nil {
		t.Fatalf("evidence list tool transport error: %v", err)
	}
	if listResult.IsError || !strings.Contains(listResult.Content[0].Text, bundle.ID) || !strings.Contains(listResult.Content[0].Text, `"mutated": false`) {
		t.Fatalf("evidence list tool result = %#v", listResult)
	}
	result, err := auditor.CallTool(ctx, "nivora_get_evidence_bundle", map[string]any{"id": bundle.ID, "authorization": "Bearer should-not-leak"})
	if err != nil {
		t.Fatalf("evidence tool transport error: %v", err)
	}
	if result.IsError {
		t.Fatalf("evidence tool result = %#v", result)
	}
	body := result.Content[0].Text
	for _, forbidden := range []string{"should-not-leak", "tokenHash", "BEGIN PRIVATE KEY"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("evidence MCP output leaked sensitive value %q: %s", forbidden, body)
		}
	}
}

func TestMCPPlanDeploymentLocalDoesNotMutateDeploymentRuns(t *testing.T) {
	ctx := context.Background()
	server, deployments := newTestMCPServerAndDeploymentService(t, domainauth.RoleDeveloper, "token")
	before, err := deployments.List(ctx)
	if err != nil {
		t.Fatalf("list before: %v", err)
	}
	result, err := server.CallTool(ctx, "nivora_plan_deployment_local", map[string]any{"content": deploymentDefinitionYAML()})
	if err != nil {
		t.Fatalf("plan tool transport error: %v", err)
	}
	if result.IsError {
		t.Fatalf("plan tool error = %#v", result)
	}
	after, err := deployments.List(ctx)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("plan-only tool mutated deployments: before=%d after=%d", len(before), len(after))
	}
}

func TestMCPRedactsSecretLikeData(t *testing.T) {
	body := mustJSON(map[string]any{
		"token":         "raw-token-value",
		"tokenHash":     "hashed-token-value",
		"password":      "password-value",
		"secret":        "secret-value",
		"authorization": "Bearer raw-token-value",
		"access_key":    "access-key-value",
		"bearer":        "bearer-value",
		"client_secret": "client-secret-value",
		"refresh_token": "refresh-token-value",
		"id_token":      "id-token-value",
		"session":       "session-value",
		"kubeconfig":    "apiVersion: v1\nclusters: []",
		"nested": map[string]any{
			"private_key": "-----BEGIN PRIVATE KEY-----\nvalue\n-----END PRIVATE KEY-----",
			"message":     "Authorization: Bearer raw-token-value",
		},
	})
	for _, forbidden := range []string{
		"raw-token-value",
		"hashed-token-value",
		"password-value",
		"secret-value",
		"access-key-value",
		"bearer-value",
		"client-secret-value",
		"refresh-token-value",
		"id-token-value",
		"session-value",
		"BEGIN PRIVATE KEY",
		"apiVersion: v1",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("secret-like value leaked in %s", body)
		}
	}
	if !strings.Contains(body, "[REDACTED]") {
		t.Fatalf("expected redaction marker in %s", body)
	}
}

func TestMCPResponseCapReturnsStructuredTruncation(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	server.services.Config.MCP.MaxResponseBytes = 32

	resource, err := server.ReadResource(context.Background(), "nivora://capabilities/current")
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if !strings.Contains(resource.Text, `"truncated": true`) || !strings.Contains(resource.Text, "max_response_bytes") {
		t.Fatalf("expected structured truncation response, got %s", resource.Text)
	}

	result, err := server.CallTool(context.Background(), "nivora_get_capability_status", nil)
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError || !strings.Contains(result.Content[0].Text, `"truncated": true`) {
		t.Fatalf("expected capped tool response, got %#v", result)
	}
}

func TestMCPJSONRPCResponseCapAppliesToCatalogResponses(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	server.services.Config.MCP.MaxResponseBytes = 320

	response := server.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","id":7,"method":"resources/list","params":{}}`))
	if response.Error == nil {
		t.Fatalf("expected capped JSON-RPC error, got %#v", response)
	}
	if response.Error.Code != rpcInternalError {
		t.Fatalf("unexpected error code = %d", response.Error.Code)
	}
	body := mustMarshal(t, response)
	if len([]byte(body)) > server.services.Config.MCP.MaxResponseBytes {
		t.Fatalf("capped response length = %d, want <= %d: %s", len([]byte(body)), server.services.Config.MCP.MaxResponseBytes, body)
	}
	if !strings.Contains(body, "mcp_response_too_large") || strings.Contains(body, "nivora://capabilities/current") {
		t.Fatalf("unexpected capped response body: %s", body)
	}
}

func TestMCPServeStdioAppliesTransportResponseCap(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	server.services.Config.MCP.MaxResponseBytes = 320
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":"req-1","method":"tools/list","params":{}}` + "\n")
	var output bytes.Buffer

	if err := server.ServeStdio(context.Background(), input, &output); err != nil {
		t.Fatalf("ServeStdio: %v", err)
	}
	body := output.String()
	if len([]byte(strings.TrimSpace(body))) > server.services.Config.MCP.MaxResponseBytes {
		t.Fatalf("stdio response length = %d, want <= %d: %s", len([]byte(strings.TrimSpace(body))), server.services.Config.MCP.MaxResponseBytes, body)
	}
	if !strings.Contains(body, "mcp_response_too_large") || strings.Contains(body, "nivora_status") {
		t.Fatalf("unexpected stdio capped response: %s", body)
	}
}

func TestMCPRequestContextUsesConfiguredTimeout(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	server.services.Config.MCP.RequestTimeout = "25ms"
	ctx, cancel := server.requestContext(context.Background())
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected request context deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > time.Second {
		t.Fatalf("unexpected deadline remaining = %s", remaining)
	}
}

func TestMCPJSONRPCRateLimit(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	server.services.Config.MCP.MaxRequestsPerMinute = 1

	first := server.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if first.Error != nil {
		t.Fatalf("first request error = %#v", first.Error)
	}
	second := server.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","id":2,"method":"initialize","params":{}}`))
	if second.Error == nil || second.Error.Code != rpcInternalError {
		t.Fatalf("expected rate-limit error, got %#v", second)
	}
	if second.ID != float64(2) {
		t.Fatalf("rate-limit response id = %#v, want 2", second.ID)
	}
	body := mustMarshal(t, second)
	if !strings.Contains(body, "mcp_rate_limited") {
		t.Fatalf("rate-limit response missing code: %s", body)
	}
}

func TestMCPJSONRPCRequestBodyLimit(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	server.services.Config.MCP.MaxRequestBytes = 64
	body := `{"jsonrpc":"2.0","id":3,"method":"initialize","params":{"padding":"` + strings.Repeat("x", 80) + `"}}`

	response := server.HandleJSONRPC(context.Background(), []byte(body))
	if response.Error == nil || response.Error.Code != rpcInternalError {
		t.Fatalf("expected request-too-large error, got %#v", response)
	}
	got := mustMarshal(t, response)
	if !strings.Contains(got, "mcp_request_too_large") || strings.Contains(got, ProtocolVersion) {
		t.Fatalf("unexpected request limit response: %s", got)
	}
}

func TestMCPMissingDeploymentHealthReturnsStructuredError(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	_, err := server.ReadResource(context.Background(), "nivora://deployments/missing/health")
	if err == nil {
		t.Fatalf("expected missing deployment error")
	}
}

func TestMCPJSONRPCErrorsAreStructured(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	bad := server.HandleJSONRPC(context.Background(), []byte(`{`))
	if bad.Error == nil || bad.Error.Code != rpcParseError {
		t.Fatalf("bad JSON response = %#v", bad)
	}
	unknown := server.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","id":2,"method":"not/a-method"}`))
	if unknown.Error == nil || unknown.Error.Code != rpcMethodNotFound {
		t.Fatalf("unknown method response = %#v", unknown)
	}
	missingArg := server.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"nivora_get_pipeline_run","arguments":{}}}`))
	if missingArg.Result == nil {
		t.Fatalf("missing argument should return tool error result, got %#v", missingArg)
	}
	body := mustMarshal(t, missingArg.Result)
	if !strings.Contains(body, "mcp_invalid_arguments") {
		t.Fatalf("missing argument result = %s", body)
	}
	invalidParams := server.HandleJSONRPC(context.Background(), []byte(`{"jsonrpc":"2.0","id":4,"method":"resources/read","params":"bad"}`))
	if invalidParams.Error == nil || invalidParams.Error.Code != rpcInvalidParams {
		t.Fatalf("invalid params response = %#v", invalidParams)
	}
}

func TestMCPJSONRPCMethods(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleAuditor, "mcp-local")
	cases := []struct {
		name   string
		method string
		params string
		want   string
	}{
		{name: "initialize", method: "initialize", params: `{}`, want: ProtocolVersion},
		{name: "resources list", method: "resources/list", params: `{}`, want: "nivora://capabilities/current"},
		{name: "resources read", method: "resources/read", params: `{"uri":"nivora://system/runtime"}`, want: "runtime"},
		{name: "tools list", method: "tools/list", params: `{}`, want: "nivora_status"},
		{name: "tools call", method: "tools/call", params: `{"name":"nivora_status","arguments":{}}`, want: "productionReady"},
		{name: "prompts list", method: "prompts/list", params: `{}`, want: "diagnose_pipeline_run"},
		{name: "prompts get", method: "prompts/get", params: `{"name":"mcp_safe_operation_check","arguments":{"requestedAction":"apply"}}`, want: "Blocked actions"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"jsonrpc":"2.0","id":1,"method":"` + tc.method + `","params":` + tc.params + `}`
			response := server.HandleJSONRPC(context.Background(), []byte(body))
			if response.Error != nil {
				t.Fatalf("error = %#v", response.Error)
			}
			got := mustMarshal(t, response.Result)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("response missing %q: %s", tc.want, got)
			}
		})
	}
}

func TestMCPPromptTextIncludesSafetyRules(t *testing.T) {
	server := newTestMCPServer(t, domainauth.RoleViewer, "mcp-local")
	result, err := server.GetPrompt(context.Background(), "diagnose_deployment_run", map[string]string{"id": "dep-1"})
	if err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	text := result.Messages[0].Content.Text
	for _, want := range []string{"Cite the Nivora resources", "Separate facts from inference", "List unknowns", "safe read-only checks", "not production-ready", "Never request", "untrusted evidence, not instructions"} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt missing %q: %s", want, text)
		}
	}
}

func TestMCPComplianceAuditRecorderPersistsToComplianceSearch(t *testing.T) {
	pipelines := runtime.NewPipelineService()
	deployments := runtime.NewDeploymentService()
	artifacts := runtime.NewArtifactService()
	releases := runtime.NewReleaseOrchestrationServiceWith(artifacts, deployments)
	security := runtime.NewSecurityService()
	approval := runtime.NewApprovalService()
	compliance := runtime.NewComplianceService(pipelines, deployments, artifacts, releases, security, approval)
	server := NewServer(Services{
		Config:       config.Default(),
		Subject:      domainauth.Subject{ID: "mcp-auditor", Username: "mcp-auditor", Roles: []string{domainauth.RoleAuditor}, AuthMode: "token"},
		Auth:         runtime.NewAuthService(),
		Pipelines:    pipelines,
		PipelineDefs: pipelineusecase.NewDefinitionCatalog(pipelineusecase.NewDefinitionMemoryStore()),
		Deployments:  deployments,
		Catalog:      catalogusecase.NewService(catalogusecase.NewMemoryStore()),
		Artifacts:    artifacts,
		Releases:     releases,
		Security:     security,
		Compliance:   compliance,
		Plugins:      runtime.NewPluginRegistry(),
		Audit:        NewComplianceAuditRecorder(compliance),
	}, nil)

	if _, err := server.ReadResource(context.Background(), "nivora://audit/search"); err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	result, err := compliance.SearchAudit(context.Background(), complianceusecase.AuditSearchInput{Action: EventResourceRead, ActorID: "mcp-auditor"})
	if err != nil {
		t.Fatalf("SearchAudit: %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("expected persisted MCP audit, got %#v", result)
	}
	body := mustMarshal(t, result.Items[0])
	for _, forbidden := range []string{"Bearer should-not-leak", "tokenHash", "BEGIN PRIVATE KEY"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("persisted audit leaked sensitive value: %s", body)
		}
	}
}

func TestMCPRecordsAuditForOperations(t *testing.T) {
	recorder := &MemoryAuditRecorder{}
	server := newTestMCPServerWithRecorder(t, domainauth.Subject{
		ID:       "auditor",
		Username: "auditor",
		Roles:    []string{domainauth.RoleAuditor},
		AuthMode: "token",
	}, recorder)
	if _, err := server.ReadResource(context.Background(), "nivora://audit/search"); err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if _, err := server.CallTool(context.Background(), "nivora_apply_deployment", nil); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if _, err := server.GetPrompt(context.Background(), "mcp_safe_operation_check", map[string]string{"requestedAction": "apply"}); err != nil {
		t.Fatalf("GetPrompt: %v", err)
	}
	entries := recorder.Entries()
	if len(entries) < 3 {
		t.Fatalf("expected audit entries, got %#v", entries)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Action] = true
		body := mustMarshal(t, entry)
		for _, forbidden := range []string{"raw-token-value", "tokenHash", "BEGIN PRIVATE KEY", "Authorization: Bearer"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("audit leaked sensitive value: %s", body)
			}
		}
	}
	for _, want := range []string{EventResourceRead, EventToolDenied, EventPromptRendered} {
		if !seen[want] {
			t.Fatalf("missing audit action %s in %#v", want, entries)
		}
	}
}

func TestMCPAuditSanitizesReasonAndSubject(t *testing.T) {
	entry := newMCPAudit(domainauth.Subject{ID: "actor", AuthMode: "token"}, auditDecision{
		Event:    EventToolDenied,
		Subject:  "nivora_get_secret?token=raw-token-value",
		Scope:    "tool",
		Decision: "denied",
		Reason:   "Authorization: Bearer raw-token-value",
	})
	body := mustMarshal(t, entry)
	if strings.Contains(body, "raw-token-value") || strings.Contains(body, "Authorization: Bearer") {
		t.Fatalf("audit entry leaked sensitive data: %s", body)
	}
	if entry.Metadata["auth_mode"] != "token" || entry.Metadata["decision"] != "denied" {
		t.Fatalf("audit metadata missing: %#v", entry.Metadata)
	}
}

func TestMCPRedactionAcrossOutputs(t *testing.T) {
	sensitive := audit.AuditLog{
		ID:        "audit-sensitive",
		ActorID:   "actor",
		Action:    EventToolDenied,
		Subject:   "subject",
		Reason:    "password=secret-password-value",
		Metadata:  map[string]string{"token_hash": "hash-should-not-leak", "private_key": "-----BEGIN PRIVATE KEY-----"},
		CreatedAt: newMCPAudit(domainauth.Subject{ID: "actor"}, auditDecision{Event: EventToolDenied}).CreatedAt,
	}
	body := mustJSON(map[string]any{
		"resource": sensitive,
		"tool":     errorToolResult(OperationError{Code: "mcp_invalid_arguments", Message: "Bearer raw-token-value"}),
		"prompt":   "Never print kubeconfig: apiVersion: v1",
	})
	for _, forbidden := range []string{"secret-password-value", "hash-should-not-leak", "BEGIN PRIVATE KEY", "raw-token-value", "apiVersion: v1"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("sensitive output leaked %q in %s", forbidden, body)
		}
	}
}

func newTestMCPServer(t *testing.T, role string, authMode string) *Server {
	t.Helper()
	server, _ := newTestMCPServerAndDeploymentService(t, role, authMode)
	return server
}

func newTestMCPServerAndDeploymentService(t *testing.T, role string, authMode string) (*Server, *deploymentusecase.Service) {
	t.Helper()
	deployments := runtime.NewDeploymentService()
	server := newTestMCPServerWithServices(t, domainauth.Subject{
		ID:          "subject-" + role,
		Username:    "subject-" + role,
		DisplayName: "subject-" + role,
		Roles:       []string{role},
		AuthMode:    authMode,
	}, deployments)
	return server, deployments
}

func newTestMCPServerWithSubject(t *testing.T, subject domainauth.Subject) *Server {
	t.Helper()
	return newTestMCPServerWithServices(t, subject, runtime.NewDeploymentService())
}

func newTestMCPServerWithServices(t *testing.T, subject domainauth.Subject, deploymentSvc *deploymentusecase.Service) *Server {
	t.Helper()
	return newTestMCPServerWithRecorderAndServices(t, subject, &MemoryAuditRecorder{}, deploymentSvc)
}

func newTestMCPServerWithRecorder(t *testing.T, subject domainauth.Subject, recorder *MemoryAuditRecorder) *Server {
	t.Helper()
	return newTestMCPServerWithRecorderAndServices(t, subject, recorder, runtime.NewDeploymentService())
}

func newTestMCPServerWithRecorderAndServices(t *testing.T, subject domainauth.Subject, recorder *MemoryAuditRecorder, deploymentSvc *deploymentusecase.Service) *Server {
	t.Helper()
	cfg := config.Default()
	cfg.MCP.AllowPlanTools = true
	pipelines := runtime.NewPipelineService()
	artifacts := runtime.NewArtifactService()
	releases := runtime.NewReleaseOrchestrationServiceWith(artifacts, deploymentSvc)
	security := runtime.NewSecurityService()
	approval := runtime.NewApprovalService()
	return NewServer(Services{
		Config:       cfg,
		Subject:      subject,
		Auth:         runtime.NewAuthService(),
		Pipelines:    pipelines,
		PipelineDefs: pipelineusecase.NewDefinitionCatalog(pipelineusecase.NewDefinitionMemoryStore()),
		Deployments:  deploymentSvc,
		Catalog:      catalogusecase.NewService(catalogusecase.NewMemoryStore()),
		Artifacts:    artifacts,
		Releases:     releases,
		Security:     security,
		Compliance:   runtime.NewComplianceService(pipelines, deploymentSvc, artifacts, releases, security, approval),
		Plugins:      runtime.NewPluginRegistry(),
		Audit:        recorder,
	}, nil)
}

func createMCPCatalogAndPipelineFixture(t *testing.T, server *Server) (string, string) {
	t.Helper()
	ctx := context.Background()
	org, err := server.services.Catalog.CreateOrg(ctx, catalogusecase.CreateOrgInput{Name: "Sevoniva"})
	if err != nil {
		t.Fatalf("create org fixture: %v", err)
	}
	project, err := server.services.Catalog.CreateProject(ctx, catalogusecase.CreateProjectInput{OrgID: org.ID, Name: "Nivora"})
	if err != nil {
		t.Fatalf("create project fixture: %v", err)
	}
	if _, err := server.services.Catalog.CreateApplication(ctx, catalogusecase.CreateApplicationInput{ProjectID: project.ID, Name: "demo-api"}); err != nil {
		t.Fatalf("create application fixture: %v", err)
	}
	environment, err := server.services.Catalog.CreateEnvironment(ctx, catalogusecase.CreateEnvironmentInput{ProjectID: project.ID, Name: "staging"})
	if err != nil {
		t.Fatalf("create environment fixture: %v", err)
	}
	if _, err := server.services.Catalog.CreateRepository(ctx, catalogusecase.CreateRepositoryInput{ProjectID: project.ID, Name: "demo-repo", URL: "https://example.invalid/sevoniva/demo.git", CredentialRef: "credential-ref-placeholder"}); err != nil {
		t.Fatalf("create repository fixture: %v", err)
	}
	if _, err := server.services.Catalog.CreateReleaseTarget(ctx, catalogusecase.CreateReleaseTargetInput{ProjectID: project.ID, EnvironmentID: environment.ID, Name: "noop-staging", TargetType: "noop"}); err != nil {
		t.Fatalf("create target fixture: %v", err)
	}
	definition, err := server.services.PipelineDefs.Create(ctx, pipelineusecase.DefinitionCreateInput{
		ProjectID: project.ID,
		Definition: pipelineusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Pipeline",
			Metadata:   pipelineusecase.Metadata{Name: "mcp-demo-pipeline"},
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "build",
				Jobs: []pipelineusecase.Job{{
					Name:     "echo",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "hello", Run: "echo hello"}},
				}},
			}}},
		},
	})
	if err != nil {
		t.Fatalf("create pipeline definition fixture: %v", err)
	}
	return project.ID, definition.Pipeline.ID
}

func createMCPArtifactFixture(t *testing.T, server *Server) (string, string) {
	t.Helper()
	record, err := server.services.Artifacts.CreateRelease(context.Background(), artifactusecase.CreateReleaseInput{
		ActorID: "mcp-fixture",
		Definition: artifactusecase.ReleaseDefinition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Release",
			Metadata: artifactusecase.ReleaseMetadata{
				Name: "mcp-artifact-release",
			},
			Spec: artifactusecase.ReleaseSpec{
				Version:     "1.2.3",
				Application: "demo",
				Environment: "staging",
				Artifacts: []artifactusecase.ReleaseArtifactSpec{{
					Name:      "demo-image",
					Type:      "image",
					Role:      "runtime",
					Required:  true,
					Reference: "registry.example.com/team/demo:1.2.3@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					Metadata: map[string]string{
						"component": "api",
					},
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("create artifact fixture release: %v", err)
	}
	if len(record.Artifacts) != 1 {
		t.Fatalf("expected one artifact in fixture, got %#v", record.Artifacts)
	}
	return record.Artifacts[0].ID, record.Release.ID
}

func createMCPObservabilityFixture(t *testing.T, server *Server) (string, string) {
	t.Helper()
	ctx := context.Background()
	pipelineRecord, err := server.services.Pipelines.CreateAndRun(ctx, pipelineusecase.CreateRunInput{
		ActorID: "mcp-fixture",
		Definition: pipelineusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Pipeline",
			Metadata:   pipelineusecase.Metadata{Name: "mcp-observe-pipeline"},
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "observe",
				Jobs: []pipelineusecase.Job{{
					Name:     "emit-log",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "stdout", Run: `printf "mcp-observe-log"`}},
				}},
			}}},
		},
	})
	if err != nil {
		t.Fatalf("create pipeline observability fixture: %v", err)
	}

	manifest := filepath.Join(t.TempDir(), "configmap.yaml")
	if err := os.WriteFile(manifest, []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: mcp-observe-config
data:
  message: observed
`), 0o600); err != nil {
		t.Fatalf("write deployment manifest fixture: %v", err)
	}
	deploymentRecord, err := server.services.Deployments.CreateAndRun(ctx, deploymentusecase.CreateRunInput{
		ActorID: "mcp-fixture",
		Definition: deploymentusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Deployment",
			Metadata:   deploymentusecase.Metadata{Name: "mcp-observe-deployment"},
			Spec: deploymentusecase.Spec{
				Application: "demo",
				Environment: "dev",
				Target: deploymentusecase.Target{
					Type:      "kubernetes-yaml",
					Name:      "local-noop",
					Namespace: "default",
				},
				Manifests: []string{manifest},
				Options: deploymentusecase.Options{
					DryRun: true,
					Apply:  false,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create deployment observability fixture: %v", err)
	}
	return pipelineRecord.Record.Run.ID, deploymentRecord.Record.Run.ID
}

func createScopedMCPDeploymentRun(t *testing.T, store *deploymentusecase.MemoryStore, deployments *deploymentusecase.Service, projectID string) string {
	t.Helper()
	result, err := deployments.CreateAndRun(context.Background(), deploymentusecase.CreateRunInput{
		ActorID: "mcp-scope-fixture",
		Definition: deploymentusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Deployment",
			Metadata:   deploymentusecase.Metadata{Name: "mcp-scope-" + projectID},
			Spec: deploymentusecase.Spec{
				Application: "demo",
				Environment: "dev",
				Target: deploymentusecase.Target{
					Type:      "kubernetes-yaml",
					Name:      "local",
					Namespace: "default",
				},
				Manifests: []string{"examples/yaml/configmap.yaml"},
				Options: deploymentusecase.Options{
					DryRun: true,
					Apply:  false,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("create scoped deployment fixture: %v", err)
	}
	record := result.Record
	record.Environment.ProjectID = projectID
	record.Target.ProjectID = projectID
	if err := store.Save(context.Background(), record); err != nil {
		t.Fatalf("save scoped deployment fixture: %v", err)
	}
	return record.Run.ID
}

func createScopedMCPPipelineRun(t *testing.T, pipelines *pipelineusecase.Service, projectID string) string {
	t.Helper()
	result, err := pipelines.CreateAndRun(context.Background(), pipelineusecase.CreateRunInput{
		ActorID:   "mcp-pipeline-scope-fixture",
		ProjectID: projectID,
		Definition: pipelineusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "Pipeline",
			Metadata:   pipelineusecase.Metadata{Name: "mcp-scope-" + projectID},
			Spec: pipelineusecase.Spec{Stages: []pipelineusecase.Stage{{
				Name: "scope",
				Jobs: []pipelineusecase.Job{{
					Name:     "emit",
					Executor: "shell",
					Steps:    []pipelineusecase.Step{{Name: "stdout", Run: `printf "` + projectID + `"`}},
				}},
			}}},
		},
	})
	if err != nil {
		t.Fatalf("create scoped pipeline fixture: %v", err)
	}
	return result.Record.Run.ID
}

func createScopedMCPReleaseExecution(t *testing.T, store releaseusecase.Store, releases *releaseusecase.Service, projectID string) string {
	t.Helper()
	record, err := releases.Deploy(context.Background(), releaseusecase.DeployInput{
		ActorID: "mcp-release-scope-fixture",
		Definition: releaseusecase.Definition{
			APIVersion: "nivora.io/v1alpha1",
			Kind:       "ReleaseOrchestration",
			Metadata:   releaseusecase.Metadata{Name: "mcp-scope-" + projectID},
			Spec: releaseusecase.Spec{
				Environment: "dev",
				Strategy:    releaseusecase.StrategyPlanOnly,
				Release: artifactusecase.ReleaseDefinition{
					APIVersion: "nivora.io/v1alpha1",
					Kind:       "Release",
					Metadata:   artifactusecase.ReleaseMetadata{Name: "mcp-release-" + projectID},
					Spec: artifactusecase.ReleaseSpec{
						Version:     "1.0.0",
						Application: "demo",
						Environment: "dev",
						Artifacts: []artifactusecase.ReleaseArtifactSpec{{
							Name:      "demo-image",
							Type:      "image",
							Reference: "registry.example.com/team/demo:1.0.0@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						}},
					},
				},
				Targets: []releaseusecase.TargetSpec{{
					Name: "noop-" + projectID,
					Type: "noop",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("create scoped release execution fixture: %v", err)
	}
	for i := range record.Plan.Targets {
		record.Plan.Targets[i].ProjectID = projectID
	}
	if err := store.SaveExecution(context.Background(), record); err != nil {
		t.Fatalf("save scoped release execution fixture: %v", err)
	}
	return record.Execution.ID
}

func resultCountIsZero(t *testing.T, result ToolResult) bool {
	t.Helper()
	if result.IsError || len(result.Content) == 0 {
		return false
	}
	var body struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &body); err != nil {
		t.Fatalf("unmarshal tool result: %v\n%s", err, result.Content[0].Text)
	}
	return body.Count == 0
}

func hasResource(resources []Resource, uri string) bool {
	for _, resource := range resources {
		if resource.URI == uri {
			return true
		}
	}
	return false
}

func hasTool(tools []Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func hasPrompt(prompts []Prompt, name string) bool {
	for _, prompt := range prompts {
		if prompt.Name == name {
			return true
		}
	}
	return false
}

func mustMarshal(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(body)
}

func deploymentDefinitionYAML() string {
	return `apiVersion: nivora.io/v1alpha1
kind: Deployment
metadata:
  name: mcp-plan-only
spec:
  application: demo
  environment: dev
  target:
    type: kubernetes-yaml
    name: local
    namespace: default
  manifests:
    - examples/yaml/deployment.yaml
  options:
    dryRun: true
    apply: false
`
}
