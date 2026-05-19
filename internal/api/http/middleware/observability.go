package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const (
	correlationIDKey contextKey = "correlation_id"
	traceIDKey       contextKey = "trace_id"
)

const (
	HeaderRequestID     = "X-Request-Id"
	HeaderCorrelationID = "X-Correlation-Id"
	HeaderTraceID       = "X-Trace-Id"
)

func RequestContext() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := chimiddleware.GetReqID(r.Context())
			correlationID := strings.TrimSpace(r.Header.Get(HeaderCorrelationID))
			if correlationID == "" {
				correlationID = requestID
			}
			traceID := traceIDFromHeader(r.Header.Get("traceparent"))
			if traceID == "" {
				traceID = strings.TrimSpace(r.Header.Get(HeaderTraceID))
			}

			ctx := context.WithValue(r.Context(), correlationIDKey, correlationID)
			ctx = context.WithValue(ctx, traceIDKey, traceID)

			if requestID != "" {
				w.Header().Set(HeaderRequestID, requestID)
			}
			if correlationID != "" {
				w.Header().Set(HeaderCorrelationID, correlationID)
			}
			if traceID != "" {
				w.Header().Set(HeaderTraceID, traceID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func StructuredAccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			attrs := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", chimiddleware.GetReqID(r.Context()),
				"correlation_id", CorrelationID(r.Context()),
				"trace_id", TraceID(r.Context()),
			}
			attrs = appendRouteFields(attrs, r)
			logger.InfoContext(r.Context(), "http request completed", attrs...)
		})
	}
}

func CorrelationID(ctx context.Context) string {
	if value, ok := ctx.Value(correlationIDKey).(string); ok {
		return value
	}
	return ""
}

func TraceID(ctx context.Context) string {
	if value, ok := ctx.Value(traceIDKey).(string); ok {
		return value
	}
	return ""
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func traceIDFromHeader(value string) string {
	parts := strings.Split(value, "-")
	if len(parts) < 4 {
		return ""
	}
	traceID := parts[1]
	if len(traceID) != 32 {
		return ""
	}
	return traceID
}

func appendRouteFields(attrs []any, r *http.Request) []any {
	path := r.URL.Path
	id := chi.URLParam(r, "id")
	fields := map[string]string{}
	if executionID := chi.URLParam(r, "execution_id"); executionID != "" {
		fields["release_execution_id"] = executionID
	} else if strings.Contains(path, "/releases/executions/") && id != "" {
		fields["release_execution_id"] = id
	} else if strings.Contains(path, "/pipeline-runs/") && id != "" {
		fields["run_id"] = id
	} else if strings.Contains(path, "/deployments/") && id != "" {
		fields["deployment_id"] = id
	} else if strings.Contains(path, "/jobs/") && id != "" {
		fields["job_id"] = id
	} else if strings.Contains(path, "/runners/") && id != "" {
		fields["runner_id"] = id
	} else if strings.Contains(path, "/releases/") && id != "" {
		fields["release_id"] = id
	}
	for key, value := range fields {
		if value != "" {
			attrs = append(attrs, key, value)
		}
	}
	return attrs
}
