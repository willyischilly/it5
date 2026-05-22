package app

import (
	"net/http"

	"planner-backend/internal/app/static"

	"github.com/gin-gonic/gin"
)

func registerDocs(r *gin.Engine) {
	r.GET("/api/openapi.yaml", serveEmbedded("openapi.yaml", "application/yaml; charset=utf-8"))
	r.GET("/swagger", serveEmbedded("docs.html", "text/html; charset=utf-8"))
	r.GET("/swagger/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger")
	})
}

func serveEmbedded(name, contentType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := static.FS.ReadFile(name)
		if err != nil {
			c.String(http.StatusNotFound, "not found")
			return
		}
		c.Data(http.StatusOK, contentType, data)
	}
}
