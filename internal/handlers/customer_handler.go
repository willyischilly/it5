package handlers

import (
	"net/http"
	"strconv"

	"planner-backend/internal/middleware"
	"planner-backend/pkg/response"
	"planner-backend/internal/services"

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
		pdfBytes, err := services.BuildReportPDF(report)
		if err != nil {
			response.Internal(c, "failed to generate PDF")
			return
		}
		c.Header("Content-Disposition", "attachment; filename=report.pdf")
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
		return
	}
	response.JSON(c, http.StatusOK, report)
}

func parseRequestID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid request id")
		return 0, false
	}
	return uint(id), true
}
