package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	"github.com/sevoniva/nivora/internal/infra/crypto"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

const jsonMime = "application/json"

var blockedActionTools = map[string]string{
	"nivora_apply_deployment":    "destructive deployment actions require a future guarded-action MCP phase",
	"nivora_sync_argocd":         "Argo CD sync requires explicit confirmation outside this MCP foundation",
	"nivora_execute_rollback":    "rollback execution is not exposed through MCP in this phase",
	"nivora_approve":             "approval decisions must use guarded control-plane APIs",
	"nivora_reject":              "approval decisions must use guarded control-plane APIs",
	"nivora_approve_request":     "approval decisions must use guarded control-plane APIs",
	"nivora_reject_request":      "approval decisions must use guarded control-plane APIs",
	"nivora_rotate_token":        "token rotation is intentionally excluded from MCP tools",
	"nivora_get_secret":          "secret value retrieval is never exposed by normal MCP tools",
	"nivora_register_runner":     "runner registration requires control-plane RBAC and one-time token handling",
	"nivora_remote_host_deploy":  "remote host deployment is intentionally excluded from MCP tools",
	"nivora_git_push":            "Git push is intentionally excluded from MCP tools",
	"nivora_kubernetes_prune":    "Kubernetes prune/delete actions are intentionally excluded from MCP tools",
	"nivora_kubernetes_delete":   "Kubernetes prune/delete actions are intentionally excluded from MCP tools",
	"nivora_rollback_deployment": "rollback execution is not exposed through MCP in this phase",
}

type Server struct {
	services Services
	logger   *slog.Logger
}

func NewServer(services Services, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	if services.Audit == nil {
		services.Audit = &MemoryAuditRecorder{}
	}
	if services.Catalog == nil {
		services.Catalog = catalogusecase.NewService(catalogusecase.NewMemoryStore())
	}
	if services.PipelineDefs == nil {
		services.PipelineDefs = pipelineusecase.NewDefinitionCatalog(pipelineusecase.NewDefinitionMemoryStore())
	}
	return &Server{services: services, logger: logger}
}

func (s *Server) ListResources(ctx context.Context) ([]Resource, error) {
	if err := s.require(ctx, domainauth.PermissionProjectRead, "mcp.resources", "resources/list"); err != nil {
		return nil, err
	}
	return []Resource{
		resource("nivora://capabilities/current", "Capability status", "Current implemented/partial/foundation capability status"),
		resource("nivora://system/runtime", "Runtime status", "Runtime configuration and recovery summary"),
		resource("nivora://api/inventory", "API inventory", "Current public API inventory"),
		resource("nivora://catalog/summary", "Catalog summary", "Organization, project, application, environment, repository, and target summary"),
		resource("nivora://pipelines/definitions", "Pipeline definitions", "Pipeline definition catalog"),
		resource("nivora://pipelines/definitions/{id}", "Pipeline definition", "Pipeline definition record by id"),
		resource("nivora://pipelines/runs/{id}", "PipelineRun", "PipelineRun record by id"),
		resource("nivora://pipelines/runs/{id}/timeline", "PipelineRun timeline", "PipelineRun timeline by id"),
		resource("nivora://pipelines/runs/{id}/logs", "PipelineRun logs", "PipelineRun logs by id"),
		resource("nivora://deployments/{id}", "DeploymentRun", "DeploymentRun record by id"),
		resource("nivora://deployments/{id}/timeline", "DeploymentRun timeline", "DeploymentRun timeline by id"),
		resource("nivora://deployments/{id}/resources", "Deployment resources", "Deployment resource inventory by id"),
		resource("nivora://deployments/{id}/health", "Deployment health", "Deployment health by id"),
		resource("nivora://deployments/{id}/diff", "Deployment diff", "Deployment diff by id"),
		resource("nivora://releases/{id}", "Release", "Release record by id"),
		resource("nivora://artifacts", "Artifacts", "Release-bound artifact inventory"),
		resource("nivora://artifacts/{id}", "Artifact", "Tracked artifact by id"),
		resource("nivora://artifacts/{id}/releases", "Artifact release bindings", "Release bindings for a tracked artifact"),
		resource("nivora://releases/executions/{id}", "ReleaseExecution", "ReleaseExecution record by id"),
		resource("nivora://releases/executions/{id}/timeline", "ReleaseExecution timeline", "ReleaseExecution timeline by id"),
		resource("nivora://runners/summary", "Runner summary", "Runner fleet summary"),
		resource("nivora://security/summary", "Security summary", "Security scan summary"),
		resource("nivora://audit/search", "Audit search", "Audit records visible to the MCP subject"),
		resource("nivora://evidence/bundles/{id}", "Evidence bundle", "Persisted evidence bundle by id"),
		resource("nivora://plugins/capabilities", "Plugin capabilities", "Built-in plugin capability metadata"),
	}, nil
}

