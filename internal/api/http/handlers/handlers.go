package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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

func ReadyWithConfig(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := dependencyChecks(cfg)
		status := "ready"
		code := http.StatusOK
		if hasCriticalFailure(checks) {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}
		RespondJSON(w, code, dto.StatusResponse{Status: status, Checks: checks})
	}
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
		runtime := systemRuntimeResponse(r, cfg)
		RespondJSON(w, http.StatusOK, dto.SystemDiagnosticsResponse{
			Runtime:       runtime,
			Metrics:       telemetry.DefaultMetrics().Snapshot(),
			Checks:        dependencyChecks(cfg),
			RequestID:     runtime.RequestID,
			CorrelationID: runtime.CorrelationID,
			TraceID:       runtime.TraceID,
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
		RuntimeMode:   runtimeMode(cfg),
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

func runtimeMode(cfg config.Config) string {
	if cfg.Database.RuntimeStore == "postgres" {
		return "postgres"
	}
	return "in_memory"
}

func dependencyChecks(cfg config.Config) []dto.DiagnosticCheck {
	now := time.Now().UTC()
	checks := []dto.DiagnosticCheck{
		{
			Name:          "http",
			Component:     "server",
			Status:        "ok",
			Critical:      true,
			Message:       "API router is accepting requests",
			RecoveryHint:  "If this fails, restart the server process or inspect load balancer routing.",
			Documentation: "docs/operations/ha-disaster-recovery.md#server-restart",
			CheckedAt:     now,
		},
		{
			Name:          "database",
			Component:     "postgres",
			Status:        "ok",
			Critical:      true,
			Message:       "database runtime store is configured",
			RecoveryHint:  "Run database backup/restore checks and confirm migrations before restarting workers.",
			Documentation: "docs/operations/backup-restore.md#database",
			CheckedAt:     now,
		},
		{
			Name:          "object_store",
			Component:     "object-store",
			Status:        "ok",
			Critical:      false,
			Message:       "object store configuration is present",
			RecoveryHint:  "Restore object store snapshots before replaying runs that reference manifest/log artifacts.",
			Documentation: "docs/operations/backup-restore.md#object-store",
			CheckedAt:     now,
		},
		{
			Name:          "event_bus",
			Component:     "events",
			Status:        "ok",
			Critical:      false,
			Message:       "event bus configuration is present",
			RecoveryHint:  "If publish fails, preserve pending outbox records and retry publication after recovery.",
			Documentation: "docs/operations/ha-disaster-recovery.md#event-publish-failure",
			CheckedAt:     now,
		},
		{
			Name:          "event_outbox",
			Component:     "runtime",
			Status:        "ok",
			Critical:      false,
			Message:       "event outbox recovery endpoint is available",
			RecoveryHint:  "Use POST /api/v1/system/runtime/reconcile after dependency recovery.",
			Documentation: "docs/operations/runtime-recovery.md",
			CheckedAt:     now,
		},
		{
			Name:          "runner_reconnect",
			Component:     "runner",
			Status:        "ok",
			Critical:      false,
			Message:       "runner heartbeat and offline detection endpoints are available",
			RecoveryHint:  "Runners should reconnect with their runner token; mark stale runners offline if needed.",
			Documentation: "docs/operations/runner-fleet.md",
			CheckedAt:     now,
		},
	}
	if cfg.Database.RuntimeStore == "memory" {
		checks[1].Status = "warning"
		checks[1].Message = "in-memory runtime store is configured; state is not recoverable after process restart"
		checks[1].RecoveryHint = "Use database.runtime_store=postgres for production-direction recovery testing."
	}
	if cfg.Database.RuntimeStore == "postgres" && strings.TrimSpace(cfg.Database.URL) == "" {
		checks[1].Status = "degraded"
		checks[1].Message = "postgres runtime store is selected but database.url is empty"
	}
	if strings.TrimSpace(cfg.ObjectStore.Type) == "" {
		checks[2].Status = "degraded"
		checks[2].Message = "object_store.type is empty"
	}
	if cfg.ObjectStore.Type == "local" && strings.TrimSpace(cfg.ObjectStore.Path) == "" {
		checks[2].Status = "degraded"
		checks[2].Message = "local object store requires object_store.path"
	}
	if strings.TrimSpace(cfg.EventBus.Type) == "" {
		checks[3].Status = "degraded"
		checks[3].Message = "event_bus.type is empty"
	}
	if cfg.EventBus.Type == "memory" {
		checks[3].Status = "warning"
		checks[3].Message = "in-memory event bus is configured; durable external publication remains future work"
	}
	return checks
}

func hasCriticalFailure(checks []dto.DiagnosticCheck) bool {
	for _, check := range checks {
		if check.Critical && check.Status == "degraded" {
			return true
		}
	}
	return false
}
