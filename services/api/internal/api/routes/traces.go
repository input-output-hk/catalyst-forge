package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
)

// TraceDeps contains dependencies for trace routes
type TraceDeps struct {
	H *handlers.TraceHandler
}

// RegisterTraces registers v1 trace routes
func RegisterTraces(r *gin.Engine, deps TraceDeps) {
	v1 := r.Group("/api/v1")
	{
		traces := v1.Group("/traces")
		{
			traces.POST("", deps.H.Create)
			traces.GET("", deps.H.List)
			traces.GET("/:trace_id", deps.H.GetByID)
		}
	}
}