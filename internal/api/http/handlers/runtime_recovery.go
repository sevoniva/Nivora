package handlers

import (
	"net/http"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	runtimecenter "github.com/sevoniva/nivora/internal/usecase/runtimecenter"
)

func RuntimeRecoveryStatus(service *runtimecenter.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := service.Status(r.Context(), runtimecenter.Options{})
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "runtime_status_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		RespondJSON(w, http.StatusOK, summary)
	}
}

func ReconcileRuntime(service *runtimecenter.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := service.Reconcile(r.Context(), runtimecenter.Options{})
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "runtime_reconcile_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		RespondJSON(w, http.StatusOK, summary)
	}
}
