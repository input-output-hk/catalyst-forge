package auth

import (
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/models/auth"
	akauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
	akservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/service"
	akstore "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

// RegisterStepUpBegin binds POST /api/v1/auth/step-up/begin
func RegisterStepUpBegin(r *gin.Engine, wa akservice.WebAuthnService, users akstore.UserStore) {
	g := r.Group("/api/v1/auth")

	// @Summary Step-up (begin)
	// @Tags auth
	// @Produce json
	// @Success 200 {object} apimodels.PublicKeyOptionsResponse
	// @Router /api/v1/auth/step-up/begin [post]
	g.POST("/step-up/begin", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		user, err := users.GetByID(c.Request.Context(), ctx.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		options, sessionKey, err := wa.BeginStepUp(c.Request.Context(), user, "")
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.PublicKeyOptionsResponse{PublicKey: options, SessionKey: sessionKey})
	})
}

// RegisterStepUpComplete binds POST /api/v1/auth/step-up/complete
func RegisterStepUpComplete(r *gin.Engine, wa akservice.WebAuthnService, tokens akservice.TokenService, stepUpTTL time.Duration) {
	g := r.Group("/api/v1/auth")

	// @Summary Step-up (complete)
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.StepUpCompleteRequest true "Step-up complete request"
	// @Success 200 {object} apimodels.AccessTokenResponse
	// @Router /api/v1/auth/step-up/complete [post]
	g.POST("/step-up/complete", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			SessionKey string      `json:"session_key"`
			Credential interface{} `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := wa.FinishStepUp(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("step-up failed").Write(c.Writer)
			return
		}
		until := time.Now().UTC().Add(stepUpTTL)
		claims := akservice.AccessClaims{Sub: user.ID.String(), Email: user.Email, Roles: user.Roles, SessionVersion: user.SessionVersion, StepUpUntil: until.Unix()}
		access, err := tokens.SignAccess(c.Request.Context(), claims)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.AccessTokenResponse{AccessToken: access})
	})
}
