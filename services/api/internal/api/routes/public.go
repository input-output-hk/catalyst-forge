package routes

import (
	"github.com/gin-gonic/gin"
)

// RegisterPublic wires public endpoints like health.
func RegisterPublic(r *gin.Engine, healthHandler func(*gin.Context)) {
	// Healthz is public by design; global policy registry is applied centrally
	r.GET("/healthz", healthHandler)
}
