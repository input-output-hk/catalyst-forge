package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterTest wires a simple test endpoint under /api/v1 for Oathkeeper header verification.
func RegisterTest(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "hello world",
			})
		})
	}
}
