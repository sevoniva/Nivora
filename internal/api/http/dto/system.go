package dto

type StatusResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type SystemInfoResponse struct {
	App         string `json:"app"`
	Environment string `json:"environment"`
	EventBus    string `json:"event_bus"`
	ObjectStore string `json:"object_store"`
}
