package resolvers

import (
	"github.com/gin-gonic/gin"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
)

// Entry defines a path pattern to resolver binding.
type Entry struct {
	Pattern  string
	Resolver rbac.ResourceResolver
}

// ProjectResolver derives a project resource reference from path or query params.
func ProjectResolver(c *gin.Context) (rbac.ResourceRef, error) {
	id := c.Param("project_id")
	if id == "" {
		id = c.Query("project_id")
	}
	return rbac.ResourceRef{Type: "project", ID: id}, nil
}

// ReleaseResolver derives a release resource reference from path or query params.
func ReleaseResolver(c *gin.Context) (rbac.ResourceRef, error) {
	id := c.Param("release_id")
	if id == "" {
		id = c.Query("release_id")
	}
	return rbac.ResourceRef{Type: "release", ID: id}, nil
}

// Registry returns all resolver bindings for the application.
func Registry() []Entry {
	return []Entry{
		{Pattern: "/api/v1/projects/:project_id", Resolver: ProjectResolver},
		{Pattern: "/api/v1/releases/:release_id", Resolver: ReleaseResolver},
	}
}
