package models

import "time"

// ShouldBeOverdue — просрочка только у заявки в целом (не у отдельных задач).
func ShouldBeOverdue(req *Request) bool {
	if req.DeadlineAt == nil {
		return false
	}
	if req.Status == RequestStatusDraft || req.Status == RequestStatusCompleted {
		return false
	}
	return time.Now().After(*req.DeadlineAt)
}

func CanEditPlan(req *Request) bool {
	return req.Status == RequestStatusDraft
}

func CanCustomerDeleteTask(req *Request, task *Task) bool {
	if task.Status != TaskStatusPending {
		return false
	}
	switch req.Status {
	case RequestStatusDraft, RequestStatusSubmitted, RequestStatusInProgress:
		return true
	default:
		return false
	}
}

func CanCustomerDeleteRequest(req *Request, tasks []Task) bool {
	if req.Status == RequestStatusDraft {
		return true
	}
	if req.Status != RequestStatusSubmitted {
		return false
	}
	for _, t := range tasks {
		if t.Status != TaskStatusPending {
			return false
		}
	}
	return true
}
