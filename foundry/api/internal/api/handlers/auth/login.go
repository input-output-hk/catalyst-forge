package auth

import (
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/models/auth"
	authhttp "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/httpkit"
	akservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/service"
)

// RegisterLoginBegin binds POST /api/v1/auth/login/begin
func RegisterLoginBegin(r *gin.Engine, wa akservice.WebAuthnService) {
	g := r.Group("/api/v1/auth")

	// @Summary Begin login
	// @Tags auth
	// @Produce json
	// @Success 200 {object} apimodels.PublicKeyOptionsResponse
	// @Router /api/v1/auth/login/begin [post]
	g.POST("/login/begin", func(c *gin.Context) {
		options, sessionKey, err := wa.BeginLogin(c.Request.Context(), "")
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.PublicKeyOptionsResponse{PublicKey: options, SessionKey: sessionKey})
	})
}

// RegisterLoginComplete binds POST /api/v1/auth/login/complete
func RegisterLoginComplete(r *gin.Engine, wa akservice.WebAuthnService, refresh akservice.RefreshService, tokens akservice.TokenService, cookieCfg basehttp.CookieConfig, refreshTTL time.Duration) {
	g := r.Group("/api/v1/auth")

	// @Summary Complete login
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.LoginCompleteRequest true "Login complete request"
	// @Success 200 {object} apimodels.LoginCompleteResponse
	// @Router /api/v1/auth/login/complete [post]
	g.POST("/login/complete", func(c *gin.Context) {
		var in struct {
			SessionKey string      `json:"session_key"`
			Credential interface{} `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, _, err := wa.FinishLogin(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("authentication failed").Write(c.Writer)
			return
		}
		// Issue refresh cookie
		cookieValue, _, _, err := refresh.Issue(c.Request.Context(), user, time.Now().UTC())
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		authhttp.SetRefreshCookie(c.Writer, cookieValue, refreshTTL, cookieCfg)

		// Issue access token with AMR for WebAuthn
		claims := akservice.AccessClaims{
			Sub:            user.ID.String(),
			Email:          user.Email,
			Roles:          user.Roles,
			SessionVersion: user.SessionVersion,
			AMR:            []string{"webauthn"}, // Authentication method reference
		}
		access, err := tokens.SignAccess(c.Request.Context(), claims)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.LoginCompleteResponse{User: apimodels.UserSummary{ID: user.ID.String(), Email: user.Email, Roles: user.Roles}, AccessToken: access})
	})
}