func (s *Server) ReadResource(ctx context.Context, uri string) (ResourceContent, error) {
	if err := s.checkResourcePermission(ctx, uri); err != nil {
		s.record(ctx, EventResourceRead, uri, "system", "denied", err.Error())
		return ResourceContent{}, err
	}
	payload, err := s.readResourcePayload(ctx, uri)
	if err != nil {
		s.record(ctx, EventResourceRead, uri, "system", "failed", err.Error())
		return ResourceContent{}, err
	}
	s.record(ctx, EventResourceRead, uri, "system", "allowed", "resource read")
	return ResourceContent{URI: uri, MimeType: jsonMime, Text: mustJSON(payload)}, nil
}

func (s *Server) ListTools(ctx context.Context) ([]Tool, error) {
	if err := s.require(ctx, domainauth.PermissionProjectRead, "mcp.tools", "tools/list"); err != nil {
		return nil, err
	}
	tools := []Tool{
		tool("nivora_status", "Read current Nivora runtime and capability status", nil),
		tool("nivora_get_catalog_summary", "Read organization, project, application, environment, repository, and target catalog summary", objectSchema(map[string]any{
			"orgId":     stringProperty("optional org id filter"),
			"projectId": stringProperty("optional project id filter"),
		}, nil)),
		tool("nivora_list_pipeline_definitions", "List pipeline definitions", objectSchema(map[string]any{
			"projectId": stringProperty("optional project id filter"),
		}, nil)),
		tool("nivora_get_pipeline_definition", "Read a pipeline definition by id", idSchema("id")),
		tool("nivora_get_pipeline_run", "Read a PipelineRun by id", idSchema("id")),
		tool("nivora_get_pipeline_timeline", "Read a PipelineRun timeline by id", idSchema("id")),
		tool("nivora_get_deployment", "Read a DeploymentRun by id", idSchema("id")),
		tool("nivora_get_deployment_health", "Read DeploymentRun health by id", idSchema("id")),
		tool("nivora_get_deployment_diff", "Read DeploymentRun diff by id", idSchema("id")),
		tool("nivora_get_release_execution", "Read a ReleaseExecution by id", idSchema("id")),
		tool("nivora_list_artifacts", "List release-bound artifacts tracked by Nivora", objectSchema(map[string]any{
			"type":       stringProperty("artifact type"),
			"name":       stringProperty("artifact name substring"),
			"registry":   stringProperty("registry host"),
			"repository": stringProperty("repository substring"),
			"digest":     stringProperty("resolved digest"),
			"reference":  stringProperty("artifact reference substring"),
		}, nil)),
		tool("nivora_get_artifact", "Read a tracked artifact by id", idSchema("id")),
		tool("nivora_get_artifact_releases", "List releases bound to a tracked artifact", idSchema("id")),
		tool("nivora_get_runner_summary", "Read runner fleet summary", nil),
		tool("nivora_get_evidence_bundle", "Read a persisted evidence bundle by id", idSchema("id")),
		tool("nivora_search_audit", "Search audit records visible to the subject", objectSchema(map[string]any{
			"subject":       stringProperty("subject substring"),
			"actorId":       stringProperty("actor id"),
			"action":        stringProperty("action substring"),
			"scopeType":     stringProperty("scope type"),
			"scopeId":       stringProperty("scope id"),
			"correlationId": stringProperty("correlation id"),
		}, nil)),
		tool("nivora_get_capability_status", "Read the current capability status document", nil),
	}
	if s.services.Config.MCP.AllowPlanTools {
		tools = append(tools,
			tool("nivora_explain_pipeline_failure", "Explain PipelineRun failure facts and likely next checks", idSchema("id")),
			tool("nivora_explain_deployment", "Explain DeploymentRun risk from health, diff, warnings, and resources", idSchema("id")),
			tool("nivora_explain_deployment_risk", "Explain DeploymentRun risk from health, diff, warnings, and resources", idSchema("id")),
			tool("nivora_explain_release", "Generate a ReleaseExecution readiness summary", idSchema("id")),
			tool("nivora_generate_release_readiness_summary", "Generate a ReleaseExecution readiness summary", idSchema("id")),
			tool("nivora_evaluate_policy_local", "Evaluate local policy inputs without storing a result", objectSchema(map[string]any{
				"subjectType": stringProperty("artifact, manifest, deployment_plan, or release"),
				"subjectId":   stringProperty("subject id"),
				"reference":   stringProperty("artifact reference"),
				"content":     stringProperty("manifest content"),
			}, []string{"subjectType", "subjectId"})),
			tool("nivora_inspect_artifact", "Inspect an artifact reference without registry network access", objectSchema(map[string]any{
				"reference": stringProperty("artifact reference"),
				"type":      stringProperty("artifact type, defaults to image"),
			}, []string{"reference"})),
			tool("nivora_inspect_artifact_reference", "Inspect an artifact reference without registry network access", objectSchema(map[string]any{
				"reference": stringProperty("artifact reference"),
				"type":      stringProperty("artifact type, defaults to image"),
			}, []string{"reference"})),
			tool("nivora_plan_deployment_local", "Parse a deployment definition and return a non-mutating plan summary", objectSchema(map[string]any{
				"file":    stringProperty("local deployment definition file"),
				"content": stringProperty("deployment definition YAML/JSON content"),
			}, nil)),
		)
	}
	return tools, nil
}

