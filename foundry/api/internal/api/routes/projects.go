package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
)

// ProjectDeps contains dependencies for project routes
type ProjectDeps struct {
	H *handlers.ProjectHandler
}

// RegisterProjects registers v1 project routes (read-only)
func RegisterProjects(r *gin.Engine, deps ProjectDeps) {
	v1 := r.Group("/api/v1")
	{
		projects := v1.Group("/projects")
		{
			projects.GET("", deps.H.List)
			projects.GET("/:project_id", deps.H.GetByID)
		}

		// Repository-scoped project route
		repositories := v1.Group("/repositories")
		{
			repositories.GET("/:repo_id/projects/by-path", deps.H.GetByRepoAndPath)
		}
	}
}