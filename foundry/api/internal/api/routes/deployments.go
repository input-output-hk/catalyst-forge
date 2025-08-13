package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	auth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

type DeploymentDeps struct {
	Auth    *middleware.AuthMiddleware
	Handler *handlers.DeploymentHandler
}

func RegisterDeployments(r *gin.Engine, d DeploymentDeps) {
	r.POST("/release/:id/deploy", d.Auth.ValidatePermissions([]auth.Permission{auth.PermDeploymentWrite}), d.Handler.CreateDeployment)
	r.GET("/release/:id/deploy/:deployId", d.Auth.ValidatePermissions([]auth.Permission{auth.PermDeploymentRead}), d.Handler.GetDeployment)
	r.PUT("/release/:id/deploy/:deployId", d.Auth.ValidatePermissions([]auth.Permission{auth.PermDeploymentWrite}), d.Handler.UpdateDeployment)
	r.GET("/release/:id/deployments", d.Auth.ValidatePermissions([]auth.Permission{auth.PermDeploymentRead}), d.Handler.ListDeployments)
	r.GET("/release/:id/deploy/latest", d.Auth.ValidatePermissions([]auth.Permission{auth.PermDeploymentRead}), d.Handler.GetLatestDeployment)
	r.POST("/release/:id/deploy/:deployId/events", d.Auth.ValidatePermissions([]auth.Permission{auth.PermDeploymentEventWrite}), d.Handler.AddDeploymentEvent)
	r.GET("/release/:id/deploy/:deployId/events", d.Auth.ValidatePermissions([]auth.Permission{auth.PermDeploymentEventRead}), d.Handler.GetDeploymentEvents)
}
