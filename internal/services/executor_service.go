package services

import (
	"errors"
	"log"
	"time"

	"planner-backend/internal/models"
	"planner-backend/internal/repositories"

	"gorm.io/gorm"
)

type ExecutorService struct {
	tasks    *repositories.TaskRepository
	requests *repositories.RequestRepository
	audit    *AuditService
}

func NewExecutorService(
	tasks *repositories.TaskRepository,
	requests *repositories.RequestRepository,
	audit *AuditService,
) *ExecutorService {
	return &ExecutorService{tasks: tasks, requests: requests, audit: audit}
}

func (s *ExecutorService) ListTasks(executorID uint) ([]models.Task, error) {
	return s.tasks.ListByExecutor(executorID)
}

func (s *ExecutorService) GetTask(executorID, taskID uint) (*models.Task, error) {
	task, err := s.tasks.FindByIDAndExecutor(taskID, executorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}
		return nil, err
	}
	return task, nil
}

type UpdateStatusInput struct {
	Status string `json:"status"`
}

func (s *ExecutorService) UpdateTaskStatus(executorID, taskID uint, in UpdateStatusInput) (*models.Task, error) {
	task, err := s.tasks.FindByIDAndExecutor(taskID, executorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}
		return nil, err
	}

	oldStatus := task.Status
	if !models.ValidTaskTransition(oldStatus, in.Status) {
		return nil, errors.New("invalid status transition")
	}

	now := time.Now()
	task.Status = in.Status

	if in.Status == models.TaskStatusInProgress {
		task.StartedAt = &now
	}
	if in.Status == models.TaskStatusCompleted {
		task.CompletedAt = &now
	}

	if err := s.tasks.UpdateStatus(task); err != nil {
		return nil, err
	}

	old := oldStatus
	taskLog := &models.TaskLog{
		TaskID:    task.ID,
		UserID:    executorID,
		OldStatus: &old,
		NewStatus: in.Status,
	}
	if err := s.tasks.CreateLog(taskLog); err != nil {
		return nil, err
	}
	log.Printf("[audit] task=%d user=%d status %s -> %s", task.ID, executorID, oldStatus, in.Status)

	if err := s.syncRequestStatus(task.RequestID, executorID); err != nil {
		return nil, err
	}

	return s.tasks.FindByID(task.ID)
}

func (s *ExecutorService) EnsureRequestAccess(executorID, requestID uint) error {
	tasks, err := s.tasks.ListByExecutor(executorID)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if t.RequestID == requestID {
			return nil
		}
	}
	return errors.New("access denied to this request")
}

func (s *ExecutorService) syncRequestStatus(requestID, userID uint) error {
	req, err := s.requests.FindByID(requestID)
	if err != nil {
		return err
	}
	oldStatus := req.Status
	newStatus := oldStatus

	allDone, err := s.tasks.AllCompleted(requestID)
	if err != nil {
		return err
	}
	if allDone {
		newStatus = models.RequestStatusCompleted
	} else {
		inProgress, err := s.tasks.HasInProgress(requestID)
		if err != nil {
			return err
		}
		if inProgress {
			newStatus = models.RequestStatusInProgress
		}
	}

	if newStatus == oldStatus {
		return nil
	}

	req.Status = newStatus
	if err := s.requests.Update(req); err != nil {
		return err
	}

	old := oldStatus
	newS := newStatus
	return s.audit.LogRequest(requestID, userID, models.RequestLogActionStatusChanged, &old, &newS, "")
}
