package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"planner-backend/internal/models"
	"planner-backend/internal/repositories"

	"gorm.io/gorm"
)

type ExecutorService struct {
	tasks    *repositories.TaskRepository
	requests *repositories.RequestRepository
	users    *repositories.UserRepository
	audit    *AuditService
}

func NewExecutorService(
	tasks *repositories.TaskRepository,
	requests *repositories.RequestRepository,
	users *repositories.UserRepository,
	audit *AuditService,
) *ExecutorService {
	return &ExecutorService{tasks: tasks, requests: requests, users: users, audit: audit}
}

func (s *ExecutorService) ListRequests() ([]models.Request, error) {
	return s.requests.ListForExecutors()
}

func (s *ExecutorService) ListTasks() ([]models.Task, error) {
	return s.tasks.ListAllForExecutors()
}

func (s *ExecutorService) GetRequest(requestID uint) (*models.Request, error) {
	req, err := s.requests.FindByIDVisibleToExecutor(requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("request not found")
		}
		return nil, err
	}
	return req, nil
}

func (s *ExecutorService) GetTask(taskID uint) (*models.Task, error) {
	task, err := s.tasks.FindByIDVisibleToExecutor(taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}
		return nil, err
	}
	return task, nil
}

func (s *ExecutorService) ClaimRequest(executorID, requestID uint) (*models.Request, error) {
	executor, err := s.users.FindByID(executorID)
	if err != nil {
		return nil, errors.New("executor not found")
	}
	if executor.Role != models.RoleExecutor {
		return nil, errors.New("only executors can claim requests")
	}

	req, err := s.requests.FindByIDVisibleToExecutor(requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("request not found")
		}
		return nil, err
	}

	if req.Status != models.RequestStatusSubmitted {
		return nil, errors.New("request can only be claimed when status is submitted (tasks in plans)")
	}

	nonPending, err := s.tasks.CountNonPending(requestID)
	if err != nil {
		return nil, err
	}
	if nonPending > 0 {
		return nil, errors.New("all tasks must be in pending status to claim")
	}

	assignedOther, err := s.tasks.CountAssignedToOther(requestID, executorID)
	if err != nil {
		return nil, err
	}
	if assignedOther > 0 {
		return nil, errors.New("request already has tasks assigned to another executor")
	}

	unassigned, err := s.tasks.CountPendingUnassigned(requestID)
	if err != nil {
		return nil, err
	}
	if unassigned == 0 {
		return nil, errors.New("no unassigned pending tasks to claim")
	}

	rows, err := s.tasks.ClaimPendingTasks(requestID, executorID)
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, errors.New("failed to claim request")
	}

	fullName := executor.FullName()
	_ = s.audit.LogRequest(requestID, executorID, models.RequestLogActionClaimed, nil, nil,
		fmt.Sprintf("executor=%s tasks=%d", fullName, rows))
	log.Printf("[audit] request=%d claimed by %s (%d tasks)", requestID, fullName, rows)

	return s.requests.FindByIDVisibleToExecutor(requestID)
}

type UpdateStatusInput struct {
	Status string `json:"status"`
}

func (s *ExecutorService) UpdateTaskStatus(executorID, taskID uint, in UpdateStatusInput) (*models.Task, error) {
	task, err := s.tasks.FindByIDAndExecutor(taskID, executorID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found or not assigned to you")
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

func (s *ExecutorService) syncRequestStatus(requestID, userID uint) error {
	req, err := s.requests.FindByID(requestID)
	if err != nil {
		return err
	}
	tasks, err := s.tasks.ListByRequest(requestID)
	if err != nil {
		return err
	}
	return recomputeAndSaveRequestStatus(s.requests, s.audit, req, tasks, userID)
}
