package handlers

import (
	"net/http"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	pipelineusecase "github.com/sevoniva/nivora/internal/usecase/pipeline"
)

func RuntimeRecoveryStatus(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := service.RuntimeStatus(r.Context())
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "runtime_status_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		RespondJSON(w, http.StatusOK, summary)
	}
}

func ReconcileRuntime(service *pipelineusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := service.ReconcileRuntime(r.Context(), pipelineusecase.RuntimeRecoveryOptions{})
		if err != nil {
			RespondError(w, r, http.StatusInternalServerError, dto.ErrorResponse{Code: "runtime_reconcile_failed", Message: err.Error(), Path: r.URL.Path})
			return
		}
		RespondJSON(w, http.StatusOK, summary)
	}
}
