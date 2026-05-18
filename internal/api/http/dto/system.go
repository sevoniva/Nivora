package dto

type StatusResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type SystemInfoResponse struct {
	App         string `json:"app"`
	Environment string `json:"environment"`
	EventBus    string `json:"event_bus"`
	ObjectStore string `json:"object_store"`
}
