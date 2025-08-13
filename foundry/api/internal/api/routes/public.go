package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterPublic wires public endpoints like health.
func RegisterPublic(r *gin.Engine, healthHandler func(*gin.Context)) {
	r.GET("/healthz", healthHandler)
}
