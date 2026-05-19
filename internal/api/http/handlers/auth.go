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
		RespondJSON(w, http.StatusOK, authusecase.TokenInfo{Authenticated: subject.ID != "", Mode: subject.AuthMode, SubjectID: subject.ID})
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
