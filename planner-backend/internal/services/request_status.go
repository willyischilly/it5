package services

import (
	"planner-backend/internal/models"
	"planner-backend/internal/repositories"
)

func computeRequestStatus(req *models.Request, tasks []models.Task) string {
	if len(tasks) == 0 {
		if req.Status == models.RequestStatusDraft {
			return models.RequestStatusDraft
		}
		if models.ShouldBeOverdue(req) {
			return models.RequestStatusOverdue
		}
		return req.Status
	}

	allCompleted := true
	hasInProgress := false
	for _, t := range tasks {
		if t.Status != models.TaskStatusCompleted {
			allCompleted = false
		}
		if t.Status == models.TaskStatusInProgress {
			hasInProgress = true
		}
	}
	if allCompleted {
		return models.RequestStatusCompleted
	}
	if models.ShouldBeOverdue(req) {
		return models.RequestStatusOverdue
	}
	if hasInProgress {
		return models.RequestStatusInProgress
	}
	if req.Status == models.RequestStatusDraft {
		return models.RequestStatusDraft
	}
	return models.RequestStatusSubmitted
}

func (s *CustomerService) applyOverdueIfNeeded(req *models.Request) error {
	if !models.ShouldBeOverdue(req) {
		return nil
	}
	if req.Status == models.RequestStatusOverdue {
		return nil
	}
	if req.Status != models.RequestStatusSubmitted && req.Status != models.RequestStatusInProgress {
		return nil
	}
	old := req.Status
	newSt := models.RequestStatusOverdue
	if err := s.requests.UpdateStatus(req.ID, newSt); err != nil {
		return err
	}
	req.Status = newSt
	return s.audit.LogRequest(req.ID, req.CustomerID, models.RequestLogActionStatusChanged, &old, &newSt, "deadline passed")
}

func (s *CustomerService) refreshRequest(customerID, requestID uint) (*models.Request, error) {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return nil, err
	}
	if err := s.applyOverdueIfNeeded(req); err != nil {
		return nil, err
	}
	return s.requests.FindByIDAndCustomer(requestID, customerID)
}

func recomputeAndSaveRequestStatus(
	requests *repositories.RequestRepository,
	audit *AuditService,
	req *models.Request,
	tasks []models.Task,
	userID uint,
) error {
	newStatus := computeRequestStatus(req, tasks)
	if newStatus == req.Status {
		return nil
	}
	old := req.Status
	if err := requests.UpdateStatus(req.ID, newStatus); err != nil {
		return err
	}
	req.Status = newStatus
	return audit.LogRequest(req.ID, userID, models.RequestLogActionStatusChanged, &old, &newStatus, "")
}
