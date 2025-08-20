package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
)

// RenderedReleaseDeps contains dependencies for rendered release routes
type RenderedReleaseDeps struct {
	H *handlers.RenderedReleaseHandler
}

// RegisterRenderedReleases registers v1 rendered release routes
func RegisterRenderedReleases(r *gin.Engine, deps RenderedReleaseDeps) {
	v1 := r.Group("/api/v1")
	{
		rendered := v1.Group("/rendered-releases")
		{
			rendered.POST("", deps.H.Create)
			rendered.GET("", deps.H.List)
			rendered.GET("/:rendered_release_id", deps.H.GetByID)
			rendered.PATCH("/:rendered_release_id", deps.H.Update)
			rendered.DELETE("/:rendered_release_id", deps.H.Delete)
		}

		// convenience route by deployment
		deployments := v1.Group("/deployments")
		{
			deployments.GET("/:deployment_id/rendered-release", deps.H.GetByDeployment)
		}
	}
}
