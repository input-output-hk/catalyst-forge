package auth

import (
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/models/auth"
	akauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
)

func RegisterMeSession(r *gin.Engine) {
	g := r.Group("/api/v1/auth")

	// @Summary Me
	// @Tags auth
	// @Produce json
	// @Success 200 {object} apimodels.MeResponse
	// @Router /api/v1/auth/me [get]
	g.GET("/me", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.MeResponse{ID: ctx.UserID.String(), Email: ctx.Email, Roles: ctx.Roles})
	})

	// @Summary Session
	// @Tags auth
	// @Produce json
	// @Success 200 {object} apimodels.SessionResponse
	// @Router /api/v1/auth/session [get]
	g.GET("/session", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.SessionResponse{Valid: true, SessionVersion: ctx.SessionVersion, StepUpRequired: ctx.RequiresStepUp(time.Now().UTC()), StepUpUntil: ctx.StepUpValidUntil})
	})
}
