package handlers

import (
	"errors"

	"planner-backend/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func notFoundOrInternal(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || err.Error() == "request not found" || err.Error() == "task not found" {
		response.NotFound(c, err.Error())
		return
	}
	response.Internal(c, err.Error())
}
