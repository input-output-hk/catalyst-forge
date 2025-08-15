package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
)

// BuildDeps contains dependencies for build routes
type BuildDeps struct {
	H *handlers.BuildHandler
}

// RegisterBuilds registers v1 build routes
func RegisterBuilds(r *gin.Engine, deps BuildDeps) {
	v1 := r.Group("/api/v1")
	{
		builds := v1.Group("/builds")
		{
			builds.POST("", deps.H.Create)
			builds.GET("", deps.H.List)
			builds.GET("/:build_id", deps.H.GetByID)
			builds.PATCH("/:build_id", deps.H.Update)
		}
	}
}