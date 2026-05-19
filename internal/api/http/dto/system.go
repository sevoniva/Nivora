package dto

type StatusResponse struct {
	Status string `json:"status"`
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
	Runtime SystemRuntimeResponse `json:"runtime"`
	Metrics any                   `json:"metrics"`
	Checks  []DiagnosticCheck     `json:"checks"`
}

type DiagnosticCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}
