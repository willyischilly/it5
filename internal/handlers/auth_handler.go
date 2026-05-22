package handlers

import (
	"net/http"

	"planner-backend/internal/middleware"
	"planner-backend/internal/services"
	"planner-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *services.AuthService
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var in services.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	result, err := h.auth.Register(in)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.JSON(c, http.StatusCreated, result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.auth.Me(middleware.GetUserID(c))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, user)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in services.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	result, err := h.auth.Login(in)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}
	response.JSON(c, http.StatusOK, result)
}
