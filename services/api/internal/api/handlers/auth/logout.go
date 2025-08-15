package auth

import (
	"github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	authhttp "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/httpkit"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
)

type LogoutDeps struct {
	Service akservice.RefreshService
	Cookie  httpkit.CookieConfig
}

func RegisterLogout(r *gin.Engine, deps LogoutDeps) {
	g := r.Group("/api/v1/auth")

	// @Summary Logout current session
	// @Tags auth
	// @Success 204
	// @Router /api/v1/auth/logout [post]
	g.POST("/logout", func(c *gin.Context) {
		cookie, err := authhttp.GetRefreshCookie(c.Request)
		if err == nil && cookie != "" {
			_ = deps.Service.RevokeCurrent(c.Request.Context(), cookie, "user logout")
		}
		authhttp.ClearRefreshCookie(c.Writer, deps.Cookie)
		c.Status(204)
	})
}
