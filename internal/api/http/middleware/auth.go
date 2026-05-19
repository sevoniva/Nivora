package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

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
		decision := service.Evaluate(authusecase.EvaluateInput{Subject: subject, Action: action, Resource: auth.Resource{Type: "http", ID: r.URL.Path}})
		if !decision.Allowed {
			service.RecordDenied(r.Context(), subject, action, auth.Resource{Type: "http", ID: r.URL.Path})
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
