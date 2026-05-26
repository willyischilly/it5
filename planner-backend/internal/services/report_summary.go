package services

import (
	"time"

	"planner-backend/internal/models"
)

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
	list, err := s.requests.ListByCustomer(customerID)
	if err != nil {
		return nil, err
	}

	out := &SummaryReportResponse{
		GeneratedAt: time.Now(),
		Requests:    make([]RequestSummaryRow, 0, len(list)),
	}

	for i := range list {
		_ = s.applyOverdueIfNeeded(&list[i])
		row := requestSummaryRow(&list[i])
		out.Requests = append(out.Requests, row)

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

	return out, nil
}
