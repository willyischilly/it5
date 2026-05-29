package services

import (
	"strings"
	"time"

	"planner-backend/internal/models"
)

func taskExecutorFullName(t models.Task) string {
	if t.Executor == nil {
		return "—"
	}
	name := strings.TrimSpace(t.Executor.FullName())
	if name == "" {
		return "—"
	}
	return name
}

type RequestSummaryRow struct {
	RequestID      uint       `json:"request_id"`
	Title          string     `json:"title"`
	Contour        string     `json:"contour"`
	Status         string     `json:"status"`
	DeadlineAt     *time.Time `json:"deadline_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	TotalTasks     int        `json:"total_tasks"`
	CompletedTasks int        `json:"completed_tasks"`
	TotalHours     int        `json:"total_hours"`
	CompletedHours int        `json:"completed_hours"`
}

type SummaryTaskRow struct {
	RequestID        uint   `json:"request_id"`
	RequestTitle     string `json:"request_title"`
	WorkName         string `json:"work_name"`
	ExecutorFullName string `json:"executor_full_name,omitempty"`
	Status           string `json:"status"`
	NormativeHours   int    `json:"normative_hours"`
}

type SummaryReportResponse struct {
	GeneratedAt        time.Time           `json:"generated_at"`
	TotalRequests      int                 `json:"total_requests"`
	CompletedRequests  int                 `json:"completed_requests"`
	InProgressRequests int                 `json:"in_progress_requests"`
	SubmittedRequests  int                 `json:"submitted_requests"`
	DraftRequests      int                 `json:"draft_requests"`
	OverdueRequests    int                 `json:"overdue_requests"`
	TotalTasks         int                 `json:"total_tasks"`
	CompletedTasks     int                 `json:"completed_tasks"`
	TotalHours         int                 `json:"total_hours"`
	CompletedHours     int                 `json:"completed_hours"`
	Requests           []RequestSummaryRow `json:"requests"`
	Tasks              []SummaryTaskRow    `json:"tasks"`
}

func taskStats(tasks []models.Task) (total, completed, totalHours, completedHours int) {
	for _, t := range tasks {
		hours := 0
		if t.Work != nil {
			hours = t.Work.NormativeHours
		}
		total++
		totalHours += hours
		if t.Status == models.TaskStatusCompleted {
			completed++
			completedHours += hours
		}
	}
	return
}

func requestSummaryRow(req *models.Request) RequestSummaryRow {
	contour := ""
	if req.Contour != nil {
		contour = req.Contour.Name
	}
	total, completed, th, ch := taskStats(req.Tasks)
	return RequestSummaryRow{
		RequestID:      req.ID,
		Title:          req.Title,
		Contour:        contour,
		Status:         req.Status,
		DeadlineAt:     req.DeadlineAt,
		CreatedAt:      req.CreatedAt,
		TotalTasks:     total,
		CompletedTasks: completed,
		TotalHours:     th,
		CompletedHours: ch,
	}
}

func (s *CustomerService) GetAllReportsSummary(customerID uint) (*SummaryReportResponse, error) {
	return s.GetAllReportsSummaryForUser(customerID, models.RoleCustomer)
}

func (s *CustomerService) GetAllReportsSummaryForUser(userID uint, role string) (*SummaryReportResponse, error) {
	var list []models.Request
	var err error
	if role == models.RoleAdmin {
		list, err = s.requests.ListAll()
	} else {
		list, err = s.requests.ListByCustomer(userID)
	}
	if err != nil {
		return nil, err
	}
	return buildSummaryReport(list, func(i int) { _ = s.applyOverdueIfNeeded(&list[i]) }), nil
}

func buildSummaryReport(list []models.Request, beforeRow func(int)) *SummaryReportResponse {
	out := &SummaryReportResponse{
		GeneratedAt: time.Now(),
		Requests:    make([]RequestSummaryRow, 0, len(list)),
		Tasks:       make([]SummaryTaskRow, 0),
	}

	for i := range list {
		if beforeRow != nil {
			beforeRow(i)
		}
		row := requestSummaryRow(&list[i])
		out.Requests = append(out.Requests, row)

		for _, t := range list[i].Tasks {
			workName := ""
			hours := 0
			if t.Work != nil {
				workName = t.Work.Name
				hours = t.Work.NormativeHours
			}
			out.Tasks = append(out.Tasks, SummaryTaskRow{
				RequestID:        list[i].ID,
				RequestTitle:     list[i].Title,
				WorkName:         workName,
				ExecutorFullName: taskExecutorFullName(t),
				Status:           t.Status,
				NormativeHours:   hours,
			})
		}

		out.TotalRequests++
		out.TotalTasks += row.TotalTasks
		out.CompletedTasks += row.CompletedTasks
		out.TotalHours += row.TotalHours
		out.CompletedHours += row.CompletedHours

		switch list[i].Status {
		case models.RequestStatusCompleted:
			out.CompletedRequests++
		case models.RequestStatusInProgress:
			out.InProgressRequests++
		case models.RequestStatusSubmitted:
			out.SubmittedRequests++
		case models.RequestStatusDraft:
			out.DraftRequests++
		case models.RequestStatusOverdue:
			out.OverdueRequests++
		}
	}

	return out
}
