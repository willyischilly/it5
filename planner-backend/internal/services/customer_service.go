package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"planner-backend/internal/models"
	"planner-backend/internal/repositories"
	"planner-backend/pkg/validation"

	"gorm.io/gorm"
)

type CustomerService struct {
	requests *repositories.RequestRepository
	tasks    *repositories.TaskRepository
	works    *repositories.WorkRepository
	contours *repositories.ContourRepository
	users    *repositories.UserRepository
	audit    *AuditService
}

func NewCustomerService(
	requests *repositories.RequestRepository,
	tasks *repositories.TaskRepository,
	works *repositories.WorkRepository,
	contours *repositories.ContourRepository,
	users *repositories.UserRepository,
	audit *AuditService,
) *CustomerService {
	return &CustomerService{
		requests: requests,
		tasks:    tasks,
		works:    works,
		contours: contours,
		users:    users,
		audit:    audit,
	}
}

type CreateRequestInput struct {
	Title      string     `json:"title"`
	ContourID  uint       `json:"contour_id"`
	DeadlineAt *time.Time `json:"deadline_at"`
}

type UpdateRequestInput struct {
	Title      *string    `json:"title"`
	ContourID  *uint      `json:"contour_id"`
	DeadlineAt *time.Time `json:"deadline_at"`
}

type ExtendDeadlineInput struct {
	DeadlineAt time.Time `json:"deadline_at"`
}

type AddTaskItem struct {
	WorkID  uint   `json:"work_id"`
	Comment string `json:"comment"`
}

func (s *CustomerService) CreateRequest(customerID uint, in CreateRequestInput) (*models.Request, error) {
	if !validation.NonEmpty(in.Title) {
		return nil, errors.New("title is required")
	}
	if _, err := s.contours.FindByID(in.ContourID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("contour not found")
		}
		return nil, err
	}
	if err := validateFutureDeadline(in.DeadlineAt); err != nil {
		return nil, err
	}

	req := &models.Request{
		Title:      in.Title,
		CustomerID: customerID,
		ContourID:  in.ContourID,
		Status:     models.RequestStatusDraft,
		DeadlineAt: in.DeadlineAt,
		TotalHours: 0,
	}
	if err := s.requests.Create(req); err != nil {
		return nil, err
	}

	status := models.RequestStatusDraft
	_ = s.audit.LogRequest(req.ID, customerID, models.RequestLogActionCreated, nil, &status,
		fmt.Sprintf("title=%q contour_id=%d", in.Title, in.ContourID))

	return s.requests.FindByID(req.ID)
}

func (s *CustomerService) ListRequests(customerID uint) ([]models.Request, error) {
	list, err := s.requests.ListByCustomer(customerID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		_ = s.applyOverdueIfNeeded(&list[i])
	}
	return s.requests.ListByCustomer(customerID)
}

func (s *CustomerService) GetRequest(customerID, requestID uint) (*models.Request, error) {
	return s.refreshRequest(customerID, requestID)
}

func (s *CustomerService) UpdateRequest(customerID, requestID uint, in UpdateRequestInput) (*models.Request, error) {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return nil, err
	}
	if !models.CanEditPlan(req) {
		return nil, errors.New("can only edit requests in draft status")
	}
	if in.Title != nil {
		if !validation.NonEmpty(*in.Title) {
			return nil, errors.New("title is required")
		}
		req.Title = *in.Title
	}
	if in.ContourID != nil {
		if _, err := s.contours.FindByID(*in.ContourID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("contour not found")
			}
			return nil, err
		}
		req.ContourID = *in.ContourID
	}
	if in.DeadlineAt != nil {
		if err := validateFutureDeadline(in.DeadlineAt); err != nil {
			return nil, err
		}
		req.DeadlineAt = in.DeadlineAt
	}
	if err := s.requests.Update(req); err != nil {
		return nil, err
	}
	return s.requests.FindByIDAndCustomer(requestID, customerID)
}

func (s *CustomerService) DeleteRequest(customerID, requestID uint) error {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return err
	}
	tasks, err := s.tasks.ListByRequest(requestID)
	if err != nil {
		return err
	}
	if !models.CanCustomerDeleteRequest(req, tasks) {
		return errors.New("can only delete draft requests or submitted plans where all tasks are still pending")
	}
	return s.requests.DeleteByID(requestID)
}

