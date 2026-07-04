package handlers

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainevent "github.com/sevoniva/nivora/internal/domain/event"
	"github.com/sevoniva/nivora/internal/infra/crypto"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

type AggregateTimelineItem struct {
	ID              string         `json:"id,omitempty"`
	Kind            string         `json:"kind"`
	Type            string         `json:"type"`
	Time            time.Time      `json:"time"`
	Subject         string         `json:"subject,omitempty"`
	Source          string         `json:"source,omitempty"`
	Status          string         `json:"status,omitempty"`
	Message         string         `json:"message,omitempty"`
	PipelineRunID   string         `json:"pipelineRunId,omitempty"`
	DeploymentRunID string         `json:"deploymentRunId,omitempty"`
	StageRunID      string         `json:"stageRunId,omitempty"`
	JobRunID        string         `json:"jobRunId,omitempty"`
	StepRunID       string         `json:"stepRunId,omitempty"`
	Stream          string         `json:"stream,omitempty"`
	Sequence        int64          `json:"sequence,omitempty"`
	Data            map[string]any `json:"data,omitempty"`
}

func ListAuditLogs(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scopeType, scopeID := ConstrainScopeToRequest(r, r.URL.Query().Get("scopeType"), r.URL.Query().Get("scopeId"))
		result, err := service.SearchAudit(r.Context(), complianceusecase.AuditSearchInput{
			Subject:       r.URL.Query().Get("subject"),
			ActorID:       r.URL.Query().Get("actorId"),
			Action:        r.URL.Query().Get("action"),
			ScopeType:     scopeType,
			ScopeID:       scopeID,
			CorrelationID: r.URL.Query().Get("correlationId"),
		})
		if err != nil {
			respondComplianceResult(w, r, nil, err)
			return
		}
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			respondComplianceResult(w, r, nil, pageErr)
			return
		}
		if page.Enabled {
			RespondJSON(w, http.StatusOK, paginatedPayload(result.Items, page))
			return
		}
		RespondJSON(w, http.StatusOK, result)
	}
}

func ListTimeline(pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, releases *releaseorchestration.Service, artifacts *artifactusecase.Service, security *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := collectEvents(r.Context(), r, pipelines, deployments, releases, artifacts, security)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "timeline_list_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		logs, err := collectLogs(r.Context(), r, pipelines, deployments)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "timeline_list_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		events = filterEvents(events, r)
		logs = filterLogs(logs, r)
		timeline := buildAggregateTimeline(events, logs)
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error(), Path: r.URL.Path})
			return
		}
		if page.Enabled {
			RespondJSON(w, http.StatusOK, paginatedPayload(timeline, page))
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"timeline": timeline, "count": len(timeline)})
	}
}

func ListEvents(pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, releases *releaseorchestration.Service, artifacts *artifactusecase.Service, security *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := collectEvents(r.Context(), r, pipelines, deployments, releases, artifacts, security)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "events_list_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		events = filterEvents(events, r)
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error(), Path: r.URL.Path})
			return
		}
		if page.Enabled {
			RespondJSON(w, http.StatusOK, paginatedPayload(events, page))
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"events": events, "count": len(events)})
	}
}

func ListLogs(pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logs, err := collectLogs(r.Context(), r, pipelines, deployments)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "logs_list_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		logs = filterLogs(logs, r)
		page, pageErr := parsePagination(r)
		if pageErr != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error(), Path: r.URL.Path})
			return
		}
		if page.Enabled {
			RespondJSON(w, http.StatusOK, paginatedPayload(logs, page))
			return
		}
		RespondJSON(w, http.StatusOK, map[string]any{"logs": logs, "count": len(logs)})
	}
}

