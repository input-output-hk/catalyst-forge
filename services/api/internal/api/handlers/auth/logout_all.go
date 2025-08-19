package auth

import (
	"github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	authhttp "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/httpkit"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
)

type LogoutAllDeps struct {
	Service akservice.RefreshService
	Cookie  httpkit.CookieConfig
}

func RegisterLogoutAll(r *gin.Engine, deps LogoutAllDeps) {
	g := r.Group("/api/v1/auth")

	// @Summary Logout all sessions
	// @Tags auth
	// @Success 204
	// @Router /api/v1/auth/logout-all [post]
	g.POST("/logout-all", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			c.Status(401)
			return
		}
		_ = deps.Service.RevokeAllUser(c.Request.Context(), ctx.UserID, "user logout all")

		authhttp.ClearRefreshCookie(c.Writer, deps.Cookie)
		authhttp.ClearAccessCookie(c.Writer, deps.Cookie)
		c.Status(204)
	})
}
