package handlers

import (
	"net/http"

	"github.com/sevoniva/nivora/internal/api/http/dto"
	integrationusecase "github.com/sevoniva/nivora/internal/usecase/integration"
)

func ListIntegrations(service *integrationusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := service.List(r.Context())
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "integration_error", Message: err.Error()})
			return
		}
		RespondJSON(w, http.StatusOK, result)
	}
}
