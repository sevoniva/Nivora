package dto

import "time"

type StatusResponse struct {
	Status string            `json:"status"`
	Checks []DiagnosticCheck `json:"checks,omitempty"`
}

type ErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Path      string `json:"path,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type SystemInfoResponse struct {
	App         string `json:"app"`
	Environment string `json:"environment"`
	EventBus    string `json:"event_bus"`
	ObjectStore string `json:"object_store"`
	RuntimeMode string `json:"runtime_mode"`
}

type SystemRuntimeResponse struct {
	App           string          `json:"app"`
	Environment   string          `json:"environment"`
	RuntimeMode   string          `json:"runtime_mode"`
	Telemetry     TelemetryStatus `json:"telemetry"`
	RequestID     string          `json:"request_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	TraceID       string          `json:"trace_id,omitempty"`
}

type TelemetryStatus struct {
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint,omitempty"`
	MetricsEndpoint string `json:"metrics_endpoint"`
	Tracing         string `json:"tracing"`
}

type SystemDiagnosticsResponse struct {
	Runtime       SystemRuntimeResponse `json:"runtime"`
	Metrics       any                   `json:"metrics"`
	Checks        []DiagnosticCheck     `json:"checks"`
	RequestID     string                `json:"request_id,omitempty"`
	CorrelationID string                `json:"correlation_id,omitempty"`
	TraceID       string                `json:"trace_id,omitempty"`
}

type DiagnosticCheck struct {
	Name          string    `json:"name"`
	Component     string    `json:"component,omitempty"`
	Status        string    `json:"status"`
	Critical      bool      `json:"critical"`
	Message       string    `json:"message,omitempty"`
	RecoveryHint  string    `json:"recovery_hint,omitempty"`
	Documentation string    `json:"documentation,omitempty"`
	CheckedAt     time.Time `json:"checked_at,omitempty"`
}
