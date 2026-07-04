package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	apimcp "github.com/sevoniva/nivora/internal/api/mcp"
	"github.com/sevoniva/nivora/internal/infra/config"
)

func RemoteMCPJSONRPC(cfg config.Config, server *apimcp.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if server == nil || !cfg.MCP.Enabled || cfg.MCP.Mode != "http" {
			RespondError(w, r, http.StatusNotFound, dto.ErrorResponse{Code: "mcp_remote_disabled", Message: "remote MCP JSON-RPC is not enabled"})
			return
		}
		if r.Header.Get("X-Nivora-Runner-Token") != "" || strings.HasPrefix(bearerTokenValue(r.Header.Get("Authorization")), "nvr_runner_") {
			RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "mcp_runner_token_denied", Message: "runner tokens cannot use MCP"})
			return
		}
		subject := apimiddleware.Subject(r.Context())
		if subject.ID == "" {
			RespondError(w, r, http.StatusUnauthorized, dto.ErrorResponse{Code: "unauthorized", Message: "authentication required"})
			return
		}
		if subject.AuthMode == "runner_token" || strings.HasPrefix(subject.ID, "runner:") {
			RespondError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "mcp_runner_token_denied", Message: "runner tokens cannot use MCP"})
			return
		}
		if subject.AuthMode == "dev" || subject.AuthMode == "disabled" || subject.AuthMode == "mcp-local" {
			RespondError(w, r, http.StatusUnauthorized, dto.ErrorResponse{Code: "mcp_bearer_required", Message: "remote MCP requires bearer, service-account, or OIDC authentication"})
			return
		}

		limit := int64(cfg.MCP.MaxRequestBytes)
		reader := r.Body
		if limit > 0 {
			reader = io.NopCloser(io.LimitReader(r.Body, limit+1))
		}
		body, err := io.ReadAll(reader)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "read MCP request body"})
			return
		}
		response := server.WithSubject(subject).HandleJSONRPC(r.Context(), body)
		RespondJSON(w, http.StatusOK, response)
	}
}

func bearerTokenValue(header string) string {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return ""
}
