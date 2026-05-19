package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	portsecret "github.com/sevoniva/nivora/internal/ports/secret"
	credcase "github.com/sevoniva/nivora/internal/usecase/credential"
)

func CreateSecret(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input credcase.SecretCreateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a secret create request"})
			return
		}
		ref, err := service.PutSecret(r.Context(), input)
		respondCredentialResult(w, r, ref, err)
	}
}

func ListSecretRefs(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refs, err := service.ListSecretRefs(r.Context(), credcaseScope(r))
		respondCredentialResult(w, r, refs, err)
	}
}

func RotateSecret(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input credcase.SecretRotateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a secret rotate request"})
			return
		}
		input.ID = chi.URLParam(r, "id")
		ref, err := service.RotateSecret(r.Context(), input)
		respondCredentialResult(w, r, ref, err)
	}
}

func ValidateSecretProvider(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := service.ValidateSecretProvider(r.Context(), "")
		respondCredentialResult(w, r, status, err)
	}
}

func DeleteSecret(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := service.DeleteSecret(r.Context(), chi.URLParam(r, "id"), "")
		respondCredentialResult(w, r, map[string]string{"status": "deleted"}, err)
	}
}

func CreateCredential(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input credcase.CredentialCreateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a credential create request"})
			return
		}
		cred, err := service.CreateCredential(r.Context(), input)
		respondCredentialResult(w, r, cred, err)
	}
}

func ListCredentials(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		credentials, err := service.ListCredentials(r.Context())
		respondCredentialResult(w, r, credentials, err)
	}
}

func GetCredential(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cred, err := service.GetCredential(r.Context(), chi.URLParam(r, "id"))
		respondCredentialResult(w, r, cred, err)
	}
}

func DeleteCredential(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := service.DeleteCredential(r.Context(), chi.URLParam(r, "id"))
		respondCredentialResult(w, r, map[string]string{"status": "deleted"}, err)
	}
}

func ValidateCredential(service *credcase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.ValidateCredential(r.Context(), chi.URLParam(r, "id"), "")
		respondCredentialResult(w, r, result, err)
	}
}

func credcaseScope(r *http.Request) portsecret.Scope {
	return portsecret.Scope{ScopeType: r.URL.Query().Get("scopeType"), ScopeID: r.URL.Query().Get("scopeId")}
}

func respondCredentialResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusBadRequest
	code := "credential_error"
	if errors.Is(err, credcase.ErrCredentialNotFound) {
		status = http.StatusNotFound
		code = "credential_not_found"
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: code, Message: err.Error()})
}