func (s *CustomerService) ExtendDeadline(customerID, requestID uint, in ExtendDeadlineInput) (*models.Request, error) {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return nil, err
	}
	if req.Status != models.RequestStatusOverdue {
		return nil, errors.New("deadline can only be extended for overdue requests")
	}
	if err := validateFutureDeadline(&in.DeadlineAt); err != nil {
		return nil, err
	}
	if err := s.requests.UpdateDeadline(requestID, &in.DeadlineAt); err != nil {
		return nil, err
	}
	req, err = s.requests.FindByIDAndCustomer(requestID, customerID)
	if err != nil {
		return nil, err
	}

	tasks, err := s.tasks.ListByRequest(requestID)
	if err != nil {
		return nil, err
	}
	if err := recomputeAndSaveRequestStatus(s.requests, s.audit, req, tasks, customerID); err != nil {
		return nil, err
	}
	_ = s.audit.LogRequest(requestID, customerID, models.RequestLogActionDeadlineExtended, nil, nil,
		in.DeadlineAt.Format(time.RFC3339))
	return s.requests.FindByIDAndCustomer(requestID, customerID)
}

func (s *CustomerService) getDraftOrOwned(customerID, requestID uint) (*models.Request, error) {
	req, err := s.requests.FindByIDAndCustomer(requestID, customerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("request not found")
		}
		return nil, err
	}
	return req, nil
}

type AddTasksInput struct {
	WorkIDs []uint        `json:"work_ids"`
	Tasks   []AddTaskItem `json:"tasks"`
}

func (s *CustomerService) AddTasks(customerID, requestID uint, in AddTasksInput) (*models.Request, error) {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return nil, err
	}
	if !models.CanEditPlan(req) {
		return nil, errors.New("can only modify tasks while request is in draft")
	}

	items, err := normalizeAddTasks(in)
	if err != nil {
		return nil, err
	}

	workIDs := make([]uint, len(items))
	for i, it := range items {
		workIDs[i] = it.WorkID
	}
	uniqueIDs := dedupeUints(workIDs)
	works, err := s.works.FindByIDs(uniqueIDs)
	if err != nil {
		return nil, err
	}
	if len(works) != len(uniqueIDs) {
		return nil, errors.New("one or more work ids not found")
	}

	existing, err := s.tasks.WorkIDsAlreadyInRequest(requestID, uniqueIDs)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, fmt.Errorf("work ids already in plan: %v", existing)
	}

	newTasks := make([]models.Task, 0, len(items))
	for _, it := range items {
		newTasks = append(newTasks, models.Task{
			RequestID:       requestID,
			WorkID:          it.WorkID,
			Status:          models.TaskStatusPending,
			CustomerComment: strings.TrimSpace(it.Comment),
		})
	}
	if err := s.tasks.CreateBatch(newTasks); err != nil {
		return nil, err
	}

	if err := s.recalcTotalHours(requestID); err != nil {
		return nil, err
	}

	return s.requests.FindByIDAndCustomer(requestID, customerID)
}

func (s *CustomerService) DeleteTask(customerID, requestID, taskID uint) error {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return err
	}
	task, err := s.tasks.FindByRequestAndID(requestID, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("task not found")
		}
		return err
	}
	if !models.CanCustomerDeleteTask(req, task) {
		return errors.New("can only delete tasks in pending status while the plan is not completed or overdue")
	}

	if err := s.tasks.DeleteByRequestAndID(requestID, taskID); err != nil {
		return err
	}

	if err := s.recalcTotalHours(requestID); err != nil {
		return err
	}

	count, err := s.tasks.CountByRequest(requestID)
	if err != nil {
		return err
	}
	if count == 0 && req.Status == models.RequestStatusSubmitted {
		_ = s.requests.UpdateStatus(requestID, models.RequestStatusDraft)
	} else if req.Status != models.RequestStatusDraft {
		tasks, err := s.tasks.ListByRequest(requestID)
		if err != nil {
			return err
		}
		return recomputeAndSaveRequestStatus(s.requests, s.audit, req, tasks, customerID)
	}
	return nil
}

func (s *CustomerService) recalcTotalHours(requestID uint) error {
	total, err := s.tasks.SumNormativeHours(requestID)
	if err != nil {
		return err
	}
	return s.requests.UpdateTotalHours(requestID, total)
}