func (s *Server) CallTool(ctx context.Context, name string, arguments map[string]any) (ToolResult, error) {
	if gate, ok := blockedActionTools[name]; ok {
		err := OperationError{Code: "mcp_action_not_allowed", Message: name + " is not exposed by the MCP foundation", RequiredFutureGate: gate}
		s.record(ctx, EventToolDenied, name, "system", "denied", err.Message)
		return errorToolResult(err), nil
	}
	permission := s.toolPermission(name)
	if permission == "" {
		err := OperationError{Code: "mcp_tool_not_found", Message: "unknown MCP tool " + name}
		s.record(ctx, EventToolDenied, name, "system", "denied", err.Message)
		return errorToolResult(err), nil
	}
	if err := s.require(ctx, permission, "mcp.tool", name); err != nil {
		s.record(ctx, EventToolDenied, name, "system", "denied", err.Error())
		return errorToolResult(err), nil
	}
	payload, err := s.callToolPayload(ctx, name, arguments)
	if err != nil {
		s.record(ctx, EventToolCalled, name, "system", "failed", err.Error())
		return errorToolResult(err), nil
	}
	s.record(ctx, EventToolCalled, name, "system", "allowed", "tool called")
	return textToolResult(mustJSON(payload)), nil
}

func (s *Server) ListPrompts(ctx context.Context) ([]Prompt, error) {
	if err := s.require(ctx, domainauth.PermissionProjectRead, "mcp.prompts", "prompts/list"); err != nil {
		return nil, err
	}
	return promptCatalog(), nil
}

func (s *Server) GetPrompt(ctx context.Context, name string, args map[string]string) (PromptResult, error) {
	if err := s.require(ctx, domainauth.PermissionProjectRead, "mcp.prompt", name); err != nil {
		s.record(ctx, EventPromptRendered, name, "system", "denied", err.Error())
		return PromptResult{}, err
	}
	text, ok := promptText(name, args)
	if !ok {
		return PromptResult{}, OperationError{Code: "mcp_prompt_not_found", Message: "unknown MCP prompt " + name}
	}
	s.record(ctx, EventPromptRendered, name, "system", "allowed", "prompt rendered")
	return PromptResult{
		Description: "Nivora " + name + " prompt",
		Messages: []PromptMessage{{
			Role: "user",
			Content: PromptContent{
				Type: "text",
				Text: text,
			},
		}},
	}, nil
}

