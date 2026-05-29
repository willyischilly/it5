package handlers

import (
	"net/http"
	"strconv"

	"planner-backend/internal/middleware"
	"planner-backend/internal/services"
	"planner-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type ExecutorHandler struct {
	executor *services.ExecutorService
}

func NewExecutorHandler(executor *services.ExecutorService) *ExecutorHandler {
	return &ExecutorHandler{executor: executor}
}

func (h *ExecutorHandler) ClaimRequest(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}
	req, err := h.executor.ClaimRequest(middleware.GetUserID(c), requestID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, req)
}

func (h *ExecutorHandler) ListTasks(c *gin.Context) {
	tasks, err := h.executor.ListTasks()
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, tasks)
}

func (h *ExecutorHandler) GetTask(c *gin.Context) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}
	task, err := h.executor.GetTask(taskID)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, task)
}

func (h *ExecutorHandler) UpdateStatus(c *gin.Context) {
	taskID, ok := parseTaskID(c)
	if !ok {
		return
	}
	var in services.UpdateStatusInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	task, err := h.executor.UpdateTaskStatus(middleware.GetUserID(c), taskID, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, task)
}

func parseTaskID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid task id")
		return 0, false
	}
	return uint(id), true
}
