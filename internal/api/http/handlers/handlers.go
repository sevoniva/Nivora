package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	"github.com/sevoniva/nivora/internal/infra/config"
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
