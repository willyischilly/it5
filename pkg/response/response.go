package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func JSON(c *gin.Context, code int, data interface{}) {
	c.JSON(code, data)
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}

func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message)
}

func Internal(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, message)
}
