package middleware

import (
	"planner-backend/pkg/response"

	"github.com/gin-gonic/gin"
)

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role := GetUserRole(c)
		if !allowed[role] {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return RequireRole(roles...)
}
