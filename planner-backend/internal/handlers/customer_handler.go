package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"planner-backend/internal/middleware"
	"planner-backend/internal/services"
	"planner-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type CustomerHandler struct {
	customer *services.CustomerService
}

func NewCustomerHandler(customer *services.CustomerService) *CustomerHandler {
	return &CustomerHandler{customer: customer}
}

func (h *CustomerHandler) ListWorks(c *gin.Context) {
	works, err := h.customer.ListWorks()
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, works)
}

func (h *CustomerHandler) ListContours(c *gin.Context) {
	contours, err := h.customer.ListContours()
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, contours)
}

func (h *CustomerHandler) CreateRequest(c *gin.Context) {
	var in services.CreateRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	req, err := h.customer.CreateRequest(middleware.GetUserID(c), in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, req)
}

func (h *CustomerHandler) ListRequests(c *gin.Context) {
	reqs, err := h.customer.ListRequests(middleware.GetUserID(c))
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, reqs)
}

func (h *CustomerHandler) GetRequest(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	req, err := h.customer.GetRequest(middleware.GetUserID(c), requestID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, req)
}

func (h *CustomerHandler) UpdateRequest(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	var in services.UpdateRequestInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	req, err := h.customer.UpdateRequest(middleware.GetUserID(c), requestID, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, req)
}

func (h *CustomerHandler) DeleteRequest(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	if err := h.customer.DeleteRequest(middleware.GetUserID(c), requestID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *CustomerHandler) ExtendDeadline(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	var in services.ExtendDeadlineInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	req, err := h.customer.ExtendDeadline(middleware.GetUserID(c), requestID, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, req)
}

func (h *CustomerHandler) AddTasks(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	var in services.AddTasksInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	req, err := h.customer.AddTasks(middleware.GetUserID(c), requestID, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, req)
}

func (h *CustomerHandler) DeleteTask(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	taskID, err := strconv.ParseUint(c.Param("task_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid task id")
		return
	}
	if err := h.customer.DeleteTask(middleware.GetUserID(c), requestID, uint(taskID)); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *CustomerHandler) Submit(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	req, err := h.customer.Submit(middleware.GetUserID(c), requestID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, req)
}

func (h *CustomerHandler) GetReport(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	report, err := h.customer.GetReport(middleware.GetUserID(c), requestID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	format := c.DefaultQuery("format", "json")
	if format == "pdf" {
		h.writeReportPDF(c, report)
		return
	}
	response.JSON(c, http.StatusOK, report)
}

func (h *CustomerHandler) GetReportPDF(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	report, err := h.customer.GetReport(middleware.GetUserID(c), requestID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	h.writeReportPDF(c, report)
}

func (h *CustomerHandler) writeReportPDF(c *gin.Context, report *services.ReportResponse) {
	pdfBytes, err := services.BuildReportPDF(report)
	if err != nil {
		response.Internal(c, "failed to generate PDF")
		return
	}
	name := fmt.Sprintf("report_%d.pdf", report.RequestID)
	c.Header("Content-Disposition", "attachment; filename="+name)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func (h *CustomerHandler) GetAllReportsSummary(c *gin.Context) {
	summary, err := h.customer.GetAllReportsSummary(middleware.GetUserID(c))
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	if c.Query("format") == "pdf" {
		h.writeSummaryPDF(c, summary)
		return
	}
	response.JSON(c, http.StatusOK, summary)
}

func (h *CustomerHandler) GetAllReportsSummaryPDF(c *gin.Context) {
	summary, err := h.customer.GetAllReportsSummary(middleware.GetUserID(c))
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	h.writeSummaryPDF(c, summary)
}

func (h *CustomerHandler) writeSummaryPDF(c *gin.Context, summary *services.SummaryReportResponse) {
	pdfBytes, err := services.BuildSummaryReportPDF(summary)
	if err != nil {
		response.Internal(c, "failed to generate PDF")
		return
	}
	c.Header("Content-Disposition", "attachment; filename=reports_summary.pdf")
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func parseRequestID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return 0, false
	}
	return uint(id), true
}
