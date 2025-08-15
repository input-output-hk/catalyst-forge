package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
)

// ReleaseDeps contains dependencies for release routes
type ReleaseDeps struct {
	H *handlers.ReleaseHandler
}

// RegisterReleases registers v1 release routes
func RegisterReleases(r *gin.Engine, deps ReleaseDeps) {
	v1 := r.Group("/api/v1")
	{
		releases := v1.Group("/releases")
		{
			// Main CRUD operations
			releases.POST("", deps.H.Create)
			releases.GET("", deps.H.List)
			releases.GET("/:release_id", deps.H.GetByID)
			releases.PATCH("/:release_id", deps.H.Update)
			releases.DELETE("/:release_id", deps.H.Delete)

			// Module sub-resources
			releases.GET("/:release_id/modules", deps.H.GetModules)
			releases.POST("/:release_id/modules", deps.H.AddModules)
			releases.DELETE("/:release_id/modules/:module_key", deps.H.RemoveModule)

			// Injection sub-resources
			releases.GET("/:release_id/injections", deps.H.GetInjections)
			releases.POST("/:release_id/injections", deps.H.AddInjections)
			releases.DELETE("/:release_id/injections/:injection_id", deps.H.RemoveInjection)

			// Artifact sub-resources
			releases.GET("/:release_id/artifacts", deps.H.GetArtifacts)
			releases.POST("/:release_id/artifacts", deps.H.AttachArtifact)
			releases.DELETE("/:release_id/artifacts/:artifact_id/:role", deps.H.DetachArtifact)
		}
	}
}