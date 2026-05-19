package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	"github.com/sevoniva/nivora/internal/infra/config"
	"github.com/sevoniva/nivora/internal/infra/telemetry"
	"github.com/sevoniva/nivora/internal/version"
)

func Health(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, dto.StatusResponse{Status: "ok"})
}

func Ready(w http.ResponseWriter, r *http.Request) {
	RespondJSON(w, http.StatusOK, dto.StatusResponse{Status: "ready"})
}

func Version(info version.Info) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, info)
	}
}

func SystemInfo(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, dto.SystemInfoResponse{
			App:         cfg.App.Name,
			Environment: cfg.Env,
			EventBus:    cfg.EventBus.Type,
			ObjectStore: cfg.ObjectStore.Type,
			RuntimeMode: "in_memory",
		})
	}
}

func SystemRuntime(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, systemRuntimeResponse(r, cfg))
	}
}

func SystemDiagnostics(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, dto.SystemDiagnosticsResponse{
			Runtime: systemRuntimeResponse(r, cfg),
			Metrics: telemetry.DefaultMetrics().Snapshot(),
			Checks: []dto.DiagnosticCheck{
				{Name: "http", Status: "ok", Message: "API router is accepting requests"},
				{Name: "runtime", Status: "ok", Message: "Phase 4.2 diagnostics use in-process runtime state"},
				{Name: "telemetry", Status: "ok", Message: "metrics endpoint is local and tracing is configuration-only"},
			},
		})
	}
}

func Metrics() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(telemetry.DefaultMetrics().Snapshot().PrometheusText()))
	}
}

func NotImplemented(resource string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondError(w, r, http.StatusNotImplemented, dto.ErrorResponse{
			Code:    "not_implemented",
			Message: resource + " API is not implemented in Phase 0",
			Path:    r.URL.Path,
		})
	}
}

func RespondError(w http.ResponseWriter, r *http.Request, status int, payload dto.ErrorResponse) {
	if payload.Path == "" {
		payload.Path = r.URL.Path
	}
	payload.RequestID = middleware.GetReqID(r.Context())
	RespondJSON(w, status, payload)
}

func RespondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func systemRuntimeResponse(r *http.Request, cfg config.Config) dto.SystemRuntimeResponse {
	return dto.SystemRuntimeResponse{
		App:           cfg.App.Name,
		Environment:   cfg.Env,
		RuntimeMode:   "in_memory",
		RequestID:     middleware.GetReqID(r.Context()),
		CorrelationID: apimiddleware.CorrelationID(r.Context()),
		TraceID:       apimiddleware.TraceID(r.Context()),
		Telemetry: dto.TelemetryStatus{
			Enabled:         cfg.Telemetry.Enabled,
			Endpoint:        cfg.Telemetry.Endpoint,
			MetricsEndpoint: "/metrics",
			Tracing:         "placeholder",
		},
	}
}
