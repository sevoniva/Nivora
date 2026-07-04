package handlers

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	"github.com/sevoniva/nivora/internal/domain/audit"
	domainsecurity "github.com/sevoniva/nivora/internal/domain/security"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

type visualizationSurface struct {
	Group              string `json:"group"`
	Method             string `json:"method"`
	Path               string `json:"path"`
	Description        string `json:"description"`
	RequiredPermission string `json:"requiredPermission"`
	Status             string `json:"status"`
}

func ListVisualizationSurfaces() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		surfaces := []visualizationSurface{
			{Group: "pipeline", Method: http.MethodGet, Path: "/api/v1/visualization/pipeline-runs/{id}/dag", Description: "PipelineRun DAG graph model", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "pipeline", Method: http.MethodGet, Path: "/api/v1/visualization/pipeline-runs/{id}/timeline", Description: "PipelineRun timeline items", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "pipeline", Method: http.MethodGet, Path: "/api/v1/visualization/pipeline-runs/{id}/summary", Description: "PipelineRun summary badge and counts", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "deployment", Method: http.MethodGet, Path: "/api/v1/visualization/deployments/{id}/timeline", Description: "DeploymentRun timeline items", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "deployment", Method: http.MethodGet, Path: "/api/v1/visualization/deployments/{id}/resources", Description: "Deployment resource nodes", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "deployment", Method: http.MethodGet, Path: "/api/v1/visualization/deployments/{id}/diff", Description: "Deployment diff summary", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "deployment", Method: http.MethodGet, Path: "/api/v1/visualization/deployments/{id}/health", Description: "Deployment health summary", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "release", Method: http.MethodGet, Path: "/api/v1/visualization/releases/{id}/overview", Description: "Release plan and execution overview", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "release", Method: http.MethodGet, Path: "/api/v1/visualization/releases/executions/{id}/timeline", Description: "ReleaseExecution timeline items", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "release", Method: http.MethodGet, Path: "/api/v1/visualization/releases/executions/{id}/targets", Description: "ReleaseExecution target rows", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "environment", Method: http.MethodGet, Path: "/api/v1/visualization/environments/{id}/topology", Description: "Environment topology read model", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "runner", Method: http.MethodGet, Path: "/api/v1/visualization/runners/summary", Description: "Runner dashboard summary", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "security", Method: http.MethodGet, Path: "/api/v1/visualization/security/summary", Description: "Security dashboard summary", RequiredPermission: "project.read", Status: "foundation"},
			{Group: "audit", Method: http.MethodGet, Path: "/api/v1/visualization/audit/timeline", Description: "Aggregate audit timeline", RequiredPermission: "audit.read", Status: "foundation"},
		}
		RespondJSON(w, http.StatusOK, map[string]any{
			"surfaces": surfaces,
			"count":    len(surfaces),
			"warnings": []string{"visualization APIs are backend read models only; no production web console claim is implied"},
		})
	}
}

func GetVisualizationPipelineDAG(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		nodes, edges := pipelineDAG(record)
		RespondJSON(w, http.StatusOK, map[string]any{"nodes": nodes, "edges": edges})
	}
}

func GetVisualizationPipelineTimeline(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, timelineItemsFromPipeline(timeline))
	}
}

func GetVisualizationPipelineSummary(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := service.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, pipelineSummary(record))
	}
}

func GetVisualizationDeploymentTimeline(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, timelineItemsFromDeployment(timeline))
	}
}

func GetVisualizationDeploymentResources(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resources, err := service.Resources(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, resourceNodes(resources))
	}
}

func GetVisualizationDeploymentDiff(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		diff, err := service.Diff(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, diff)
	}
}

func GetVisualizationDeploymentHealth(service *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		health, err := service.Health(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{
			"status":  statusBadge(string(health.Status)),
			"summary": health,
		})
	}
}

func GetVisualizationReleaseOverview(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := chi.URLParam(r, "id")
		plan, err := service.GetPlan(r.Context(), releaseID)
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		executions, err := service.ListExecutions(r.Context(), releaseID)
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{
			"release":    plan.Release,
			"plan":       plan.Plan,
			"summary":    releaseSummary(plan, executions),
			"executions": executions,
		})
	}
}

func GetVisualizationReleaseExecutionTimeline(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeline, err := service.Timeline(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, timelineItemsFromRelease(timeline))
	}
}

func GetVisualizationReleaseExecutionTargets(service *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targets, err := service.Targets(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, targets)
	}
}

