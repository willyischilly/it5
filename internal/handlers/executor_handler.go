package handlers

import (
	"net/http"
	"strconv"

	"planner-backend/internal/middleware"
	"planner-backend/pkg/response"
	"planner-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ExecutorHandler struct {
	executor *services.ExecutorService
}

func NewExecutorHandler(executor *services.ExecutorService) *ExecutorHandler {
	return &ExecutorHandler{executor: executor}
}

func (h *ExecutorHandler) ListTasks(c *gin.Context) {
	tasks, err := h.executor.ListTasks(middleware.GetUserID(c))
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
	task, err := h.executor.GetTask(middleware.GetUserID(c), taskID)
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
