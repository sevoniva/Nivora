package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	domainartifact "github.com/sevoniva/nivora/internal/domain/artifact"
	domainauth "github.com/sevoniva/nivora/internal/domain/auth"
	domainevent "github.com/sevoniva/nivora/internal/domain/event"
	domainrunner "github.com/sevoniva/nivora/internal/domain/runner"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	domaintenant "github.com/sevoniva/nivora/internal/domain/tenant"
	"github.com/sevoniva/nivora/internal/infra/crypto"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
	catalogusecase "github.com/sevoniva/nivora/internal/usecase/catalog"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseusecase "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
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
	services  Services
	logger    *slog.Logger
	rateLimit *rateLimitState
}

type rateLimitState struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
}

type rateLimitEntry struct {
	window time.Time
	count  int
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
	return &Server{services: services, logger: logger, rateLimit: &rateLimitState{}}
}

func (s *Server) WithSubject(subject domainauth.Subject) *Server {
	if s == nil {
		return nil
	}
	copy := *s
	copy.services = s.services
	copy.services.Subject = subject
	if copy.rateLimit == nil {
		copy.rateLimit = &rateLimitState{}
	}
	return &copy
}

func (s *Server) ListResources(ctx context.Context) ([]Resource, error) {
	if err := s.require(ctx, domainauth.PermissionProjectRead, "mcp.resources", "resources/list"); err != nil {
		return nil, err
	}
	return []Resource{
		resource("nivora://capabilities/current", "Capability status", "Current implemented/partial/foundation capability status"),
		resource("nivora://system/runtime", "Runtime status", "Runtime configuration and recovery summary"),
		resource("nivora://runtime/recovery", "Runtime recovery status", "Pipeline, deployment, release, and outbox recovery summary"),
		resource("nivora://api/inventory", "API inventory", "Current public API inventory"),
		resource("nivora://events", "Runtime events", "Aggregate runtime events with MCP response caps"),
		resource("nivora://logs", "Runtime logs", "Aggregate runtime log chunks with MCP response caps"),
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
		resource("nivora://security/findings", "Security findings", "Security findings summary and current finding list"),
		resource("nivora://policy/results/summary", "Policy result summary", "Policy gate decision summary derived from security scan records"),
		resource("nivora://audit/search", "Audit search", "Audit records visible to the MCP subject"),
		resource("nivora://evidence/bundles", "Evidence bundles", "Persisted evidence bundles visible to the MCP subject"),
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
	return ResourceContent{URI: uri, MimeType: jsonMime, Text: s.capResponseText(mustJSON(payload))}, nil
}

func (s *Server) ListTools(ctx context.Context) ([]Tool, error) {
	if err := s.require(ctx, domainauth.PermissionProjectRead, "mcp.tools", "tools/list"); err != nil {
		return nil, err
	}
	tools := []Tool{
		tool("nivora_status", "Read current Nivora runtime and capability status", nil),
		tool("nivora_get_runtime_recovery_status", "Read runtime recovery status across pipeline, deployment, release, and outbox state", nil),
		tool("nivora_search_events", "Search aggregate runtime events with MCP response caps", objectSchema(map[string]any{
			"type":            stringProperty("event type substring"),
			"source":          stringProperty("event source substring"),
			"subject":         stringProperty("event subject substring"),
			"runId":           stringProperty("pipeline or deployment run id"),
			"pipelineRunId":   stringProperty("PipelineRun id"),
			"deploymentRunId": stringProperty("DeploymentRun id"),
			"releaseId":       stringProperty("Release id"),
			"artifactId":      stringProperty("Artifact id"),
			"securityScanId":  stringProperty("SecurityScan id"),
			"limit":           integerProperty("page size, 1-100"),
			"offset":          integerProperty("zero-based offset"),
		}, nil)),
		tool("nivora_search_logs", "Search aggregate runtime logs with MCP response caps", objectSchema(map[string]any{
			"runId":           stringProperty("pipeline or deployment run id"),
			"pipelineRunId":   stringProperty("PipelineRun id"),
			"deploymentRunId": stringProperty("DeploymentRun id"),
			"stageRunId":      stringProperty("StageRun id"),
			"jobRunId":        stringProperty("JobRun id"),
			"stepRunId":       stringProperty("StepRun id"),
			"stream":          stringProperty("stdout, stderr, or system"),
			"contains":        stringProperty("case-insensitive log content substring"),
			"limit":           integerProperty("page size, 1-100"),
			"offset":          integerProperty("zero-based offset"),
		}, nil)),
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
		tool("nivora_list_security_findings", "List security findings with optional filters", objectSchema(map[string]any{
			"scanId":      stringProperty("optional scan id"),
			"subjectType": stringProperty("optional subject type"),
			"subjectId":   stringProperty("optional subject id"),
			"severity":    stringProperty("optional severity"),
			"category":    stringProperty("optional category"),
		}, nil)),
		tool("nivora_get_policy_result_summary", "Read policy gate decision summary from security scan records", objectSchema(map[string]any{
			"subjectType": stringProperty("optional subject type"),
			"subjectId":   stringProperty("optional subject id"),
		}, nil)),
		tool("nivora_get_evidence_bundle", "Read a persisted evidence bundle by id", idSchema("id")),
		tool("nivora_list_evidence_bundles", "List persisted evidence bundles", objectSchema(map[string]any{
			"subjectType": stringProperty("optional evidence subject type"),
			"subjectId":   stringProperty("optional evidence subject id"),
		}, nil)),
		tool("nivora_search_audit", "Search audit records visible to the subject", objectSchema(map[string]any{
			"subject":       stringProperty("subject substring"),
			"subjectType":   stringProperty("subject type"),
			"subjectId":     stringProperty("subject id"),
			"actorId":       stringProperty("actor id"),
			"action":        stringProperty("action substring"),
			"scopeType":     stringProperty("scope type"),
			"scopeId":       stringProperty("scope id"),
			"requestId":     stringProperty("request id"),
			"correlationId": stringProperty("correlation id"),
			"limit":         integerProperty("page size, 1-100"),
			"offset":        integerProperty("zero-based offset"),
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
	return textToolResult(s.capResponseText(mustJSON(payload))), nil
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
		return map[string]any{
			"maturity":        "hardened beta-candidate",
			"productionReady": false,
			"scope":           s.subjectScopeSummary(),
			"warnings":        s.capabilityStatusWarnings(),
			"content":         string(body),
		}, nil
	case uri == "nivora://system/runtime":
		return s.runtimeRecoveryStatus(ctx)
	case uri == "nivora://runtime/recovery":
		return s.runtimeRecoveryStatus(ctx)
	case uri == "nivora://api/inventory":
		body, err := readProjectFile("docs/API_INVENTORY.md")
		if err != nil {
			return nil, err
		}
		return map[string]any{"content": string(body)}, nil
	case uri == "nivora://events":
		return s.eventSearch(ctx, mcpEventFilter{})
	case uri == "nivora://logs":
		return s.logSearch(ctx, mcpLogFilter{})
	case uri == "nivora://catalog/summary":
		return s.catalogSummary(ctx, "", "")
	case uri == "nivora://pipelines/definitions":
		definitions, err := s.scopedPipelineDefinitions(ctx, "")
		if err != nil {
			return nil, err
		}
		return map[string]any{"definitions": definitions}, nil
	case strings.HasPrefix(uri, "nivora://pipelines/definitions/"):
		id := strings.TrimPrefix(uri, "nivora://pipelines/definitions/")
		definition, err := s.services.PipelineDefs.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensurePipelineDefinitionScope(definition, "pipeline definition "+id); err != nil {
			return nil, err
		}
		return definition, nil
	case strings.HasPrefix(uri, "nivora://pipelines/runs/"):
		return s.pipelineResource(ctx, strings.TrimPrefix(uri, "nivora://pipelines/runs/"))
	case strings.HasPrefix(uri, "nivora://deployments/"):
		return s.deploymentResource(ctx, strings.TrimPrefix(uri, "nivora://deployments/"))
	case uri == "nivora://artifacts":
		artifacts, err := s.services.Artifacts.ListArtifacts(ctx, s.scopedArtifactListInput(artifactusecase.ListArtifactsInput{}))
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
	case uri == "nivora://security/findings":
		return s.securityFindings(ctx, securityusecase.ListFindingsInput{})
	case uri == "nivora://policy/results/summary":
		return s.policyResultSummary(ctx, "", "")
	case uri == "nivora://audit/search":
		return s.auditSearch(ctx, complianceusecase.AuditSearchInput{}, mcpPage{})
	case uri == "nivora://evidence/bundles":
		bundles, err := s.services.Compliance.SearchEvidenceBundles(ctx, "", "")
		if err != nil {
			return nil, err
		}
		return map[string]any{"bundles": bundles, "count": len(bundles), "mutated": false}, nil
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
		record, err := s.services.Pipelines.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensurePipelineScope(record, "pipeline run "+id); err != nil {
			return nil, err
		}
		return s.services.Pipelines.Timeline(ctx, id)
	}
	if strings.HasSuffix(rest, "/logs") {
		id := strings.TrimSuffix(rest, "/logs")
		record, err := s.services.Pipelines.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensurePipelineScope(record, "pipeline run "+id); err != nil {
			return nil, err
		}
		logs, err := s.services.Pipelines.Logs(ctx, id)
		return truncateLogs(logs), err
	}
	record, err := s.services.Pipelines.Get(ctx, rest)
	if err != nil {
		return nil, err
	}
	if err := s.ensurePipelineScope(record, "pipeline run "+rest); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Server) deploymentResource(ctx context.Context, rest string) (any, error) {
	switch {
	case strings.HasSuffix(rest, "/timeline"):
		id := strings.TrimSuffix(rest, "/timeline")
		record, err := s.services.Deployments.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensureDeploymentScope(record, "deployment run "+id); err != nil {
			return nil, err
		}
		return s.services.Deployments.Timeline(ctx, id)
	case strings.HasSuffix(rest, "/resources"):
		id := strings.TrimSuffix(rest, "/resources")
		record, err := s.services.Deployments.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensureDeploymentScope(record, "deployment run "+id); err != nil {
			return nil, err
		}
		return s.services.Deployments.Resources(ctx, id)
	case strings.HasSuffix(rest, "/health"):
		id := strings.TrimSuffix(rest, "/health")
		record, err := s.services.Deployments.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensureDeploymentScope(record, "deployment run "+id); err != nil {
			return nil, err
		}
		return s.services.Deployments.Health(ctx, id)
	case strings.HasSuffix(rest, "/diff"):
		id := strings.TrimSuffix(rest, "/diff")
		record, err := s.services.Deployments.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensureDeploymentScope(record, "deployment run "+id); err != nil {
			return nil, err
		}
		return s.services.Deployments.Diff(ctx, id)
	default:
		record, err := s.services.Deployments.Get(ctx, rest)
		if err != nil {
			return nil, err
		}
		if err := s.ensureDeploymentScope(record, "deployment run "+rest); err != nil {
			return nil, err
		}
		return record, nil
	}
}

func (s *Server) releaseExecutionResource(ctx context.Context, rest string) (any, error) {
	if strings.HasSuffix(rest, "/timeline") {
		id := strings.TrimSuffix(rest, "/timeline")
		record, err := s.services.Releases.GetExecution(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.ensureReleaseExecutionScope(record, "release execution "+id); err != nil {
			return nil, err
		}
		return s.services.Releases.Timeline(ctx, id)
	}
	record, err := s.services.Releases.GetExecution(ctx, rest)
	if err != nil {
		return nil, err
	}
	if err := s.ensureReleaseExecutionScope(record, "release execution "+rest); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *Server) artifactResource(ctx context.Context, rest string) (any, error) {
	if strings.HasSuffix(rest, "/releases") {
		id := strings.TrimSuffix(rest, "/releases")
		id = strings.TrimSuffix(id, "/")
		artifact, err := s.services.Artifacts.GetArtifact(ctx, id)
		if err != nil {
			return nil, err
		}
		if !s.canReadArtifact(artifact) {
			return nil, OperationError{Code: "mcp_scope_denied", Message: "MCP subject scope does not allow access to artifact " + id}
		}
		bindings, err := s.services.Artifacts.ArtifactReleases(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifactId": id, "releases": s.filterArtifactReleaseBindings(bindings)}, nil
	}
	artifact, err := s.services.Artifacts.GetArtifact(ctx, rest)
	if err != nil {
		return nil, err
	}
	if !s.canReadArtifact(artifact) {
		return nil, OperationError{Code: "mcp_scope_denied", Message: "MCP subject scope does not allow access to artifact " + rest}
	}
	return artifact, nil
}

func (s *Server) callToolPayload(ctx context.Context, name string, arguments map[string]any) (any, error) {
	switch name {
	case "nivora_status":
		status, err := s.services.Pipelines.RuntimeStatus(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"maturity": "hardened beta-candidate", "productionReady": false, "runtime": status}, nil
	case "nivora_get_runtime_recovery_status":
		return s.runtimeRecoveryStatus(ctx)
	case "nivora_search_events":
		page, err := mcpPageFromArgs(arguments)
		if err != nil {
			return nil, err
		}
		return s.eventSearch(ctx, mcpEventFilter{
			Type:            stringArg(arguments, "type"),
			Source:          stringArg(arguments, "source"),
			Subject:         stringArg(arguments, "subject"),
			RunID:           stringArg(arguments, "runId"),
			PipelineRunID:   stringArg(arguments, "pipelineRunId"),
			DeploymentRunID: stringArg(arguments, "deploymentRunId"),
			ReleaseID:       stringArg(arguments, "releaseId"),
			ArtifactID:      stringArg(arguments, "artifactId"),
			SecurityScanID:  stringArg(arguments, "securityScanId"),
			Limit:           page.Limit,
			Offset:          page.Offset,
		})
	case "nivora_search_logs":
		page, err := mcpPageFromArgs(arguments)
		if err != nil {
			return nil, err
		}
		return s.logSearch(ctx, mcpLogFilter{
			RunID:           stringArg(arguments, "runId"),
			PipelineRunID:   stringArg(arguments, "pipelineRunId"),
			DeploymentRunID: stringArg(arguments, "deploymentRunId"),
			StageRunID:      stringArg(arguments, "stageRunId"),
			JobRunID:        stringArg(arguments, "jobRunId"),
			StepRunID:       stringArg(arguments, "stepRunId"),
			Stream:          stringArg(arguments, "stream"),
			Contains:        stringArg(arguments, "contains"),
			Limit:           page.Limit,
			Offset:          page.Offset,
		})
	case "nivora_get_catalog_summary":
		return s.catalogSummary(ctx, stringArg(arguments, "orgId"), stringArg(arguments, "projectId"))
	case "nivora_list_pipeline_definitions":
		definitions, err := s.scopedPipelineDefinitions(ctx, stringArg(arguments, "projectId"))
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
		if err := s.ensurePipelineDefinitionScope(definition, "pipeline definition "+id); err != nil {
			return nil, err
		}
		return map[string]any{"definition": definition, "mutated": false}, nil
	case "nivora_get_pipeline_run":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.pipelineResource(ctx, id)
	case "nivora_get_pipeline_timeline":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.pipelineResource(ctx, id+"/timeline")
	case "nivora_get_deployment":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.deploymentResource(ctx, id)
	case "nivora_get_deployment_health":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.deploymentResource(ctx, id+"/health")
	case "nivora_get_deployment_diff":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.deploymentResource(ctx, id+"/diff")
	case "nivora_get_release_execution":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.releaseExecutionResource(ctx, id)
	case "nivora_list_artifacts":
		artifacts, err := s.services.Artifacts.ListArtifacts(ctx, s.scopedArtifactListInput(artifactusecase.ListArtifactsInput{
			Type:       stringArg(arguments, "type"),
			Name:       stringArg(arguments, "name"),
			Registry:   stringArg(arguments, "registry"),
			Repository: stringArg(arguments, "repository"),
			Digest:     stringArg(arguments, "digest"),
			Reference:  stringArg(arguments, "reference"),
		}))
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
		if !s.canReadArtifact(artifact) {
			return nil, OperationError{Code: "mcp_scope_denied", Message: "MCP subject scope does not allow access to artifact " + id}
		}
		return map[string]any{"artifact": artifact, "mutated": false}, nil
	case "nivora_get_artifact_releases":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		artifact, err := s.services.Artifacts.GetArtifact(ctx, id)
		if err != nil {
			return nil, err
		}
		if !s.canReadArtifact(artifact) {
			return nil, OperationError{Code: "mcp_scope_denied", Message: "MCP subject scope does not allow access to artifact " + id}
		}
		bindings, err := s.services.Artifacts.ArtifactReleases(ctx, id)
		if err != nil {
			return nil, err
		}
		return map[string]any{"artifactId": id, "releases": s.filterArtifactReleaseBindings(bindings), "mutated": false}, nil
	case "nivora_get_runner_summary":
		return s.runnerSummary(ctx)
	case "nivora_list_security_findings":
		return s.securityFindings(ctx, securityusecase.ListFindingsInput{
			ScanID:      stringArg(arguments, "scanId"),
			SubjectType: domainsecurity.SubjectType(stringArg(arguments, "subjectType")),
			SubjectID:   stringArg(arguments, "subjectId"),
			Severity:    domainsecurity.Severity(stringArg(arguments, "severity")),
			Category:    domainsecurity.FindingCategory(stringArg(arguments, "category")),
		})
	case "nivora_get_policy_result_summary":
		return s.policyResultSummary(ctx, stringArg(arguments, "subjectType"), stringArg(arguments, "subjectId"))
	case "nivora_get_evidence_bundle":
		id, err := requiredString(arguments, "id")
		if err != nil {
			return nil, err
		}
		return s.services.Compliance.GetEvidenceBundle(ctx, id)
	case "nivora_list_evidence_bundles":
		bundles, err := s.services.Compliance.SearchEvidenceBundles(ctx, stringArg(arguments, "subjectType"), stringArg(arguments, "subjectId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"bundles": bundles, "count": len(bundles), "mutated": false}, nil
	case "nivora_search_audit":
		page, err := mcpPageFromArgs(arguments)
		if err != nil {
			return nil, err
		}
		return s.auditSearch(ctx, complianceusecase.AuditSearchInput{
			Subject:       stringArg(arguments, "subject"),
			SubjectType:   stringArg(arguments, "subjectType"),
			SubjectID:     stringArg(arguments, "subjectId"),
			ActorID:       stringArg(arguments, "actorId"),
			Action:        stringArg(arguments, "action"),
			ScopeType:     stringArg(arguments, "scopeType"),
			ScopeID:       stringArg(arguments, "scopeId"),
			RequestID:     stringArg(arguments, "requestId"),
			CorrelationID: stringArg(arguments, "correlationId"),
		}, page)
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
	if err := s.ensurePipelineScope(record, "pipeline run "+id); err != nil {
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
	if err := s.ensureDeploymentScope(record, "deployment run "+id); err != nil {
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
	if err := s.ensureReleaseExecutionScope(record, "release execution "+id); err != nil {
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
	runners = s.filterRunnersForSubject(runners)
	statusCounts := map[string]int{}
	for _, runner := range runners {
		statusCounts[runner.Status]++
	}
	return map[string]any{"total": len(runners), "statusCounts": statusCounts, "runners": runners, "scope": s.subjectScopeSummary(), "mutated": false}, nil
}

func (s *Server) runtimeRecoveryStatus(ctx context.Context) (any, error) {
	const defaultLimit = 100
	staleAfter := 2 * time.Minute
	pipelineStatus, err := s.services.Pipelines.RuntimeStatus(ctx)
	if err != nil {
		return nil, err
	}
	deploymentStatus, err := s.services.Deployments.RuntimeStatus(ctx, staleAfter, defaultLimit)
	if err != nil {
		return nil, err
	}
	releaseStatus, err := s.services.Releases.RuntimeStatus(ctx, staleAfter, defaultLimit)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"config":       s.runtimeConfigSummary(),
		"pipeline":     pipelineStatus,
		"deployment":   deploymentStatus,
		"release":      releaseStatus,
		"staleAfter":   staleAfter.String(),
		"limit":        defaultLimit,
		"mutated":      false,
		"readOnly":     true,
		"nextSafeStep": "Use guarded runtime reconcile APIs or CLI outside MCP when recovery action is explicitly approved.",
	}, nil
}

func (s *Server) securitySummary(ctx context.Context) (any, error) {
	scans, err := s.services.Security.List(ctx)
	if err != nil {
		return nil, err
	}
	totalFindings := 0
	severityCounts := map[string]int{}
	categoryCounts := map[string]int{}
	decisionCounts := map[string]int{}
	visibleScans := make([]securityusecase.ScanRecord, 0, len(scans))
	for _, scan := range scans {
		if !s.canReadSecurityScan(scan) {
			continue
		}
		visibleScans = append(visibleScans, scan)
		totalFindings += scan.Scan.Summary.Total
		if scan.Policy.Decision != "" {
			decisionCounts[string(scan.Policy.Decision)]++
		}
		for _, finding := range scan.Scan.Findings {
			severityCounts[string(finding.Severity)]++
			categoryCounts[string(finding.Category)]++
		}
	}
	return map[string]any{
		"scanCount":            len(visibleScans),
		"findingCount":         totalFindings,
		"severityCounts":       severityCounts,
		"categoryCounts":       categoryCounts,
		"policyDecisionCounts": decisionCounts,
		"scans":                visibleScans,
		"mutated":              false,
	}, nil
}

func (s *Server) securityFindings(ctx context.Context, input securityusecase.ListFindingsInput) (any, error) {
	records, err := s.securityScanRecords(ctx, input)
	if err != nil {
		return nil, err
	}
	findings := securityFindingsFromRecords(records, input)
	severityCounts := map[string]int{}
	categoryCounts := map[string]int{}
	for _, finding := range findings {
		severityCounts[string(finding.Severity)]++
		categoryCounts[string(finding.Category)]++
	}
	return map[string]any{
		"filters": map[string]string{
			"scanId":      input.ScanID,
			"subjectType": string(input.SubjectType),
			"subjectId":   input.SubjectID,
			"severity":    string(input.Severity),
			"category":    string(input.Category),
		},
		"summary": map[string]any{
			"total":          len(findings),
			"severityCounts": severityCounts,
			"categoryCounts": categoryCounts,
		},
		"findings": findings,
		"mutated":  false,
	}, nil
}

func (s *Server) securityScanRecords(ctx context.Context, input securityusecase.ListFindingsInput) ([]securityusecase.ScanRecord, error) {
	var records []securityusecase.ScanRecord
	if strings.TrimSpace(input.ScanID) != "" {
		record, err := s.services.Security.Get(ctx, strings.TrimSpace(input.ScanID))
		if err != nil {
			return nil, err
		}
		records = []securityusecase.ScanRecord{record}
	} else {
		var err error
		records, err = s.services.Security.ListScans(ctx, securityusecase.ListScansInput{
			SubjectType:   input.SubjectType,
			SubjectID:     input.SubjectID,
			ProjectID:     input.ProjectID,
			EnvironmentID: input.EnvironmentID,
		})
		if err != nil {
			return nil, err
		}
	}
	filtered := make([]securityusecase.ScanRecord, 0, len(records))
	for _, record := range records {
		if s.canReadSecurityScan(record) {
			filtered = append(filtered, record)
		}
	}
	return filtered, nil
}

func securityFindingsFromRecords(records []securityusecase.ScanRecord, input securityusecase.ListFindingsInput) []domainsecurity.SecurityFinding {
	findings := make([]domainsecurity.SecurityFinding, 0)
	for _, record := range records {
		for _, finding := range record.Scan.Findings {
			if input.Severity != "" && finding.Severity != input.Severity {
				continue
			}
			if input.Category != "" && finding.Category != input.Category {
				continue
			}
			copied := finding
			metadata := make(map[string]string, len(copied.Metadata)+5)
			for key, value := range copied.Metadata {
				metadata[key] = value
			}
			metadata["scanId"] = record.Scan.ID
			metadata["subjectType"] = string(record.Scan.SubjectType)
			metadata["subjectId"] = record.Scan.SubjectID
			if record.Scan.ProjectID != "" {
				metadata["projectId"] = record.Scan.ProjectID
			}
			if record.Scan.EnvironmentID != "" {
				metadata["environmentId"] = record.Scan.EnvironmentID
			}
			copied.Metadata = metadata
			findings = append(findings, copied)
		}
	}
	return findings
}

func (s *Server) policyResultSummary(ctx context.Context, subjectType string, subjectID string) (any, error) {
	scans, err := s.services.Security.ListScans(ctx, securityusecase.ListScansInput{
		SubjectType: domainsecurity.SubjectType(subjectType),
		SubjectID:   subjectID,
	})
	if err != nil {
		return nil, err
	}
	decisionCounts := map[string]int{}
	results := make([]map[string]any, 0, len(scans))
	for _, scan := range scans {
		if !s.canReadSecurityScan(scan) {
			continue
		}
		decision := string(scan.Policy.Decision)
		if decision == "" {
			decision = "unknown"
		}
		decisionCounts[decision]++
		results = append(results, map[string]any{
			"scanId":       scan.Scan.ID,
			"subjectType":  scan.Scan.SubjectType,
			"subjectId":    scan.Scan.SubjectID,
			"decision":     scan.Policy.Decision,
			"reason":       scan.Policy.Reason,
			"findingCount": scan.Scan.Summary.Total,
			"evaluatedAt":  scan.Policy.EvaluatedAt,
		})
	}
	return map[string]any{
		"filters": map[string]string{
			"subjectType": subjectType,
			"subjectId":   subjectID,
		},
		"decisionCounts": decisionCounts,
		"policyResults":  results,
		"mutated":        false,
	}, nil
}

const (
	mcpAggregateResponseLimit = 100
	mcpAggregateMaxPageLimit  = 100
)

type mcpEventFilter struct {
	Type            string
	Source          string
	Subject         string
	RunID           string
	PipelineRunID   string
	DeploymentRunID string
	ReleaseID       string
	ArtifactID      string
	SecurityScanID  string
	Limit           int
	Offset          int
}

type mcpLogFilter struct {
	RunID           string
	PipelineRunID   string
	DeploymentRunID string
	StageRunID      string
	JobRunID        string
	StepRunID       string
	Stream          string
	Contains        string
	Limit           int
	Offset          int
}

type mcpPage struct {
	Limit  int
	Offset int
}

func (s *Server) eventSearch(ctx context.Context, filter mcpEventFilter) (any, error) {
	var (
		events   []domainevent.Event
		warnings []string
	)
	if records, err := s.services.Pipelines.List(ctx); err != nil {
		warnings = append(warnings, "pipeline events unavailable: "+err.Error())
	} else {
		for _, record := range records {
			if !s.canReadPipeline(record) {
				continue
			}
			events = append(events, record.Events...)
		}
	}
	if records, err := s.services.Deployments.List(ctx); err != nil {
		warnings = append(warnings, "deployment events unavailable: "+err.Error())
	} else {
		for _, record := range records {
			if !s.canReadDeployment(record) {
				continue
			}
			events = append(events, record.Events...)
		}
	}
	if records, err := s.services.Releases.ListExecutions(ctx, ""); err != nil {
		warnings = append(warnings, "release execution events unavailable: "+err.Error())
	} else {
		for _, record := range records {
			if !s.canReadReleaseExecution(record) {
				continue
			}
			events = append(events, record.Events...)
		}
	}
	if records, err := s.services.Artifacts.ListReleases(ctx); err != nil {
		warnings = append(warnings, "artifact events unavailable: "+err.Error())
	} else {
		for _, record := range records {
			if !s.canReadArtifactReleaseRecord(record) {
				continue
			}
			events = append(events, record.Events...)
		}
	}
	if records, err := s.services.Security.List(ctx); err != nil {
		warnings = append(warnings, "security events unavailable: "+err.Error())
	} else {
		for _, record := range records {
			if !s.canReadSecurityScan(record) {
				continue
			}
			events = append(events, record.Events...)
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Time.Equal(events[j].Time) {
			return events[i].ID < events[j].ID
		}
		return events[i].Time.Before(events[j].Time)
	})
	filtered := make([]domainevent.Event, 0, len(events))
	for _, evt := range events {
		if filterMCPEvent(evt, filter) {
			filtered = append(filtered, evt)
		}
	}
	page, err := normalizeMCPPage(filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	limited, pagination, truncated := paginateMCPItems(filtered, page)
	return map[string]any{
		"filters":    filter,
		"events":     limited,
		"count":      len(filtered),
		"limit":      page.Limit,
		"offset":     page.Offset,
		"pagination": pagination,
		"truncated":  truncated,
		"warnings":   warnings,
		"mutated":    false,
	}, nil
}

func (s *Server) logSearch(ctx context.Context, filter mcpLogFilter) (any, error) {
	var (
		logs     []domainevent.LogChunk
		warnings []string
	)
	if records, err := s.services.Pipelines.List(ctx); err != nil {
		warnings = append(warnings, "pipeline logs unavailable: "+err.Error())
	} else {
		for _, record := range records {
			if !s.canReadPipeline(record) {
				continue
			}
			logs = append(logs, record.Logs...)
		}
	}
	if records, err := s.services.Deployments.List(ctx); err != nil {
		warnings = append(warnings, "deployment logs unavailable: "+err.Error())
	} else {
		for _, record := range records {
			if !s.canReadDeployment(record) {
				continue
			}
			logs = append(logs, record.Logs...)
		}
	}
	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].CreatedAt.Equal(logs[j].CreatedAt) {
			if logs[i].Sequence == logs[j].Sequence {
				return logs[i].ID < logs[j].ID
			}
			return logs[i].Sequence < logs[j].Sequence
		}
		return logs[i].CreatedAt.Before(logs[j].CreatedAt)
	})
	filtered := make([]domainevent.LogChunk, 0, len(logs))
	for _, log := range logs {
		if filterMCPLog(log, filter) {
			filtered = append(filtered, log)
		}
	}
	page, err := normalizeMCPPage(filter.Limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	limited, pagination, truncated := paginateMCPItems(filtered, page)
	return map[string]any{
		"filters":    filter,
		"logs":       truncateLogs(limited),
		"count":      len(filtered),
		"limit":      page.Limit,
		"offset":     page.Offset,
		"pagination": pagination,
		"truncated":  truncated,
		"warnings":   warnings,
		"mutated":    false,
	}, nil
}

func (s *Server) auditSearch(ctx context.Context, input complianceusecase.AuditSearchInput, page mcpPage) (any, error) {
	result, err := s.services.Compliance.SearchAudit(ctx, input)
	if err != nil {
		return nil, err
	}
	normalizedPage, err := normalizeMCPPage(page.Limit, page.Offset)
	if err != nil {
		return nil, err
	}
	limited, pagination, truncated := paginateMCPItems(result.Items, normalizedPage)
	return map[string]any{
		"items":      limited,
		"count":      result.Count,
		"limit":      normalizedPage.Limit,
		"offset":     normalizedPage.Offset,
		"pagination": pagination,
		"truncated":  truncated,
		"mutated":    false,
	}, nil
}

func filterMCPEvent(evt domainevent.Event, filter mcpEventFilter) bool {
	if filter.Type != "" && !containsFoldMCP(evt.Type, filter.Type) {
		return false
	}
	if filter.Source != "" && !containsFoldMCP(evt.Source, filter.Source) {
		return false
	}
	if filter.Subject != "" && !containsFoldMCP(evt.Subject, filter.Subject) {
		return false
	}
	for _, id := range []string{
		filter.RunID,
		filter.PipelineRunID,
		filter.DeploymentRunID,
		filter.ReleaseID,
		filter.ArtifactID,
		filter.SecurityScanID,
	} {
		if id != "" && !mcpEventMatchesIdentifier(evt, id) {
			return false
		}
	}
	return true
}

func filterMCPLog(log domainevent.LogChunk, filter mcpLogFilter) bool {
	if filter.RunID != "" && log.PipelineRunID != filter.RunID && log.DeploymentRunID != filter.RunID {
		return false
	}
	if filter.PipelineRunID != "" && log.PipelineRunID != filter.PipelineRunID {
		return false
	}
	if filter.DeploymentRunID != "" && log.DeploymentRunID != filter.DeploymentRunID {
		return false
	}
	if filter.StageRunID != "" && log.StageRunID != filter.StageRunID {
		return false
	}
	if filter.JobRunID != "" && log.JobRunID != filter.JobRunID {
		return false
	}
	if filter.StepRunID != "" && log.StepRunID != filter.StepRunID {
		return false
	}
	if filter.Stream != "" && !strings.EqualFold(log.Stream, filter.Stream) {
		return false
	}
	if filter.Contains != "" && !containsFoldMCP(log.Content, filter.Contains) {
		return false
	}
	return true
}

func mcpEventMatchesIdentifier(evt domainevent.Event, id string) bool {
	if evt.ID == id || evt.Subject == id {
		return true
	}
	for _, value := range evt.Data {
		if mcpAnyString(value) == id {
			return true
		}
	}
	return false
}

func mcpAnyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}

func containsFoldMCP(value string, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func paginateMCPItems[T any](items []T, page mcpPage) ([]T, map[string]any, bool) {
	start := page.Offset
	if start > len(items) {
		start = len(items)
	}
	end := start + page.Limit
	if end > len(items) {
		end = len(items)
	}
	hasMore := end < len(items)
	return items[start:end], map[string]any{
		"limit":      page.Limit,
		"offset":     page.Offset,
		"total":      len(items),
		"nextOffset": end,
		"hasMore":    hasMore,
	}, hasMore
}

func normalizeMCPPage(limit int, offset int) (mcpPage, error) {
	if limit == 0 {
		limit = mcpAggregateResponseLimit
	}
	if limit < 0 {
		return mcpPage{}, OperationError{Code: "mcp_invalid_arguments", Message: "limit must be a positive integer"}
	}
	if limit > mcpAggregateMaxPageLimit {
		return mcpPage{}, OperationError{Code: "mcp_invalid_arguments", Message: fmt.Sprintf("limit must be <= %d", mcpAggregateMaxPageLimit)}
	}
	if offset < 0 {
		return mcpPage{}, OperationError{Code: "mcp_invalid_arguments", Message: "offset must be a non-negative integer"}
	}
	return mcpPage{Limit: limit, Offset: offset}, nil
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

func (s *Server) ensurePipelineScope(record pipelineusecase.RunRecord, resource string) error {
	return s.ensureSubjectScope(resource, record.Pipeline.ProjectID, "")
}

func (s *Server) ensurePipelineDefinitionScope(record pipelineusecase.DefinitionRecord, resource string) error {
	return s.ensureSubjectScope(resource, record.Pipeline.ProjectID, "")
}

func (s *Server) scopedPipelineDefinitions(ctx context.Context, requestedProjectID string) ([]pipelineusecase.DefinitionRecord, error) {
	subject := s.services.Subject
	switch strings.TrimSpace(subject.ScopeType) {
	case "":
		return s.services.PipelineDefs.List(ctx, requestedProjectID)
	case domaintenant.ScopeGlobal:
		return s.services.PipelineDefs.List(ctx, requestedProjectID)
	case domaintenant.ScopeProject:
		scopeID := strings.TrimSpace(subject.ScopeID)
		if scopeID == "" {
			return []pipelineusecase.DefinitionRecord{}, nil
		}
		return s.services.PipelineDefs.List(ctx, scopeID)
	default:
		return []pipelineusecase.DefinitionRecord{}, nil
	}
}

func (s *Server) ensureDeploymentScope(record deploymentusecase.RunRecord, resource string) error {
	projectID := firstNonEmpty(record.Environment.ProjectID, record.Target.ProjectID)
	environmentID := firstNonEmpty(record.Run.EnvironmentID, record.Environment.ID, record.Target.EnvironmentID)
	return s.ensureSubjectScope(resource, projectID, environmentID)
}

func (s *Server) ensureReleaseExecutionScope(record releaseusecase.ExecutionRecord, resource string) error {
	projectID := ""
	for _, target := range record.Plan.Targets {
		if target.ProjectID != "" {
			projectID = target.ProjectID
			break
		}
	}
	for _, deployment := range record.Deployments {
		projectID = firstNonEmpty(projectID, deployment.Environment.ProjectID, deployment.Target.ProjectID)
		if projectID != "" {
			break
		}
	}
	environmentID := firstNonEmpty(record.Execution.EnvironmentID, record.Plan.EnvironmentID)
	return s.ensureSubjectScope(resource, projectID, environmentID)
}

func (s *Server) canReadPipeline(record pipelineusecase.RunRecord) bool {
	return s.ensurePipelineScope(record, record.Run.ID) == nil
}

func (s *Server) canReadDeployment(record deploymentusecase.RunRecord) bool {
	return s.ensureDeploymentScope(record, record.Run.ID) == nil
}

func (s *Server) canReadReleaseExecution(record releaseusecase.ExecutionRecord) bool {
	return s.ensureReleaseExecutionScope(record, record.Execution.ID) == nil
}

func (s *Server) canReadSecurityScan(record securityusecase.ScanRecord) bool {
	return s.ensureSubjectScope(record.Scan.ID, record.Scan.ProjectID, record.Scan.EnvironmentID) == nil
}

func (s *Server) scopedArtifactListInput(input artifactusecase.ListArtifactsInput) artifactusecase.ListArtifactsInput {
	subject := s.services.Subject
	switch strings.TrimSpace(subject.ScopeType) {
	case domaintenant.ScopeProject:
		input.ProjectID = strings.TrimSpace(subject.ScopeID)
		input.EnvironmentID = ""
	case domaintenant.ScopeEnvironment:
		input.EnvironmentID = strings.TrimSpace(subject.ScopeID)
		input.ProjectID = ""
	}
	return input
}

func (s *Server) canReadArtifact(artifact domainartifact.Artifact) bool {
	projectID := artifact.Metadata["projectId"]
	environmentID := artifact.Metadata["environmentId"]
	return s.ensureSubjectScope(artifact.ID, projectID, environmentID) == nil
}

func (s *Server) canReadArtifactReleaseRecord(record artifactusecase.ReleaseRecord) bool {
	projectID := record.Release.Metadata["projectId"]
	environmentID := firstNonEmpty(record.Release.Metadata["environmentId"], record.Release.EnvironmentID)
	return s.ensureSubjectScope(record.Release.ID, projectID, environmentID) == nil
}

func (s *Server) filterArtifactReleaseBindings(bindings []artifactusecase.ArtifactReleaseBinding) []artifactusecase.ArtifactReleaseBinding {
	if len(bindings) == 0 {
		return bindings
	}
	filtered := make([]artifactusecase.ArtifactReleaseBinding, 0, len(bindings))
	for _, binding := range bindings {
		if s.canReadArtifactReleaseBinding(binding) {
			filtered = append(filtered, binding)
		}
	}
	return filtered
}

func (s *Server) canReadArtifactReleaseBinding(binding artifactusecase.ArtifactReleaseBinding) bool {
	projectID := firstNonEmpty(binding.Binding.Metadata["projectId"], binding.Release.Metadata["projectId"])
	environmentID := firstNonEmpty(binding.Binding.Metadata["environmentId"], binding.Release.Metadata["environmentId"], binding.Release.EnvironmentID)
	return s.ensureSubjectScope(binding.Binding.ID, projectID, environmentID) == nil
}

func (s *Server) filterRunnersForSubject(runners []domainrunner.Runner) []domainrunner.Runner {
	if len(runners) == 0 {
		return runners
	}
	scopeType := strings.TrimSpace(s.services.Subject.ScopeType)
	scopeID := strings.TrimSpace(s.services.Subject.ScopeID)
	if scopeType == "" || scopeType == domaintenant.ScopeGlobal || scopeID == "" {
		return runners
	}
	filtered := make([]domainrunner.Runner, 0, len(runners))
	for _, runner := range runners {
		if s.canReadRunner(runner) {
			filtered = append(filtered, runner)
		}
	}
	return filtered
}

func (s *Server) canReadRunner(runner domainrunner.Runner) bool {
	scopeType := strings.TrimSpace(s.services.Subject.ScopeType)
	scopeID := strings.TrimSpace(s.services.Subject.ScopeID)
	if scopeType == "" || scopeType == domaintenant.ScopeGlobal || scopeID == "" {
		return true
	}
	switch scopeType {
	case domaintenant.ScopeOrg:
		return runner.Labels["orgId"] == scopeID
	case domaintenant.ScopeProject:
		return runner.Labels["projectId"] == scopeID
	case domaintenant.ScopeEnvironment:
		return runner.Labels["environmentId"] == scopeID
	default:
		return false
	}
}

func (s *Server) subjectScopeSummary() map[string]string {
	scopeType := strings.TrimSpace(s.services.Subject.ScopeType)
	scopeID := strings.TrimSpace(s.services.Subject.ScopeID)
	if scopeType == "" || scopeType == domaintenant.ScopeGlobal || scopeID == "" {
		return map[string]string{"type": domaintenant.ScopeGlobal}
	}
	return map[string]string{"type": scopeType, "id": scopeID}
}

func (s *Server) capabilityStatusWarnings() []string {
	scopeType := strings.TrimSpace(s.services.Subject.ScopeType)
	scopeID := strings.TrimSpace(s.services.Subject.ScopeID)
	if scopeType == "" || scopeType == domaintenant.ScopeGlobal || scopeID == "" {
		return nil
	}
	return []string{"capability status is global maturity metadata; tenant inventory is not included in this resource"}
}

func (s *Server) ensureSubjectScope(resource string, projectID string, environmentID string) error {
	subject := s.services.Subject
	scopeType := strings.TrimSpace(subject.ScopeType)
	scopeID := strings.TrimSpace(subject.ScopeID)
	if scopeType == "" || scopeType == domaintenant.ScopeGlobal || scopeID == "" {
		return nil
	}
	switch scopeType {
	case domaintenant.ScopeProject:
		if projectID != "" && projectID == scopeID {
			return nil
		}
	case domaintenant.ScopeEnvironment:
		if environmentID != "" && environmentID == scopeID {
			return nil
		}
	default:
		return OperationError{Code: "mcp_scope_denied", Message: "unsupported MCP subject scope for " + resource}
	}
	return OperationError{Code: "mcp_scope_denied", Message: "MCP subject scope does not allow access to " + resource}
}

func (s *Server) checkResourcePermission(ctx context.Context, uri string) error {
	if uri == "nivora://audit/search" || uri == "nivora://evidence/bundles" || strings.HasPrefix(uri, "nivora://evidence/bundles/") {
		return s.require(ctx, domainauth.PermissionAuditRead, "mcp.resource", uri)
	}
	return s.require(ctx, domainauth.PermissionProjectRead, "mcp.resource", uri)
}

func (s *Server) toolPermission(name string) string {
	switch name {
	case "nivora_search_audit", "nivora_get_evidence_bundle", "nivora_list_evidence_bundles":
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
		"nivora_get_runtime_recovery_status",
		"nivora_search_events",
		"nivora_search_logs",
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
		"nivora_list_security_findings",
		"nivora_get_policy_result_summary",
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

func (s *Server) requestContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeoutText := strings.TrimSpace(s.services.Config.MCP.RequestTimeout)
	if timeoutText == "" {
		return ctx, func() {}
	}
	timeout, err := time.ParseDuration(timeoutText)
	if err != nil || timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func (s *Server) capResponseText(text string) string {
	limit := s.services.Config.MCP.MaxResponseBytes
	if limit <= 0 || len([]byte(text)) <= limit {
		return text
	}
	return mustJSON(map[string]any{
		"truncated": true,
		"limit":     limit,
		"preview":   truncateUTF8Bytes(text, limit),
		"message":   "MCP response exceeded configured max_response_bytes",
	})
}

func truncateUTF8Bytes(value string, limit int) string {
	if limit <= 0 || len([]byte(value)) <= limit {
		return value
	}
	last := 0
	for idx := range value {
		if idx > limit {
			break
		}
		last = idx
	}
	if last == 0 {
		return ""
	}
	return value[:last]
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

func integerProperty(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
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

func mcpPageFromArgs(args map[string]any) (mcpPage, error) {
	limit, err := intArg(args, "limit")
	if err != nil {
		return mcpPage{}, err
	}
	offset, err := intArg(args, "offset")
	if err != nil {
		return mcpPage{}, err
	}
	return normalizeMCPPage(limit, offset)
}

func intArg(args map[string]any, key string) (int, error) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, nil
	}
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case float64:
		parsed := int(typed)
		if float64(parsed) != typed {
			return 0, OperationError{Code: "mcp_invalid_arguments", Message: key + " must be an integer"}
		}
		return parsed, nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, OperationError{Code: "mcp_invalid_arguments", Message: key + " must be an integer"}
		}
		return parsed, nil
	default:
		return 0, OperationError{Code: "mcp_invalid_arguments", Message: key + " must be an integer"}
	}
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