func GetVisualizationEnvironmentTopology(deployments *deploymentusecase.Service, releases *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		topology, err := environmentTopology(r.Context(), chi.URLParam(r, "id"), deployments, releases)
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, topology)
	}
}

func GetVisualizationRunnerSummary(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		runners, err := service.ListRunners(r.Context())
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		nodes := make([]dto.ResourceNode, 0, len(runners))
		counts := map[string]int{"total": len(runners)}
		for _, runner := range runners {
			counts[runner.Status]++
			nodes = append(nodes, dto.ResourceNode{
				ID:     runner.ID,
				Type:   "runner",
				Name:   runner.Name,
				Status: statusBadge(runner.Status),
				Metadata: map[string]any{
					"executors":        runner.Executors,
					"labels":           runner.Labels,
					"lastHeartbeatAt":  runner.LastHeartbeatAt,
					"createdAt":        runner.CreatedAt,
					"runtimeComponent": "runner",
				},
			})
		}
		RespondJSON(w, http.StatusOK, dto.RunnerSummary{
			DashboardSummary: dto.DashboardSummary{Title: "Runner summary", Counts: counts, UpdatedAt: time.Now()},
			Runners:          nodes,
		})
	}
}

func GetVisualizationSecuritySummary(service *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := service.List(r.Context())
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		counts := map[string]int{"scans": len(records)}
		findings := map[string]int{}
		status := "Unknown"
		var updated time.Time
		for _, record := range records {
			counts[string(record.Scan.Status)]++
			if record.Scan.CreatedAt.After(updated) {
				updated = record.Scan.CreatedAt
			}
			summary := record.Scan.Summary
			findings["critical"] += summary.Critical
			findings["high"] += summary.High
			findings["medium"] += summary.Medium
			findings["low"] += summary.Low
		}
		if findings["critical"] > 0 {
			status = "Critical"
		} else if findings["high"] > 0 {
			status = "Warning"
		} else if len(records) > 0 {
			status = "Healthy"
		}
		RespondJSON(w, http.StatusOK, dto.SecuritySummary{
			DashboardSummary: dto.DashboardSummary{Title: "Security summary", Status: statusBadge(status), Counts: counts, UpdatedAt: updated},
			Findings:         findings,
		})
	}
}

func GetVisualizationAuditTimeline(pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, releases *releaseorchestration.Service, security *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := auditTimeline(r.Context(), pipelines, deployments, releases, security)
		if err != nil {
			respondVisualizationError(w, r, err)
			return
		}
		RespondJSON(w, http.StatusOK, items)
	}
}

func pipelineDAG(record pipelineusecase.RunRecord) ([]dto.GraphNode, []dto.GraphEdge) {
	nodes := []dto.GraphNode{{
		ID:     record.Run.ID,
		Type:   "pipelineRun",
		Label:  record.Pipeline.Name,
		Status: statusBadge(string(record.Run.Status)),
	}}
	edges := make([]dto.GraphEdge, 0)
	var previousStageID string
	for stageIndex, stage := range record.Stages {
		stageID := stage.Stage.ID
		nodes = append(nodes, dto.GraphNode{ID: stageID, Type: "stageRun", Label: stage.Stage.Name, Status: statusBadge(string(stage.Stage.Status))})
		edges = append(edges, dto.GraphEdge{ID: "edge-" + record.Run.ID + "-" + stageID, Source: record.Run.ID, Target: stageID, Label: "contains"})
		if previousStageID != "" {
			edges = append(edges, dto.GraphEdge{ID: "edge-" + previousStageID + "-" + stageID, Source: previousStageID, Target: stageID, Label: "next"})
		}
		previousStageID = stageID
		for jobIndex, job := range stage.Jobs {
			jobID := job.Job.ID
			nodes = append(nodes, dto.GraphNode{
				ID:     jobID,
				Type:   "jobRun",
				Label:  job.Job.Name,
				Status: statusBadge(string(job.Job.Status)),
				Metadata: map[string]any{
					"runnerId": job.Job.RunnerID,
					"attempt":  job.Job.Attempt,
					"order":    jobIndex,
				},
			})
			edges = append(edges, dto.GraphEdge{ID: "edge-" + stageID + "-" + jobID, Source: stageID, Target: jobID, Label: "contains"})
			for stepIndex, step := range job.Steps {
				stepID := step.ID
				nodes = append(nodes, dto.GraphNode{
					ID:     stepID,
					Type:   "stepRun",
					Label:  step.Name,
					Status: statusBadge(string(step.Status)),
					Metadata: map[string]any{
						"order":      stepIndex,
						"stageOrder": stageIndex,
					},
				})
				edges = append(edges, dto.GraphEdge{ID: "edge-" + jobID + "-" + stepID, Source: jobID, Target: stepID, Label: "contains"})
			}
		}
	}
	return nodes, edges
}