func (s *CustomerService) Submit(customerID, requestID uint) (*models.Request, error) {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return nil, err
	}
	if !models.CanEditPlan(req) {
		return nil, errors.New("only draft requests can be submitted")
	}
	if req.DeadlineAt != nil && time.Now().After(*req.DeadlineAt) {
		return nil, errors.New("deadline has already passed")
	}

	count, err := s.tasks.CountByRequest(requestID)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("request must have at least one task")
	}

	executors, err := s.users.ListByRole(models.RoleExecutor)
	if err != nil {
		return nil, err
	}
	if len(executors) == 0 {
		return nil, errors.New("no executors available for assignment")
	}

	tasks, err := s.tasks.ListByRequest(requestID)
	if err != nil {
		return nil, err
	}

	for i := range tasks {
		executor := executors[i%len(executors)]
		if err := s.tasks.AssignExecutor(tasks[i].ID, executor.ID); err != nil {
			return nil, err
		}
	}

	oldStatus := req.Status
	req.Status = models.RequestStatusSubmitted
	if err := s.requests.Update(req); err != nil {
		return nil, err
	}

	newStatus := models.RequestStatusSubmitted
	_ = s.audit.LogRequest(requestID, customerID, models.RequestLogActionSubmitted, &oldStatus, &newStatus,
		fmt.Sprintf("tasks=%d executors=%d", len(tasks), len(executors)))

	return s.requests.FindByIDAndCustomer(requestID, customerID)
}

type ReportTask struct {
	Name            string `json:"name"`
	NormativeHours  int    `json:"normative_hours"`
	Status          string `json:"status"`
	CustomerComment string `json:"customer_comment,omitempty"`
}

type ReportResponse struct {
	RequestID      uint         `json:"request_id"`
	Title          string       `json:"title"`
	Contour        string       `json:"contour"`
	Status         string       `json:"status"`
	DeadlineAt     *time.Time   `json:"deadline_at,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	Tasks          []ReportTask `json:"tasks"`
	TotalTasks     int          `json:"total_tasks"`
	CompletedTasks int          `json:"completed_tasks"`
	TotalHours     int          `json:"total_hours"`
	CompletedHours int          `json:"completed_hours"`
}

func (s *CustomerService) GetReport(customerID, requestID uint) (*ReportResponse, error) {
	req, err := s.refreshRequest(customerID, requestID)
	if err != nil {
		return nil, err
	}

	reportTasks := make([]ReportTask, 0, len(req.Tasks))
	completed := 0
	completedHours := 0

	for _, t := range req.Tasks {
		name := ""
		hours := 0
		if t.Work != nil {
			name = t.Work.Name
			hours = t.Work.NormativeHours
		}
		reportTasks = append(reportTasks, ReportTask{
			Name: name, NormativeHours: hours, Status: t.Status,
			CustomerComment: t.CustomerComment,
		})
		if t.Status == models.TaskStatusCompleted {
			completed++
			completedHours += hours
		}
	}

	contourName := ""
	if req.Contour != nil {
		contourName = req.Contour.Name
	}

	return &ReportResponse{
		RequestID:      req.ID,
		Title:          req.Title,
		Contour:        contourName,
		Status:         req.Status,
		DeadlineAt:     req.DeadlineAt,
		CreatedAt:      req.CreatedAt,
		Tasks:          reportTasks,
		TotalTasks:     len(req.Tasks),
		CompletedTasks: completed,
		TotalHours:     req.TotalHours,
		CompletedHours: completedHours,
	}, nil
}

func (s *CustomerService) ListWorks() ([]models.Work, error) {
	return s.works.List()
}

func (s *CustomerService) ListContours() ([]models.DeploymentContour, error) {
	return s.contours.List()
}

func dedupeUints(ids []uint) []uint {
	seen := make(map[uint]bool, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (s *CustomerService) ViewRequestByID(requestID uint) (*models.Request, error) {
	req, err := s.requests.FindByID(requestID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("request not found")
		}
		return nil, err
	}
	if err := s.applyOverdueIfNeeded(req); err != nil {
		return nil, err
	}
	return s.requests.FindByID(requestID)
}

func validateFutureDeadline(deadline *time.Time) error {
	if deadline == nil {
		return nil
	}
	if !deadline.After(time.Now()) {
		return errors.New("deadline must be in the future")
	}
	return nil
}

func normalizeAddTasks(in AddTasksInput) ([]AddTaskItem, error) {
	if len(in.Tasks) > 0 {
		for _, t := range in.Tasks {
			if t.WorkID == 0 {
				return nil, errors.New("work_id is required for each task")
			}
		}
		return in.Tasks, nil
	}
	if len(in.WorkIDs) == 0 {
		return nil, errors.New("tasks or work_ids is required")
	}
	out := make([]AddTaskItem, len(in.WorkIDs))
	for i, id := range in.WorkIDs {
		out[i] = AddTaskItem{WorkID: id}
	}
	return out, nil
}
