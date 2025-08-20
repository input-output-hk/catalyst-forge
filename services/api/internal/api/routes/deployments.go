package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
)

// DeploymentDeps contains dependencies for deployment routes
type DeploymentDeps struct {
	H *handlers.DeploymentHandler
}

// RegisterDeployments registers v1 deployment routes
func RegisterDeployments(r *gin.Engine, deps DeploymentDeps) {
	v1 := r.Group("/api/v1")
	{
		deployments := v1.Group("/deployments")
		{
			// Main CRUD operations
			deployments.POST("", deps.H.Create)
			deployments.GET("", deps.H.List)
			deployments.GET("/:deployment_id", deps.H.GetByID)
			deployments.PATCH("/:deployment_id", deps.H.Update)
			deployments.DELETE("/:deployment_id", deps.H.Delete)
		}
	}
}
