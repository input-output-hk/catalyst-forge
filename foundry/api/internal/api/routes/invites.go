package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	auth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

type InviteDeps struct {
	CORS    gin.HandlerFunc
	Auth    *middleware.AuthMiddleware
	Handler *handlers.InviteHandler
}

func RegisterInvites(r *gin.Engine, d InviteDeps) {
	r.POST("/auth/invites", d.CORS, d.Auth.ValidatePermissions([]auth.Permission{auth.PermUserWrite}), d.Handler.CreateInvite)
	r.GET("/verify", d.Handler.Verify)
}
