package services

import (
	"errors"
	"fmt"
	"time"

	"planner-backend/internal/models"
	"planner-backend/pkg/validation"
	"planner-backend/internal/repositories"

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
	Title     string `json:"title"`
	ContourID uint   `json:"contour_id"`
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

	req := &models.Request{
		Title:      in.Title,
		CustomerID: customerID,
		ContourID:  in.ContourID,
		Status:     models.RequestStatusDraft,
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
	return s.requests.ListByCustomer(customerID)
}

func (s *CustomerService) GetRequest(customerID, requestID uint) (*models.Request, error) {
	return s.getDraftOrOwned(customerID, requestID)
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
	WorkIDs []uint `json:"work_ids"`
}

func (s *CustomerService) AddTasks(customerID, requestID uint, in AddTasksInput) (*models.Request, error) {
	req, err := s.getDraftOrOwned(customerID, requestID)
	if err != nil {
		return nil, err
	}
	if req.Status != models.RequestStatusDraft {
		return nil, errors.New("can only modify tasks in draft requests")
	}
	if len(in.WorkIDs) == 0 {
		return nil, errors.New("work_ids is required")
	}

	uniqueIDs := dedupeUints(in.WorkIDs)
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

	newTasks := make([]models.Task, 0, len(uniqueIDs))
	for _, wid := range uniqueIDs {
		newTasks = append(newTasks, models.Task{
			RequestID: requestID,
			WorkID:    wid,
			Status:    models.TaskStatusPending,
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
	if req.Status != models.RequestStatusDraft {
		return errors.New("can only modify tasks in draft requests")
	}

	if err := s.tasks.DeleteByRequestAndID(requestID, taskID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("task not found")
		}
		return err
	}

	return s.recalcTotalHours(requestID)
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
	if req.Status != models.RequestStatusDraft {
		return nil, errors.New("only draft requests can be submitted")
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
	Name           string `json:"name"`
	NormativeHours int    `json:"normative_hours"`
	Status         string `json:"status"`
}

type ReportResponse struct {
	RequestID      uint         `json:"request_id"`
	Title          string       `json:"title"`
	Contour        string       `json:"contour"`
	CreatedAt      time.Time    `json:"created_at"`
	Tasks          []ReportTask `json:"tasks"`
	TotalTasks     int          `json:"total_tasks"`
	CompletedTasks int          `json:"completed_tasks"`
	TotalHours     int          `json:"total_hours"`
	CompletedHours int          `json:"completed_hours"`
}

func (s *CustomerService) GetReport(customerID, requestID uint) (*ReportResponse, error) {
	req, err := s.getDraftOrOwned(customerID, requestID)
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
	return req, nil
}
