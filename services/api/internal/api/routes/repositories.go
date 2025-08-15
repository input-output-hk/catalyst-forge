package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
)

// RepositoryDeps contains dependencies for repository routes
type RepositoryDeps struct {
	H *handlers.RepositoryHandler
}

// RegisterRepositories registers v1 repository routes (read-only)
func RegisterRepositories(r *gin.Engine, deps RepositoryDeps) {
	v1 := r.Group("/api/v1")
	{
		repositories := v1.Group("/repositories")
		{
			repositories.GET("", deps.H.List)
			repositories.GET("/by-path/:host/:org/:name", deps.H.GetByPath)
			repositories.GET("/:repo_id", deps.H.GetByID)
		}
	}
}