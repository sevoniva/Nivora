package handlers

import (
	"context"
	"net/http"
	"sort"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainevent "github.com/sevoniva/nivora/internal/domain/event"
	artifactusecase "github.com/sevoniva/nivora/internal/usecase/artifact"
	complianceusecase "github.com/sevoniva/nivora/internal/usecase/compliance"
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
	securityusecase "github.com/sevoniva/nivora/internal/usecase/security"
)

func ListAuditLogs(service *complianceusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.SearchAudit(r.Context(), complianceusecase.AuditSearchInput{
			Subject:       r.URL.Query().Get("subject"),
			ActorID:       r.URL.Query().Get("actorId"),
			Action:        r.URL.Query().Get("action"),
			ScopeType:     r.URL.Query().Get("scopeType"),
			ScopeID:       r.URL.Query().Get("scopeId"),
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

func ListEvents(pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, releases *releaseorchestration.Service, artifacts *artifactusecase.Service, security *securityusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events, err := collectEvents(r.Context(), pipelines, deployments, releases, artifacts, security)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "events_list_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
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
		logs, err := collectLogs(r.Context(), pipelines, deployments)
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "logs_list_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
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

func collectEvents(ctx context.Context, pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service, releases *releaseorchestration.Service, artifacts *artifactusecase.Service, security *securityusecase.Service) ([]domainevent.Event, error) {
	var events []domainevent.Event
	if pipelines != nil {
		records, err := pipelines.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	if deployments != nil {
		records, err := deployments.List(ctx)
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
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	if artifacts != nil {
		records, err := artifacts.ListReleases(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			events = append(events, record.Events...)
		}
	}
	if security != nil {
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

func collectLogs(ctx context.Context, pipelines *pipelineusecase.Service, deployments *deploymentusecase.Service) ([]domainevent.LogChunk, error) {
	var logs []domainevent.LogChunk
	if pipelines != nil {
		records, err := pipelines.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			logs = append(logs, record.Logs...)
		}
	}
	if deployments != nil {
		records, err := deployments.List(ctx)
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
