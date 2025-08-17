package auth

import (
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

func RegisterMeSession(r *gin.Engine, users store.UserStore) {
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
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.MeResponse{ID: ctx.UserID.String(), Email: ctx.Email, FullName: ctx.FullName, Roles: ctx.Roles})
	})

	// @Summary Update profile (full name)
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Router /api/v1/auth/me [patch]
	g.PATCH("/me", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			FullName string `json:"full_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if users == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		if err := users.UpdateFullName(c.Request.Context(), ctx.UserID, in.FullName); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
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
