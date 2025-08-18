package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/api/handlers"
)

// OrgDeps contains dependencies for organization routes
type OrgDeps struct {
	H *handlers.OrgHandler
}

// RegisterOrgs registers admin organization routes
func RegisterOrgs(r *gin.Engine, deps OrgDeps) {
	v1 := r.Group("/api/v1")
	admin := v1.Group("/admin")
	orgs := admin.Group("/orgs")
	{
		orgs.GET("", deps.H.List)
		orgs.GET(":id", deps.H.Get)
		orgs.POST("", deps.H.Create)
		orgs.PATCH(":id/name", deps.H.UpdateName)
		orgs.PATCH(":id/default", deps.H.SetDefault)
	}
}
