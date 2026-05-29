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

func (h *RequestHandler) ListRequests(c *gin.Context) {
	role := middleware.GetUserRole(c)
	userID := middleware.GetUserID(c)

	switch role {
	case models.RoleCustomer:
		reqs, err := h.customer.ListRequests(userID)
		if err != nil {
			response.Internal(c, err.Error())
			return
		}
		response.JSON(c, http.StatusOK, reqs)
	case models.RoleExecutor:
		reqs, err := h.executor.ListRequests()
		if err != nil {
			response.Internal(c, err.Error())
			return
		}
		response.JSON(c, http.StatusOK, reqs)
	default:
		response.Forbidden(c, "insufficient permissions")
	}
}

func (h *RequestHandler) GetRequest(c *gin.Context) {
	requestID, ok := parseRequestID(c)
	if !ok {
		return
	}

	role := middleware.GetUserRole(c)

	switch role {
	case models.RoleCustomer:
		req, err := h.customer.GetRequest(middleware.GetUserID(c), requestID)
		if err != nil {
			response.NotFound(c, err.Error())
			return
		}
		response.JSON(c, http.StatusOK, req)
	case models.RoleExecutor:
		req, err := h.executor.GetRequest(requestID)
		if err != nil {
			response.NotFound(c, err.Error())
			return
		}
		response.JSON(c, http.StatusOK, req)
	default:
		response.Forbidden(c, "insufficient permissions")
	}
}