func buildAggregateTimeline(events []domainevent.Event, logs []domainevent.LogChunk) []AggregateTimelineItem {
	items := make([]AggregateTimelineItem, 0, len(events)+len(logs))
	for _, evt := range events {
		items = append(items, AggregateTimelineItem{
			ID:      evt.ID,
			Kind:    "event",
			Type:    evt.Type,
			Time:    evt.Time,
			Subject: evt.Subject,
			Source:  evt.Source,
			Status:  crypto.RedactString(anyString(evt.Data["status"])),
			Message: crypto.RedactString(anyString(evt.Data["message"])),
			Data:    redactAnyMap(evt.Data),
		})
	}
	for _, log := range logs {
		items = append(items, AggregateTimelineItem{
			ID:              log.ID,
			Kind:            "log",
			Type:            "log." + log.Stream,
			Time:            log.CreatedAt,
			Subject:         firstNonEmpty(log.StepRunID, log.JobRunID, log.DeploymentRunID, log.PipelineRunID),
			Message:         crypto.RedactString(log.Content),
			PipelineRunID:   log.PipelineRunID,
			DeploymentRunID: log.DeploymentRunID,
			StageRunID:      log.StageRunID,
			JobRunID:        log.JobRunID,
			StepRunID:       log.StepRunID,
			Stream:          log.Stream,
			Sequence:        log.Sequence,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Time.Equal(items[j].Time) {
			if items[i].Kind == items[j].Kind {
				if items[i].Sequence == items[j].Sequence {
					return items[i].ID < items[j].ID
				}
				return items[i].Sequence < items[j].Sequence
			}
			return items[i].Kind == "event"
		}
		return items[i].Time.Before(items[j].Time)
	})
	return items
}

func collectEvents(ctx context.Context, r *http.Request, pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, releases *releaseorchestration.Service, artifacts *artifactusecase.Service, security *securityusecase.Service) ([]domainevent.Event, error) {
	var events []domainevent.Event
	scopeType, scopeID := TenantScopeFilter(r)
	if pipelines != nil {
		records, err := pipelines.ListFiltered(ctx, scopeType, scopeID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	if deployments != nil {
		records, err := deployments.ListFiltered(ctx, scopeType, scopeID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	if releases != nil {
		records, err := releases.ListExecutions(ctx, "")
		if err != nil {
			return nil, err
		}
		records = filterReleaseExecutionsForRequest(r, records)
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	if artifacts != nil && scopeType == "" {
		records, err := artifacts.ListReleases(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	if security != nil && scopeType == "" {
		records, err := security.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Time.Before(events[j].Time) })
	return events, nil
}

func collectLogs(ctx context.Context, r *http.Request, pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service) ([]domainevent.LogChunk, error) {
	var logs []domainevent.LogChunk
	scopeType, scopeID := TenantScopeFilter(r)
	if pipelines != nil {
		records, err := pipelines.ListFiltered(ctx, scopeType, scopeID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			logs = append(logs, record.Logs...)
		}
	}
	if deployments != nil {
		records, err := deployments.ListFiltered(ctx, scopeType, scopeID)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			logs = append(logs, record.Logs...)
		}
	}
	sort.Slice(logs, func(i, j int) bool {
		if logs[i].CreatedAt.Equal(logs[j].CreatedAt) {
			return logs[i].Sequence < logs[j].Sequence
		}
		return logs[i].CreatedAt.Before(logs[j].CreatedAt)
	})
	return logs, nil
}

func filterEvents(events []domainevent.Event, r *http.Request) []domainevent.Event {
	query := r.URL.Query()
	eventType := query.Get("type")
	source := query.Get("source")
	subject := query.Get("subject")
	runID := query.Get("runId")
	pipelineRunID := query.Get("pipelineRunId")
	deploymentRunID := query.Get("deploymentRunId")
	releaseID := query.Get("releaseId")
	artifactID := query.Get("artifactId")
	securityScanID := query.Get("securityScanId")
	if eventType == "" && source == "" && subject == "" && runID == "" && pipelineRunID == "" && deploymentRunID == "" && releaseID == "" && artifactID == "" && securityScanID == "" {
		return events
	}
	filtered := make([]domainevent.Event, 0, len(events))
	for _, evt := range events {
		if eventType != "" && !containsFold(evt.Type, eventType) {
			continue
		}
		if source != "" && !containsFold(evt.Source, source) {
			continue
		}
		if subject != "" && !containsFold(evt.Subject, subject) {
			continue
		}
		if runID != "" && !eventMatchesIdentifier(evt, runID, "runId", "pipelineRunId", "deploymentRunId", "releaseExecutionId") {
			continue
		}
		if pipelineRunID != "" && !eventMatchesIdentifier(evt, pipelineRunID, "runId", "pipelineRunId") {
			continue
		}
		if deploymentRunID != "" && !eventMatchesIdentifier(evt, deploymentRunID, "runId", "deploymentRunId") {
			continue
		}
		if releaseID != "" && !eventMatchesIdentifier(evt, releaseID, "releaseId") {
			continue
		}
		if artifactID != "" && !eventMatchesIdentifier(evt, artifactID, "artifactId") {
			continue
		}
		if securityScanID != "" && !eventMatchesIdentifier(evt, securityScanID, "scanId", "securityScanId") {
			continue
		}
		filtered = append(filtered, evt)
	}
	return filtered
}

func eventMatchesIdentifier(evt domainevent.Event, value string, dataKeys ...string) bool {
	if value == "" {
		return true
	}
	if evt.Subject == value {
		return true
	}
	for _, key := range dataKeys {
		if dataValue, ok := evt.Data[key]; ok && anyString(dataValue) == value {
			return true
		}
	}
	return false
}

func filterLogs(logs []domainevent.LogChunk, r *http.Request) []domainevent.LogChunk {
	query := r.URL.Query()
	runID := query.Get("runId")
	pipelineRunID := query.Get("pipelineRunId")
	deploymentRunID := query.Get("deploymentRunId")
	stageRunID := query.Get("stageRunId")
	jobRunID := query.Get("jobRunId")
	stepRunID := query.Get("stepRunId")
	stream := query.Get("stream")
	contains := query.Get("contains")
	if runID == "" && pipelineRunID == "" && deploymentRunID == "" && stageRunID == "" && jobRunID == "" && stepRunID == "" && stream == "" && contains == "" {
		return logs
	}
	filtered := make([]domainevent.LogChunk, 0, len(logs))
	for _, log := range logs {
		if runID != "" && log.PipelineRunID != runID && log.DeploymentRunID != runID {
			continue
		}
		if pipelineRunID != "" && log.PipelineRunID != pipelineRunID {
			continue
		}
		if deploymentRunID != "" && log.DeploymentRunID != deploymentRunID {
			continue
		}
		if stageRunID != "" && log.StageRunID != stageRunID {
			continue
		}
		if jobRunID != "" && log.JobRunID != jobRunID {
			continue
		}
		if stepRunID != "" && log.StepRunID != stepRunID {
			continue
		}
		if stream != "" && log.Stream != stream {
			continue
		}
		if contains != "" && !containsFold(log.Content, contains) {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func containsFold(value string, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func redactAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		if crypto.IsSensitiveKey(key) {
			out[key] = "[REDACTED]"
			continue
		}
		switch typed := value.(type) {
		case string:
			out[key] = crypto.RedactString(typed)
		case map[string]any:
			out[key] = redactAnyMap(typed)
		case map[string]string:
			out[key] = crypto.RedactMap(typed)
		default:
			out[key] = typed
		}
	}
	return out
}
