package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	"github.com/sevoniva/nivora/internal/domain/auth"
	"github.com/sevoniva/nivora/internal/infra/config"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

type subjectKey struct{}

func Authenticate(cfg config.AuthConfig, service *authusecase.Service, writeError func(http.ResponseWriter, *http.Request, int, dto.ErrorResponse)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if service == nil {
				next.ServeHTTP(w, r)
				return
			}
			mode := cfg.Mode
			if !cfg.Enabled {
				mode = "disabled"
			}
			if isRunnerProtocolRequest(r) && runnerProtocolToken(r) != "" {
				runnerID := chi.URLParam(r, "id")
				subject := auth.Subject{ID: "runner:" + runnerID, Username: runnerID, DisplayName: runnerID, AuthMode: "runner_token", ScopeType: "runner", ScopeID: runnerID}
				next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), subjectKey{}, subject)))
				return
			}
			subject, err := service.Authenticate(r.Context(), authusecase.AuthenticateInput{
				Mode:         mode,
				DevUser:      cfg.DevUser,
				Token:        bearerToken(r.Header.Get("Authorization")),
				StaticToken:  os.Getenv(cfg.StaticTokenEnv),
				OIDCIssuer:   firstNonEmpty(cfg.OIDC.Issuer, cfg.Issuer),
				OIDCAudience: cfg.OIDC.ClientID,
			})
			if err != nil {
				writeError(w, r, http.StatusUnauthorized, dto.ErrorResponse{Code: "unauthorized", Message: "authentication required"})
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), subjectKey{}, subject)))
		})
	}
}

func isRunnerProtocolRequest(r *http.Request) bool {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/v1/runners/") {
		return false
	}
	if strings.Contains(path, "/heartbeat") || strings.Contains(path, "/jobs/") {
		return true
	}
	return false
}

func runnerProtocolToken(r *http.Request) string {
	if token := r.Header.Get("X-Nivora-Runner-Token"); token != "" {
		return token
	}
	return bearerToken(r.Header.Get("Authorization"))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func RequirePermission(service *authusecase.Service, action string, writeError func(http.ResponseWriter, *http.Request, int, dto.ErrorResponse), next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject := Subject(r.Context())
		resource := scopedResource(r)
		decision := service.Evaluate(authusecase.EvaluateInput{Subject: subject, Action: action, Resource: resource})
		if !decision.Allowed {
			service.RecordDenied(r.Context(), subject, action, resource)
			writeError(w, r, http.StatusForbidden, dto.ErrorResponse{Code: "forbidden", Message: decision.Reason})
			return
		}
		next(w, r)
	}
}

func Subject(ctx context.Context) auth.Subject {
	subject, _ := ctx.Value(subjectKey{}).(auth.Subject)
	return subject
}

func scopedResource(r *http.Request) auth.Resource {
	resource := auth.Resource{Type: "http", ID: r.URL.Path}
	query := r.URL.Query()
	if scopeType := query.Get("scopeType"); scopeType != "" {
		resource.ScopeType = scopeType
		resource.ScopeID = query.Get("scopeId")
		return resource
	}
	if projectID := query.Get("projectId"); projectID != "" {
		resource.ScopeType = "project"
		resource.ScopeID = projectID
		return resource
	}
	if environmentID := query.Get("environmentId"); environmentID != "" {
		resource.ScopeType = "environment"
		resource.ScopeID = environmentID
		return resource
	}
	if projectID := chi.URLParam(r, "project_id"); projectID != "" {
		resource.ScopeType = "project"
		resource.ScopeID = projectID
		return resource
	}
	if projectID := chi.URLParam(r, "id"); strings.HasPrefix(r.URL.Path, "/api/v1/projects/") && projectID != "" {
		resource.ScopeType = "project"
		resource.ScopeID = projectID
		return resource
	}
	if environmentID := chi.URLParam(r, "id"); strings.HasPrefix(r.URL.Path, "/api/v1/environments/") && environmentID != "" {
		resource.ScopeType = "environment"
		resource.ScopeID = environmentID
		return resource
	}
	return resource
}

func bearerToken(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
		return parts[1]
	}
	return ""
}
