package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
)

// ArtifactDeps contains dependencies for artifact routes
type ArtifactDeps struct {
	H *handlers.ArtifactHandler
}

// RegisterArtifacts registers v1 artifact routes
func RegisterArtifacts(r *gin.Engine, deps ArtifactDeps) {
	v1 := r.Group("/api/v1")
	{
		artifacts := v1.Group("/artifacts")
		{
			artifacts.POST("", deps.H.Create)
			artifacts.GET("", deps.H.List)
			artifacts.GET("/:artifact_id", deps.H.GetByID)
			artifacts.PATCH("/:artifact_id", deps.H.Update)
			artifacts.DELETE("/:artifact_id", deps.H.Delete)
		}
	}
}