func pipelineSummary(record pipelineusecase.RunRecord) dto.DashboardSummary {
	counts := map[string]int{"stages": len(record.Stages), "logs": len(record.Logs), "events": len(record.Events)}
	for _, stage := range record.Stages {
		counts["jobs"] += len(stage.Jobs)
		for _, job := range stage.Jobs {
			counts["steps"] += len(job.Steps)
		}
	}
	return dto.DashboardSummary{
		ID:     record.Run.ID,
		Title:  "PipelineRun " + record.Run.ID,
		Status: statusBadge(string(record.Run.Status)),
		Counts: counts,
		Metadata: map[string]string{
			"pipeline": record.Pipeline.Name,
			"reason":   record.Run.FailureReason,
		},
		UpdatedAt: record.Run.UpdatedAt,
	}
}

func releaseSummary(plan releaseorchestration.PlanRecord, executions []releaseorchestration.ExecutionRecord) dto.DashboardSummary {
	counts := map[string]int{
		"targets":     len(plan.Plan.Targets),
		"plans":       len(plan.Plan.DeploymentPlans),
		"executions":  len(executions),
		"artifacts":   len(plan.Plan.ArtifactSummary),
		"policyGates": len(plan.Plan.PolicyResults),
	}
	status := "Created"
	if len(executions) > 0 {
		status = string(executions[len(executions)-1].Execution.Status)
	}
	return dto.DashboardSummary{
		ID:        plan.Plan.ReleaseID,
		Title:     "Release " + plan.Release.Name,
		Status:    statusBadge(status),
		Counts:    counts,
		UpdatedAt: plan.Plan.CreatedAt,
	}
}

func environmentTopology(ctx context.Context, environmentID string, deployments *deploymentusecase.Service, releases *releaseorchestration.Service) (dto.EnvironmentTopology, error) {
	_ = releases
	select {
	case <-ctx.Done():
		return dto.EnvironmentTopology{}, ctx.Err()
	default:
	}
	records, err := deployments.List(ctx)
	if err != nil {
		return dto.EnvironmentTopology{}, err
	}
	topology := dto.EnvironmentTopology{EnvironmentID: environmentID, HealthSummary: dto.DashboardSummary{Title: "Environment health", Counts: map[string]int{}, Status: statusBadge("Unknown")}}
	appSeen := map[string]struct{}{}
	targetSeen := map[string]struct{}{}
	for _, record := range records {
		if record.Environment.ID != environmentID && record.Environment.Name != environmentID {
			continue
		}
		if _, ok := appSeen[record.Run.ApplicationID]; record.Run.ApplicationID != "" && !ok {
			appSeen[record.Run.ApplicationID] = struct{}{}
			topology.Applications = append(topology.Applications, dto.ResourceNode{ID: record.Run.ApplicationID, Type: "application", Name: record.Run.ApplicationID})
		}
		targetID := record.Target.ID
		if targetID == "" {
			targetID = record.Run.ReleaseTargetID
		}
		if _, ok := targetSeen[targetID]; targetID != "" && !ok {
			targetSeen[targetID] = struct{}{}
			topology.Targets = append(topology.Targets, dto.ResourceNode{ID: targetID, Type: string(record.Run.TargetType), Name: record.Target.Name, Status: statusBadge(string(record.Run.Status))})
		}
		topology.LatestDeployments = append(topology.LatestDeployments, dto.ResourceNode{
			ID:     record.Run.ID,
			Type:   "deploymentRun",
			Name:   record.Definition.Metadata.Name,
			Status: statusBadge(string(record.Run.Status)),
		})
		topology.Resources = append(topology.Resources, resourceNodes(record.Inventory.Desired)...)
		topology.HealthSummary.Counts[string(record.Health.Status)]++
		if record.Health.Status != "" {
			topology.HealthSummary.Status = statusBadge(string(record.Health.Status))
		}
		topology.HealthSummary.UpdatedAt = record.Run.UpdatedAt
	}
	return topology, nil
}

