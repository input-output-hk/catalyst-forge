package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	authhttp "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/httpkit"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
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
func RegisterLoginComplete(r *gin.Engine, wa akservice.WebAuthnService, refresh akservice.RefreshService, tokens akservice.TokenService, cookieCfg basehttp.CookieConfig, refreshTTL time.Duration, csrf basehttp.CSRF) {
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
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, _, err := wa.FinishLogin(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("authentication failed").Write(c.Writer)
			return
		}
		// Reject disabled accounts
		if user.SuspendedAt != nil {
			_ = basehttp.NewForbiddenError("account is disabled").Write(c.Writer)
			return
		}
		// Issue refresh cookie
		ctx := context.WithValue(c.Request.Context(), "client_ua", c.Request.UserAgent())
		ctx = context.WithValue(ctx, "client_ip", c.ClientIP())
		cookieValue, _, _, err := refresh.Issue(ctx, user, time.Now().UTC())
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		authhttp.SetRefreshCookie(c.Writer, cookieValue, refreshTTL, cookieCfg)

		// Ensure browser has a valid CSRF cookie immediately after login
		if csrf != nil {
			if tok, err := csrf.Generate(); err == nil {
				csrf.SetCookie(c.Writer, tok)
			}
		}

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
		// Set access cookie for browser flows to simplify frontend; TTL based on token service config
		// We don't have direct access to TTL, so set cookie expiry to match refresh cookie's window best-effort if shorter tokens are used.
		// Using 30 minutes as a sane default matches default AccessTokenTTL.
		authhttp.SetAccessCookie(c.Writer, access, 30*time.Minute, cookieCfg)
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.LoginCompleteResponse{User: apimodels.UserSummary{ID: user.ID.String(), Email: user.Email, FullName: user.FullName, Roles: user.Roles}, AccessToken: access})
	})
}
