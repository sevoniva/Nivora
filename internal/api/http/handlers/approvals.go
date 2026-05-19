package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sevoniva/nivora/internal/api/http/dto"
	domainapproval "github.com/sevoniva/nivora/internal/domain/approval"
	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
	approvalusecase "github.com/sevoniva/nivora/internal/usecase/approval"
)

func CreateApproval(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input approvalusecase.ApprovalCreateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be an approval request"})
			return
		}
		approval, err := service.CreateApprovalRequest(r.Context(), input)
		respondApprovalResult(w, r, approval, err)
	}
}

func ListApprovals(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := service.ListApprovals(r.Context())
		respondApprovalResult(w, r, payload, err)
	}
}

func GetApproval(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := service.GetApproval(r.Context(), chi.URLParam(r, "id"))
		respondApprovalResult(w, r, payload, err)
	}
}

func ApproveApproval(service *approvalusecase.Service) http.HandlerFunc {
	return decideApproval(service, true)
}

func RejectApproval(service *approvalusecase.Service) http.HandlerFunc {
	return decideApproval(service, false)
}

func CancelApproval(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input approvalusecase.DecisionInput
		_ = json.NewDecoder(r.Body).Decode(&input)
		payload, err := service.Cancel(r.Context(), chi.URLParam(r, "id"), input)
		respondApprovalResult(w, r, payload, err)
	}
}

func decideApproval(service *approvalusecase.Service, approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input approvalusecase.DecisionInput
		_ = json.NewDecoder(r.Body).Decode(&input)
		var payload domainapproval.ApprovalRequest
		var err error
		if approve {
			payload, err = service.Approve(r.Context(), chi.URLParam(r, "id"), input)
		} else {
			payload, err = service.Reject(r.Context(), chi.URLParam(r, "id"), input)
		}
		respondApprovalResult(w, r, payload, err)
	}
}

func ListChangeWindows(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := service.ListChangeWindows(r.Context())
		respondApprovalResult(w, r, payload, err)
	}
}

func CreateChangeWindow(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input domainapproval.ChangeWindow
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a change window"})
			return
		}
		payload, err := service.CreateChangeWindow(r.Context(), input)
		respondApprovalResult(w, r, payload, err)
	}
}

func GetChangeWindow(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := service.GetChangeWindow(r.Context(), chi.URLParam(r, "id"))
		respondApprovalResult(w, r, payload, err)
	}
}

func EvaluateChangeWindow(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input approvalusecase.ChangeWindowEvaluateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a change window evaluation request"})
			return
		}
		payload, err := service.EvaluateChangeWindowInput(r.Context(), input)
		respondApprovalResult(w, r, payload, err)
	}
}

func ListNotifications(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := service.ListNotifications(r.Context())
		respondApprovalResult(w, r, payload, err)
	}
}

func TestNotification(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input domainnotification.Notification
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_request", Message: "request body must be a notification"})
			return
		}
		payload, err := service.SendNotification(r.Context(), input)
		respondApprovalResult(w, r, payload, err)
	}
}

func respondApprovalResult(w http.ResponseWriter, r *http.Request, payload any, err error) {
	if err == nil {
		RespondJSON(w, http.StatusOK, payload)
		return
	}
	status := http.StatusBadRequest
	if errors.Is(err, approvalusecase.ErrApprovalNotFound) || errors.Is(err, approvalusecase.ErrChangeWindowNotFound) {
		status = http.StatusNotFound
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: "governance_error", Message: err.Error()})
}
