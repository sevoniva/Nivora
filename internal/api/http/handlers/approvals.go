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
	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
	releaseorchestration "github.com/sevoniva/nivora/internal/usecase/releaseorchestration"
)

type approvalSubjectResumeResponse struct {
	Approval    domainapproval.ApprovalRequest `json:"approval"`
	SubjectID   string                         `json:"subjectId"`
	SubjectType string                         `json:"subjectType"`
	Result      any                            `json:"result,omitempty"`
}

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

func ExpireApproval(service *approvalusecase.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input approvalusecase.DecisionInput
		_ = json.NewDecoder(r.Body).Decode(&input)
		payload, err := service.Expire(r.Context(), chi.URLParam(r, "id"), input)
		respondApprovalResult(w, r, payload, err)
	}
}

func ResumeApprovalSubject(approvalService *approvalusecase.Service, deploymentService *deploymentusecase.Service, releaseService *releaseorchestration.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		approval, err := approvalService.GetApproval(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			respondApprovalSubjectResumeError(w, r, err)
			return
		}
		if approval.SubjectID == "" {
			respondApprovalSubjectResumeError(w, r, errors.New("approval subjectId is required to resume a subject"))
			return
		}
		if approval.Status == domainapproval.StatusPending {
			respondApprovalSubjectResumeError(w, r, errors.New("approval request is still Pending"))
			return
		}
		actorID := approvalDecisionActor(approval)
		response := approvalSubjectResumeResponse{Approval: approval, SubjectID: approval.SubjectID, SubjectType: approval.SubjectType}
		switch approval.SubjectType {
		case domainapproval.SubjectDeployment:
			record, err := deploymentService.ApplyApprovalDecision(r.Context(), approval.SubjectID, approval, actorID)
			if err != nil {
				respondApprovalSubjectResumeError(w, r, err)
				return
			}
			response.Result = record
			RespondJSON(w, http.StatusOK, response)
		case domainapproval.SubjectRelease:
			record, err := releaseService.ApplyApprovalDecision(r.Context(), approval.SubjectID, approval, actorID)
			if err != nil {
				respondApprovalSubjectResumeError(w, r, err)
				return
			}
			response.Result = record
			RespondJSON(w, http.StatusOK, response)
		case domainapproval.SubjectPipeline:
			RespondError(w, r, http.StatusNotImplemented, dto.ErrorResponse{Code: "not_implemented", Message: "approval subject resume for pipeline is not implemented"})
		default:
			respondApprovalSubjectResumeError(w, r, errors.New("approval subjectType must be deployment, release, or pipeline"))
		}
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

func approvalDecisionActor(request domainapproval.ApprovalRequest) string {
	if len(request.Decisions) > 0 {
		return request.Decisions[len(request.Decisions)-1].Approver
	}
	if request.RequestedBy != "" {
		return request.RequestedBy
	}
	return "approval-system"
}

func respondApprovalSubjectResumeError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusBadRequest
	if errors.Is(err, approvalusecase.ErrApprovalNotFound) ||
		errors.Is(err, deploymentusecase.ErrRunNotFound) ||
		errors.Is(err, releaseorchestration.ErrExecutionNotFound) {
		status = http.StatusNotFound
	}
	RespondError(w, r, status, dto.ErrorResponse{Code: "approval_subject_resume_failed", Message: err.Error()})
}
