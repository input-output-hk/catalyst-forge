package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterTest registers a simple test endpoint under /api/v1 that dumps request headers.
func RegisterTest(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"headers": c.Request.Header,
			})
		})
	}
}
