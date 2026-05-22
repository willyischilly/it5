package handlers

import (
	"net/http"
	"strconv"

	"planner-backend/pkg/response"
	"planner-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	admin *services.AdminService
}

func NewAdminHandler(admin *services.AdminService) *AdminHandler {
	return &AdminHandler{admin: admin}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return uint(id), true
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.admin.ListUsers()
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, users)
}

func (h *AdminHandler) CreateUser(c *gin.Context) {
	var in services.CreateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	user, err := h.admin.CreateUser(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, user)
}

func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in services.UpdateUserInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	user, err := h.admin.UpdateUser(id, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, user)
}

func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.admin.DeleteUser(id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) ListWorks(c *gin.Context) {
	works, err := h.admin.ListWorks()
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, works)
}

func (h *AdminHandler) CreateWork(c *gin.Context) {
	var in services.WorkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	work, err := h.admin.CreateWork(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, work)
}

func (h *AdminHandler) UpdateWork(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in services.UpdateWorkInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	work, err := h.admin.UpdateWork(id, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, work)
}

func (h *AdminHandler) DeleteWork(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.admin.DeleteWork(id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) ListContours(c *gin.Context) {
	contours, err := h.admin.ListContours()
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, contours)
}

func (h *AdminHandler) CreateContour(c *gin.Context) {
	var in services.ContourInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	contour, err := h.admin.CreateContour(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, contour)
}

func (h *AdminHandler) UpdateContour(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var in services.ContourInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	contour, err := h.admin.UpdateContour(id, in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, contour)
}

func (h *AdminHandler) DeleteContour(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.admin.DeleteContour(id); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func parseOptionalQueryUint(c *gin.Context, name string) (*uint, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid "+name)
		return nil, false
	}
	v := uint(id)
	return &v, true
}

func parseAuditLimit(c *gin.Context) (int, bool) {
	raw := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(raw)
	if err != nil {
		response.BadRequest(c, "invalid limit")
		return 0, false
	}
	return limit, true
}

func (h *AdminHandler) ListRequestLogs(c *gin.Context) {
	requestID, ok := parseOptionalQueryUint(c, "request_id")
	if !ok {
		return
	}
	limit, ok := parseAuditLimit(c)
	if !ok {
		return
	}
	logs, err := h.admin.ListRequestLogs(requestID, limit)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, logs)
}

func (h *AdminHandler) ListTaskLogs(c *gin.Context) {
	taskID, ok := parseOptionalQueryUint(c, "task_id")
	if !ok {
		return
	}
	requestID, ok := parseOptionalQueryUint(c, "request_id")
	if !ok {
		return
	}
	limit, ok := parseAuditLimit(c)
	if !ok {
		return
	}
	logs, err := h.admin.ListTaskLogs(taskID, requestID, limit)
	if err != nil {
		response.Internal(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, logs)
}
