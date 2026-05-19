package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	pluginusecase "github.com/sevoniva/nivora/internal/usecase/plugin"
)

func ListPlugins(registry *pluginusecase.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plugins, err := registry.List(r.Context())
		respondPluginResult(w, r, plugins, err)
	}
}

func GetPlugin(registry *pluginusecase.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		manifest, err := registry.Get(r.Context(), chi.URLParam(r, "name"))
		respondPluginResult(w, r, manifest, err)
	}
}

func GetPluginCapabilities(registry *pluginusecase.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		capabilities, err := registry.Capabilities(r.Context(), chi.URLParam(r, "name"))
		respondPluginResult(w, r, capabilities, err)
	}
}

func respondPluginResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusBadRequest
	code := "plugin_error"
	if errors.Is(err, pluginusecase.ErrPluginNotFound) {
		status = http.StatusNotFound
		code = "plugin_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
