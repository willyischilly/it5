package middleware

import (
	"strings"

	"planner-backend/internal/services"
	"planner-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

const ContextUserID = "userID"
const ContextUserRole = "userRole"
const ContextUserEmail = "userEmail"

func Auth(auth *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			response.Unauthorized(c, "authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Unauthorized(c, "invalid authorization header")
			c.Abort()
			return
		}

		claims, err := auth.ParseToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserRole, claims.Role)
		c.Set(ContextUserEmail, claims.Email)
		c.Next()
	}
}

func GetUserID(c *gin.Context) uint {
	id, _ := c.Get(ContextUserID)
	return id.(uint)
}

func GetUserRole(c *gin.Context) string {
	role, _ := c.Get(ContextUserRole)
	return role.(string)
}
