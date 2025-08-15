package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
)

// EnvironmentDeps contains dependencies for environment routes
type EnvironmentDeps struct {
	H *handlers.EnvironmentHandler
}

// RegisterEnvironments registers v1 environment routes
func RegisterEnvironments(r *gin.Engine, deps EnvironmentDeps) {
	v1 := r.Group("/api/v1")
	{
		environments := v1.Group("/environments")
		{
			environments.POST("", deps.H.Create)
			environments.GET("", deps.H.List)
			environments.GET("/:environment_id", deps.H.GetByID)
			environments.PATCH("/:environment_id", deps.H.Update)
			environments.DELETE("/:environment_id", deps.H.Delete)
		}

		// Project-scoped environment lookup by name
		projects := v1.Group("/projects")
		{
			projects.GET("/:project_id/environments/:name", deps.H.GetByProjectAndName)
		}
	}
}
