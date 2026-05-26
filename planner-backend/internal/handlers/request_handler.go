package handlers

import (
	"net/http"

	"planner-backend/internal/middleware"
	"planner-backend/internal/models"
	"planner-backend/internal/services"
	"planner-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type RequestHandler struct {
	customer *services.CustomerService
	executor *services.ExecutorService
}

func NewRequestHandler(customer *services.CustomerService, executor *services.ExecutorService) *RequestHandler {
	return &RequestHandler{customer: customer, executor: executor}
}

func (h *RequestHandler) GetRequest(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	userID := middleware.GetUserID(c)
	role := middleware.GetUserRole(c)

	switch role {
	case models.RoleCustomer:
		req, err := h.customer.GetRequest(userID, requestID)
		if err != nil {
			response.NotFound(c, err.Error())
			return
		}
		response.JSON(c, http.StatusOK, req)
	case models.RoleExecutor:
		if err := h.executor.EnsureRequestAccess(userID, requestID); err != nil {
			response.Forbidden(c, err.Error())
			return
		}
		req, err := h.customer.ViewRequestByID(requestID)
		if err != nil {
			response.NotFound(c, err.Error())
			return
		}
		response.JSON(c, http.StatusOK, req)
	default:
		response.Forbidden(c, "insufficient permissions")
	}
}
