package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	auth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

type ReleaseDeps struct {
	Auth    *middleware.AuthMiddleware
	Handler *handlers.ReleaseHandler
}

func RegisterReleases(r *gin.Engine, d ReleaseDeps) {
	r.POST("/release", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), d.Handler.CreateRelease)
	r.GET("/release/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), d.Handler.GetRelease)
	r.PUT("/release/:id", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), d.Handler.UpdateRelease)
	r.GET("/releases", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), d.Handler.ListReleases)
	r.GET("/release/alias/:name", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), d.Handler.GetReleaseByAlias)
	r.POST("/release/alias/:name", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), d.Handler.CreateAlias)
	r.DELETE("/release/alias/:name", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseWrite}), d.Handler.DeleteAlias)
	r.GET("/release/:id/aliases", d.Auth.ValidatePermissions([]auth.Permission{auth.PermReleaseRead}), d.Handler.ListAliases)
}
