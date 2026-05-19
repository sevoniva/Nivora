package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	apimiddleware "github.com/sevoniva/nivora/internal/api/http/middleware"
	authusecase "github.com/sevoniva/nivora/internal/usecase/auth"
)

func WhoAmI() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, authusecase.WhoAmI{Subject: apimiddleware.Subject(r.Context())})
	}
}

func AuthPermissions(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, service.Permissions())
	}
}

func AuthTokenInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subject := apimiddleware.Subject(r.Context())
		RespondJSON(w, http.StatusOK, authusecase.TokenInfo{Authenticated: subject.ID != "", Mode: subject.AuthMode, SubjectID: subject.ID, TokenID: subject.TokenID})
	}
}

func ListUsers(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := service.ListUsers(r.Context())
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "auth_error", Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, users)
	}
}

func ListRoles(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, service.Roles())
	}
}

func ListPermissions(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		RespondJSON(w, http.StatusOK, service.Permissions())
	}
}

func ListOrgMembers(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memberships, err := service.ListMemberships(r.Context(), "org", chi.URLParam(r, "id"))
		respondAuthResult(w, r, memberships, err)
	}
}

func AddOrgMember(service *authusecase.Service) http.HandlerFunc {
	return addMember(service, "org")
}

func ListProjectMembers(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memberships, err := service.ListMemberships(r.Context(), "project", chi.URLParam(r, "id"))
		respondAuthResult(w, r, memberships, err)
	}
}

func AddProjectMember(service *authusecase.Service) http.HandlerFunc {
	return addMember(service, "project")
}

func ListEnvironmentMembers(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		memberships, err := service.ListMemberships(r.Context(), "environment", chi.URLParam(r, "id"))
		respondAuthResult(w, r, memberships, err)
	}
}

func AddEnvironmentMember(service *authusecase.Service) http.HandlerFunc {
	return addMember(service, "environment")
}

func ListServiceAccounts(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := service.ListServiceAccounts(r.Context(), r.URL.Query().Get("scopeType"), r.URL.Query().Get("scopeId"))
		respondAuthResult(w, r, accounts, err)
	}
}

func CreateServiceAccount(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input authusecase.ServiceAccountInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a service account request"})
			return
		}
		account, err := service.CreateServiceAccount(r.Context(), input, apimiddleware.Subject(r.Context()).ID)
		respondAuthResult(w, r, account, err)
	}
}

func ListAPITokens(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokens, err := service.ListAPITokens(r.Context(), r.URL.Query().Get("subjectId"))
		respondAuthResult(w, r, tokens, err)
	}
}

func CreateAPIToken(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input authusecase.APITokenInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an API token request"})
			return
		}
		result, err := service.CreateAPIToken(r.Context(), input, apimiddleware.Subject(r.Context()).ID)
		if err != nil {
			respondAuthResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusCreated, result)
	}
}

func RotateAPIToken(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.RotateAPIToken(r.Context(), chi.URLParam(r, "id"), apimiddleware.Subject(r.Context()).ID)
		if err != nil {
			respondAuthResult(w, r, nil, err)
			return
		}
		RespondJSON(w, http.StatusOK, result)
	}
}

func RevokeAPIToken(service *authusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metadata, err := service.RevokeAPIToken(r.Context(), chi.URLParam(r, "id"), apimiddleware.Subject(r.Context()).ID)
		respondAuthResult(w, r, metadata, err)
	}
}

func addMember(service *authusecase.Service, scopeType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input authusecase.MembershipInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a membership request"})
			return
		}
		input.ScopeType = scopeType
		input.ScopeID = chi.URLParam(r, "id")
		membership, err := service.CreateMembership(r.Context(), input, apimiddleware.Subject(r.Context()).ID)
		respondAuthResult(w, r, membership, err)
	}
}

func respondAuthResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err != nil {
		RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "auth_error", Message: err.Error()})
		return
	}
	RespondJSON(w, http.StatusOK, payload)
}
