package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainplugin "github.com/sevoniva/nivora/internal/domain/plugin"
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

func ValidatePlugin(registry *pluginusecase.Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var manifest domainplugin.Manifest
		if err := json.NewDecoder(r.Body).Decode(&manifest); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: err.Error()})
			return
		}
		result, err := registry.Validate(r.Context(), manifest)
		if err != nil {
			respondPluginResult(w, r, result, err)
			return
		}
		status := http.StatusOK
		if !result.Valid {
			status = http.StatusBadRequest
		}
		RespondJSON(w, status, result)
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