func (s *Server) readResourcePayload(ctx context.Context, uri string) (any, error) {
	switch {
	case uri == "nivora://capabilities/current":
		body, err := readProjectFile("docs/status/CAPABILITY_STATUS.md")
		if err != nil {
			return nil, err
		}
		return map[string]any{"maturity": "hardened beta-candidate", "productionReady": false, "content": string(body)}, nil
	case uri == "nivora://system/runtime":
		runtimeStatus, err := s.services.Pipelines.RuntimeStatus(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"config": s.runtimeConfigSummary(), "runtime": runtimeStatus}, nil
	case uri == "nivora://api/inventory":
		body, err := readProjectFile("docs/API_INVENTORY.md")
		if err != nil {
			return nil, err
		}
		return map[string]any{"content": string(body)}, nil
	case uri == "nivora://catalog/summary":
		return s.catalogSummary(ctx, "", "")
	case uri == "nivora://pipelines/definitions":
		definitions, err := s.services.PipelineDefs.List(ctx, "")
		if err != nil {
			return nil, err
		}
		return map[string]any{"definitions": definitions}, nil
	case strings.HasPrefix(uri, "nivora://pipelines/definitions/"):
		id := strings.TrimPrefix(uri, "nivora://pipelines/definitions/")
		return s.services.PipelineDefs.Get(ctx, id)
	case strings.HasPrefix(uri, "nivora://pipelines/runs/"):
		return s.pipelineResource(ctx, strings.TrimPrefix(uri, "nivora://pipelines/runs/"))
	case strings.HasPrefix(uri, "nivora://deployments/"):
		return s.deploymentResource(ctx, strings.TrimPrefix(uri, "nivora://deployments/"))
	case uri == "nivora://artifacts":
		artifacts, err := s.services.Artifacts.ListArtifacts(ctx, artifactusecase.ListArtifactsInput{})
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifacts": artifacts}, nil
	case strings.HasPrefix(uri, "nivora://artifacts/"):
		return s.artifactResource(ctx, strings.TrimPrefix(uri, "nivora://artifacts/"))
	case strings.HasPrefix(uri, "nivora://releases/executions/"):
		return s.releaseExecutionResource(ctx, strings.TrimPrefix(uri, "nivora://releases/executions/"))
	case strings.HasPrefix(uri, "nivora://releases/"):
		id := strings.TrimPrefix(uri, "nivora://releases/")
		return s.services.Artifacts.GetRelease(ctx, id)
	case uri == "nivora://runners/summary":
		return s.runnerSummary(ctx)
	case uri == "nivora://security/summary":
		return s.securitySummary(ctx)
	case uri == "nivora://audit/search":
		return s.services.Compliance.SearchAudit(ctx, complianceusecase.AuditSearchInput{})
	case strings.HasPrefix(uri, "nivora://evidence/bundles/"):
		id := strings.TrimPrefix(uri, "nivora://evidence/bundles/")
		return s.services.Compliance.GetEvidenceBundle(ctx, id)
	case uri == "nivora://plugins/capabilities":
		plugins, err := s.services.Plugins.List(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"plugins": plugins}, nil
	default:
		return nil, OperationError{Code: "mcp_resource_not_found", Message: "unknown MCP resource " + uri}
	}
}

