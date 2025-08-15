package auth

import (
    "github.com/gin-gonic/gin"
    akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
    akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
)

type LogoutAllDeps struct {
    Service akservice.RefreshService
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
        c.Status(204)
    })
}