func resourceNodes(resources []deploymentusecase.ManifestResourceSummary) []dto.ResourceNode {
	nodes := make([]dto.ResourceNode, 0, len(resources))
	for _, resource := range resources {
		id := resource.Kind + "/" + resource.Namespace + "/" + resource.Name
		nodes = append(nodes, dto.ResourceNode{
			ID:        id,
			Type:      resource.Kind,
			Name:      resource.Name,
			Namespace: resource.Namespace,
			Status:    statusBadge(resource.Status),
			Health:    statusBadge(string(resource.Health)),
			Metadata: map[string]any{
				"apiVersion":    resource.APIVersion,
				"group":         resource.Group,
				"version":       resource.Version,
				"sourceFile":    resource.SourceFile,
				"documentIndex": resource.Index,
				"labels":        resource.Labels,
				"annotations":   resource.Annotations,
			},
		})
	}
	return nodes
}

func auditTimeline(ctx context.Context, pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, releases *releaseorchestration.Service, security *securityusecase.Service) ([]dto.TimelineItem, error) {
	items := []dto.TimelineItem{}
	pipelineRuns, err := pipelines.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, record := range pipelineRuns {
		items = append(items, auditItems(record.Audits, "pipelineRun")...)
	}
	deploymentRuns, err := deployments.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, record := range deploymentRuns {
		items = append(items, auditItems(record.Audits, "deploymentRun")...)
	}
	executions, err := releases.ListExecutions(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, record := range executions {
		items = append(items, auditItems(record.Audits, "releaseExecution")...)
	}
	scans, err := security.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, record := range scans {
		items = append(items, auditItems(record.Audits, "securityScan")...)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })
	return items, nil
}

func auditItems(entries []audit.AuditLog, itemType string) []dto.TimelineItem {
	items := make([]dto.TimelineItem, 0, len(entries))
	for i, entry := range entries {
		items = append(items, dto.TimelineItem{
			ID:      entry.ID,
			Type:    itemType + ".audit",
			Time:    entry.CreatedAt,
			Subject: entry.Subject,
			Message: entry.Action,
			Data: map[string]any{
				"actorId": entry.ActorID,
				"index":   strconv.Itoa(i),
			},
		})
	}
	return items
}

func timelineItemsFromPipeline(entries []pipelineusecase.TimelineEntry) []dto.TimelineItem {
	items := make([]dto.TimelineItem, 0, len(entries))
	for i, entry := range entries {
		items = append(items, dto.TimelineItem{ID: "pipeline-" + strconv.Itoa(i), Type: entry.Type, Time: entry.Time, Subject: entry.Subject, Status: statusBadge(entry.Status), Message: entry.Message})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })
	return items
}

func timelineItemsFromDeployment(entries []deploymentusecase.TimelineEntry) []dto.TimelineItem {
	items := make([]dto.TimelineItem, 0, len(entries))
	for i, entry := range entries {
		items = append(items, dto.TimelineItem{ID: "deployment-" + strconv.Itoa(i), Type: entry.Type, Time: entry.Time, Subject: entry.Subject, Status: statusBadge(entry.Status), Message: entry.Message})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })
	return items
}

func timelineItemsFromRelease(entries []releaseorchestration.TimelineEntry) []dto.TimelineItem {
	items := make([]dto.TimelineItem, 0, len(entries))
	for i, entry := range entries {
		items = append(items, dto.TimelineItem{ID: "release-" + strconv.Itoa(i), Type: entry.Type, Time: entry.Time, Subject: entry.Subject, Status: statusBadge(entry.Status), Message: entry.Message})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Time.Before(items[j].Time) })
	return items
}

func statusBadge(value string) dto.StatusBadge {
	tone := "neutral"
	switch value {
	case "Succeeded", "Healthy", "Synced", "online", string(domainsecurity.GateAllow):
		tone = "success"
	case "Failed", "Canceled", "Timeout", "Degraded", "Critical", string(domainsecurity.GateDeny):
		tone = "danger"
	case "Running", "Queued", "Pending", "Progressing", "WaitingApproval", "Retrying":
		tone = "progress"
	case "Warning", "Unknown", "Unsupported", string(domainsecurity.GateWarn), string(domainsecurity.GateRequireApproval):
		tone = "warning"
	}
	return dto.StatusBadge{Value: value, Tone: tone}
}

func respondVisualizationError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	code := "visualization_error"
	if errors.Is(err, pipelineusecase.ErrRunNotFound) {
		status = http.StatusNotFound
		code = "pipeline_run_not_found"
	}
	if errors.Is(err, deploymentusecase.ErrRunNotFound) {
		status = http.StatusNotFound
		code = "deployment_run_not_found"
	}
	if errors.Is(err, releaseorchestration.ErrPlanNotFound) {
		status = http.StatusNotFound
		code = "release_plan_not_found"
	}
	if errors.Is(err, releaseorchestration.ErrExecutionNotFound) {
		status = http.StatusNotFound
		code = "release_execution_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