func readProjectFile(path string) ([]byte, error) {
	if body, err := os.ReadFile(path); err == nil {
		return body, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for {
		candidate := filepath.Join(cwd, path)
		if body, err := os.ReadFile(candidate); err == nil {
			return body, nil
		}
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
	return os.ReadFile(path)
}

func (s *Server) pipelineResource(ctx context.Context, rest string) (any, error) {
	if strings.HasSuffix(rest, "/timeline") {
		id := strings.TrimSuffix(rest, "/timeline")
		return s.services.Pipelines.Timeline(ctx, id)
	}
	if strings.HasSuffix(rest, "/logs") {
		id := strings.TrimSuffix(rest, "/logs")
		logs, err := s.services.Pipelines.Logs(ctx, id)
		return truncateLogs(logs), err
	}
	return s.services.Pipelines.Get(ctx, rest)
}

func (s *Server) deploymentResource(ctx context.Context, rest string) (any, error) {
	switch {
	case strings.HasSuffix(rest, "/timeline"):
		return s.services.Deployments.Timeline(ctx, strings.TrimSuffix(rest, "/timeline"))
	case strings.HasSuffix(rest, "/resources"):
		return s.services.Deployments.Resources(ctx, strings.TrimSuffix(rest, "/resources"))
	case strings.HasSuffix(rest, "/health"):
		return s.services.Deployments.Health(ctx, strings.TrimSuffix(rest, "/health"))
	case strings.HasSuffix(rest, "/diff"):
		return s.services.Deployments.Diff(ctx, strings.TrimSuffix(rest, "/diff"))
	default:
		return s.services.Deployments.Get(ctx, rest)
	}
}

func (s *Server) releaseExecutionResource(ctx context.Context, rest string) (any, error) {
	if strings.HasSuffix(rest, "/timeline") {
		id := strings.TrimSuffix(rest, "/timeline")
		return s.services.Releases.Timeline(ctx, id)
	}
	return s.services.Releases.GetExecution(ctx, rest)
}

func (s *Server) artifactResource(ctx context.Context, rest string) (any, error) {
	if strings.HasSuffix(rest, "/releases") {
		id := strings.TrimSuffix(rest, "/releases")
		id = strings.TrimSuffix(id, "/")
		bindings, err := s.services.Artifacts.ArtifactReleases(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifactId": id, "releases": bindings}, nil
	}
	return s.services.Artifacts.GetArtifact(ctx, rest)
}

func (s *Server) callToolPayload(ctx context.Context, name string, arguments map[string]any) (any, error) {
	switch name {
	case "nivora_status":
		status, err := s.services.Pipelines.RuntimeStatus(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"maturity": "hardened beta-candidate", "productionReady": false, "runtime": status}, nil
	case "nivora_get_catalog_summary":
		return s.catalogSummary(ctx, stringArg(arguments, "orgId"), stringArg(arguments, "projectId"))
	case "nivora_list_pipeline_definitions":
		definitions, err := s.services.PipelineDefs.List(ctx, stringArg(arguments, "projectId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"definitions": definitions, "mutated": false}, nil
	case "nivora_get_pipeline_definition":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		definition, err := s.services.PipelineDefs.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"definition": definition, "mutated": false}, nil
	case "nivora_get_pipeline_run":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Pipelines.Get(ctx, id)
	case "nivora_get_pipeline_timeline":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Pipelines.Timeline(ctx, id)
	case "nivora_get_deployment":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Deployments.Get(ctx, id)
	case "nivora_get_deployment_health":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Deployments.Health(ctx, id)
	case "nivora_get_deployment_diff":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Deployments.Diff(ctx, id)
	case "nivora_get_release_execution":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Releases.GetExecution(ctx, id)
	case "nivora_list_artifacts":
		artifacts, err := s.services.Artifacts.ListArtifacts(ctx, artifactusecase.ListArtifactsInput{
			Type:       stringArg(arguments, "type"),
			Name:       stringArg(arguments, "name"),
			Registry:   stringArg(arguments, "registry"),
			Repository: stringArg(arguments, "repository"),
			Digest:     stringArg(arguments, "digest"),
			Reference:  stringArg(arguments, "reference"),
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifacts": artifacts, "mutated": false}, nil
	case "nivora_get_artifact":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		artifact, err := s.services.Artifacts.GetArtifact(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifact": artifact, "mutated": false}, nil
	case "nivora_get_artifact_releases":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		bindings, err := s.services.Artifacts.ArtifactReleases(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifactId": id, "releases": bindings, "mutated": false}, nil
	case "nivora_get_runner_summary":
		return s.runnerSummary(ctx)
	case "nivora_get_evidence_bundle":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Compliance.GetEvidenceBundle(ctx, id)
	case "nivora_search_audit":
		return s.services.Compliance.SearchAudit(ctx, complianceusecase.AuditSearchInput{
			Subject:       stringArg(arguments, "subject"),
			ActorID:       stringArg(arguments, "actorId"),
			Action:        stringArg(arguments, "action"),
			ScopeType:     stringArg(arguments, "scopeType"),
			ScopeID:       stringArg(arguments, "scopeId"),
			CorrelationID: stringArg(arguments, "correlationId"),
		})
	case "nivora_get_capability_status":
		return s.readResourcePayload(ctx, "nivora://capabilities/current")
	case "nivora_explain_pipeline_failure":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.explainPipeline(ctx, id)
	case "nivora_explain_deployment", "nivora_explain_deployment_risk":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.explainDeploymentRisk(ctx, id)
	case "nivora_explain_release", "nivora_generate_release_readiness_summary":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.releaseReadiness(ctx, id)
	case "nivora_evaluate_policy_local":
		result := s.services.Security.Evaluate(securityusecase.EvaluateInput{
			SubjectType: domainsecurity.SubjectType(firstNonEmpty(stringArg(arguments, "subjectType"), "artifact")),
			SubjectID:   firstNonEmpty(stringArg(arguments, "subjectId"), "mcp-local"),
			Reference:   stringArg(arguments, "reference"),
			Findings:    manifestFindings(stringArg(arguments, "content")),
			ActorID:     s.services.Subject.ID,
		})
		return map[string]any{"policyResult": result, "mutated": false}, nil
	case "nivora_inspect_artifact", "nivora_inspect_artifact_reference":
		artifactType := domainartifact.ArtifactType(firstNonEmpty(stringArg(arguments, "type"), string(domainartifact.ArtifactTypeImage)))
		reference, err := requiredString(arguments, "reference")
		if err != nil {
			return nil, err
		}
		inspection, err := s.services.Artifacts.Inspect(ctx, reference, artifactType)
		if err != nil {
			return nil, err
		}
		return map[string]any{"inspection": inspection, "mutated": false}, nil
	case "nivora_plan_deployment_local":
		return s.planDeploymentLocal(arguments)
	default:
		return nil, OperationError{Code: "mcp_tool_not_found", Message: "unknown MCP tool " + name}
	}
}

func (s *Server) explainPipeline(ctx context.Context, id string) (any, error) {
	record, err := s.services.Pipelines.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	timeline, _ := s.services.Pipelines.Timeline(ctx, id)
	logs, _ := s.services.Pipelines.Logs(ctx, id)
	return map[string]any{
		"pipelineRunId": id,
		"status":        record.Run.Status,
		"facts":         timeline,
		"logPreview":    truncateLogs(logs),
		"inference":     "Review failed job status, recent stderr log chunks, timeout/cancel flags, and runner assignment before rerunning.",
		"mutated":       false,
	}, nil
}

func (s *Server) explainDeploymentRisk(ctx context.Context, id string) (any, error) {
	record, err := s.services.Deployments.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	health, _ := s.services.Deployments.Health(ctx, id)
	diff, _ := s.services.Deployments.Diff(ctx, id)
	resources, _ := s.services.Deployments.Resources(ctx, id)
	return map[string]any{
		"deploymentRunId": id,
		"status":          record.Run.Status,
		"health":          health,
		"diff":            diff,
		"resources":       resources,
		"warnings":        record.Plan.Warnings,
		"inference":       "Treat apply, sync, rollback, host deploy, and prune as separate guarded actions outside this MCP foundation.",
		"mutated":         false,
	}, nil
}

func (s *Server) releaseReadiness(ctx context.Context, id string) (any, error) {
	record, err := s.services.Releases.GetExecution(ctx, id)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"releaseExecutionId": id,
		"status":             record.Execution.Status,
		"targets":            record.Execution.Targets,
		"security":           record.Security,
		"approval":           record.Approval,
		"mutated":            false,
		"recommendation":     "Confirm policy, approvals, target health, artifact digest state, and rollback readiness before executing any guarded action.",
	}, nil
}

func (s *Server) planDeploymentLocal(arguments map[string]any) (any, error) {
	var (
		def deploymentusecase.Definition
		err error
	)
	if content := stringArg(arguments, "content"); content != "" {
		def, err = deploymentusecase.ParseDefinition([]byte(content))
	} else {
		file, fileErr := requiredString(arguments, "file")
		if fileErr != nil {
			return nil, fileErr
		}
		def, err = deploymentusecase.LoadDefinitionFile(file)
	}
	if err != nil {
		return nil, err
	}
	warnings := []string{}
	if def.Spec.Options.Apply {
		warnings = append(warnings, "apply requested in definition, but MCP plan-only tool will not execute apply")
	}
	if def.Spec.GitOps.Sync {
		warnings = append(warnings, "Argo CD sync requested in definition, but MCP plan-only tool will not execute sync")
	}
	if def.Spec.Host.AllowRemoteHostDeploy {
		warnings = append(warnings, "remote host deploy requested in definition, but MCP plan-only tool will not execute host deploy")
	}
	return map[string]any{
		"name":          def.Metadata.Name,
		"targetType":    def.Spec.Target.Type,
		"targetName":    def.Spec.Target.Name,
		"environment":   def.Spec.Environment,
		"manifestCount": len(def.Spec.Manifests),
		"artifactCount": len(def.Spec.Artifacts),
		"warnings":      warnings,
		"mutated":       false,
	}, nil
}

func (s *Server) runnerSummary(ctx context.Context) (any, error) {
	runners, err := s.services.Pipelines.ListRunners(ctx)
	if err != nil {
		return nil, err
	}
	statusCounts := map[string]int{}
	for _, runner := range runners {
		statusCounts[runner.Status]++
	}
	return map[string]any{"total": len(runners), "statusCounts": statusCounts, "runners": runners}, nil
}

func (s *Server) securitySummary(ctx context.Context) (any, error) {
	scans, err := s.services.Security.List(ctx)
	if err != nil {
		return nil, err
	}
	totalFindings := 0
	for _, scan := range scans {
		totalFindings += scan.Scan.Summary.Total
	}
	return map[string]any{"scanCount": len(scans), "findingCount": totalFindings, "scans": scans}, nil
}

func (s *Server) catalogSummary(ctx context.Context, orgID string, projectID string) (any, error) {
	orgs, err := s.services.Catalog.ListOrgs(ctx)
	if err != nil {
		return nil, err
	}
	projects, err := s.services.Catalog.ListProjects(ctx, orgID)
	if err != nil {
		return nil, err
	}
	applications, err := s.services.Catalog.ListApplications(ctx, projectID)
	if err != nil {
		return nil, err
	}
	environments, err := s.services.Catalog.ListEnvironments(ctx, projectID)
	if err != nil {
		return nil, err
	}
	repositories, err := s.services.Catalog.ListRepositories(ctx, projectID)
	if err != nil {
		return nil, err
	}
	targets, err := s.services.Catalog.ListReleaseTargets(ctx, projectID, "")
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"filters": map[string]string{
			"orgId":     orgID,
			"projectId": projectID,
		},
		"counts": map[string]int{
			"orgs":           len(orgs),
			"projects":       len(projects),
			"applications":   len(applications),
			"environments":   len(environments),
			"repositories":   len(repositories),
			"releaseTargets": len(targets),
		},
		"orgs":           orgs,
		"projects":       projects,
		"applications":   applications,
		"environments":   environments,
		"repositories":   repositories,
		"releaseTargets": targets,
		"mutated":        false,
	}, nil
}

func (s *Server) checkResourcePermission(ctx context.Context, uri string) error {
	if uri == "nivora://audit/search" || strings.HasPrefix(uri, "nivora://evidence/bundles/") {
		return s.require(ctx, domainauth.PermissionAuditRead, "mcp.resource", uri)
	}
	return s.require(ctx, domainauth.PermissionProjectRead, "mcp.resource", uri)
}

func (s *Server) toolPermission(name string) string {
	switch name {
	case "nivora_search_audit", "nivora_get_evidence_bundle":
		return domainauth.PermissionAuditRead
	case "nivora_explain_pipeline_failure",
		"nivora_explain_deployment",
		"nivora_explain_deployment_risk",
		"nivora_explain_release",
		"nivora_generate_release_readiness_summary",
		"nivora_evaluate_policy_local",
		"nivora_inspect_artifact",
		"nivora_inspect_artifact_reference",
		"nivora_plan_deployment_local":
		return domainauth.PermissionDeploymentCreate
	case "nivora_status",
		"nivora_get_catalog_summary",
		"nivora_list_pipeline_definitions",
		"nivora_get_pipeline_definition",
		"nivora_get_pipeline_run",
		"nivora_get_pipeline_timeline",
		"nivora_get_deployment",
		"nivora_get_deployment_health",
		"nivora_get_deployment_diff",
		"nivora_get_release_execution",
		"nivora_list_artifacts",
		"nivora_get_artifact",
		"nivora_get_artifact_releases",
		"nivora_get_runner_summary",
		"nivora_get_capability_status":
		return domainauth.PermissionProjectRead
	default:
		return ""
	}
}

func (s *Server) require(_ context.Context, permission string, resourceType string, resourceID string) error {
	subject := s.services.Subject
	if subject.AuthMode == "runner_token" || strings.HasPrefix(subject.ID, "runner:") {
		return OperationError{Code: "mcp_runner_token_denied", Message: "runner tokens cannot use MCP control-plane tools"}
	}
	if subject.ID == "" {
		return OperationError{Code: "mcp_unauthorized", Message: "MCP subject is not authenticated"}
	}
	if permission == "" || s.services.Auth == nil {
		return nil
	}
	decision := s.services.Auth.Evaluate(authusecase.EvaluateInput{
		Subject:  subject,
		Action:   permission,
		Resource: domainauth.Resource{Type: resourceType, ID: resourceID},
	})
	if !decision.Allowed {
		return OperationError{Code: "mcp_forbidden", Message: decision.Reason}
	}
	return nil
}

func (s *Server) record(ctx context.Context, event string, subject string, scope string, decision string, reason string) {
	if s.services.Audit != nil {
		if err := s.services.Audit.RecordMCPAudit(ctx, newMCPAudit(s.services.Subject, auditDecision{Event: event, Subject: subject, Scope: scope, Decision: decision, Reason: reason})); err != nil {
			s.logger.Warn("mcp audit record failed", "event", event, "subject", subject, "error", err)
		}
	}
	s.logger.Info("mcp operation", "event", event, "subject", subject, "decision", decision, "reason", reason)
}

func (s *Server) runtimeConfigSummary() map[string]any {
	cfg := s.services.Config
	return map[string]any{
		"environment":       cfg.Env,
		"runtimeStore":      cfg.Database.RuntimeStore,
		"eventBus":          cfg.EventBus.Type,
		"objectStore":       cfg.ObjectStore.Type,
		"mcpMode":           cfg.MCP.Mode,
		"mcpReadOnly":       cfg.MCP.ReadOnly,
		"mcpAllowPlanTools": cfg.MCP.AllowPlanTools,
		"productionReady":   false,
	}
}

func resource(uri string, name string, description string) Resource {
	return Resource{URI: uri, Name: name, Description: description, MimeType: jsonMime}
}

func tool(name string, description string, schema map[string]any) Tool {
	return Tool{Name: name, Description: description, InputSchema: schema}
}

func idSchema(name string) map[string]any {
	return objectSchema(map[string]any{name: stringProperty(name)}, []string{name})
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required}
}

func stringProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func textToolResult(text string) ToolResult {
	return ToolResult{Content: []ToolContent{{Type: "text", Text: text}}}
}

func errorToolResult(err error) ToolResult {
	var op OperationError
	if !errors.As(err, &op) {
		op = OperationError{Code: "mcp_tool_failed", Message: err.Error()}
	}
	return ToolResult{IsError: true, Content: []ToolContent{{Type: "text", Text: mustJSON(op)}}}
}

func mustJSON(value any) string {
	safe := sanitizeAny(value)
	body, err := json.MarshalIndent(safe, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"code":"mcp_encode_failed","message":%q}`, err.Error())
	}
	return string(body)
}

func sanitizeAny(value any) any {
	body, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return value
	}
	return sanitizeJSON(decoded)
}

func sanitizeJSON(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, item := range v {
			if strings.EqualFold(key, "code") {
				out[key] = item
				continue
			}
			if crypto.IsSensitiveKey(key) && !isSafeProtocolKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = sanitizeJSON(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = sanitizeJSON(v[i])
		}
		return out
	case string:
		return crypto.RedactString(v)
	default:
		return value
	}
}

func isSafeProtocolKey(key string) bool {
	switch strings.ToLower(key) {
	case "code", "message", "requiredfuturegate":
		return true
	default:
		return false
	}
}

func truncateLogs(logs any) any {
	body, _ := json.Marshal(logs)
	const max = 32 * 1024
	if len(body) <= max {
		return logs
	}
	return map[string]any{
		"truncated": true,
		"preview":   string(body[:max]),
	}
}

func requiredString(args map[string]any, key string) (string, error) {
	value := stringArg(args, key)
	if value == "" {
		return "", OperationError{Code: "mcp_invalid_arguments", Message: key + " is required"}
	}
	return value, nil
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func manifestFindings(content string) []domainsecurity.SecurityFinding {
	if content == "" {
		return nil
	}
	lower := strings.ToLower(content)
	var findings []domainsecurity.SecurityFinding
	if strings.Contains(lower, "privileged: true") {
		findings = append(findings, domainsecurity.SecurityFinding{Severity: domainsecurity.SeverityHigh, Category: domainsecurity.CategoryMisconfiguration, Target: "manifest", Title: "Privileged container requested"})
	}
	if strings.Contains(lower, "hostpath:") {
		findings = append(findings, domainsecurity.SecurityFinding{Severity: domainsecurity.SeverityMedium, Category: domainsecurity.CategoryMisconfiguration, Target: "manifest", Title: "hostPath volume requested"})
	}
	return findings
}
