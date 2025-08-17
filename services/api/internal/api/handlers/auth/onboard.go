package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	apiauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	akcrypto "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	authhttp "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/httpkit"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
	akstore "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

type OnboardDeps struct {
	Invites akstore.InviteStore
	Users   akstore.UserStore
	Refresh akservice.RefreshService
	Tokens  akservice.TokenService
}

// RegisterOnboarding binds onboarding begin/complete
func RegisterOnboarding(r *gin.Engine, deps OnboardDeps, wa akservice.WebAuthnService, cookieCfg basehttp.CookieConfig, refreshTTL time.Duration) {
	g := r.Group("/api/v1/auth")

	// @Summary Onboard begin
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.OnboardBeginRequest true "Onboard begin request"
	// @Success 200 {object} apimodels.OnboardBeginResponse
	// @Router /api/v1/auth/onboard/begin [post]
	g.POST("/onboard/begin", func(c *gin.Context) {
		var in struct {
			InviteID   string `json:"invite_id"`
			Token      string `json:"token"`
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		invID, err := uuid.Parse(in.InviteID)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid invite id").Write(c.Writer)
			return
		}
		inv, err := deps.Invites.Get(c.Request.Context(), invID)
		if err != nil || inv == nil {
			_ = basehttp.NewBadRequestError("invalid invite").Write(c.Writer)
			return
		}
		if time.Now().UTC().After(inv.ExpiresAt) {
			_ = basehttp.NewBadRequestError("invite expired").Write(c.Writer)
			return
		}
		tokBytes, err := base64.RawURLEncoding.DecodeString(in.Token)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid token").Write(c.Writer)
			return
		}
		if subtle.ConstantTimeCompare(inv.TokenHash, akcrypto.HashSHA256(tokBytes)) != 1 {
			_ = deps.Invites.IncrementAttempts(c.Request.Context(), inv.ID)
			_ = basehttp.NewBadRequestError("invalid invite").Write(c.Writer)
			return
		}
		// Idempotent user provisioning: reuse if already created in a previous attempt
		user, getErr := deps.Users.GetByEmail(c.Request.Context(), inv.Email)
		if getErr != nil || user == nil {
			if _, createErr := deps.Users.Create(c.Request.Context(), inv.Email, inv.Roles); createErr != nil {
				// Attempt to read again in case of unique constraint race
				user, getErr = deps.Users.GetByEmail(c.Request.Context(), inv.Email)
				if getErr != nil || user == nil {
					_ = basehttp.NewInternalError().Write(c.Writer)
					return
				}
			} else {
				// Created; fetch to ensure we have full domain entity
				user, _ = deps.Users.GetByEmail(c.Request.Context(), inv.Email)
			}
		}
		requireHW := sliceContains(inv.Roles, "admin")
		options, sessionKey, err := wa.BeginRegistration(c.Request.Context(), user, in.DeviceName, requireHW)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.OnboardBeginResponse{PublicKey: options, SessionKey: sessionKey, UserID: user.ID.String()})
	})

	// @Summary Onboard complete
	// @Tags auth
	// @Accept json
	// @Param request body apimodels.OnboardCompleteRequest true "Onboard complete request"
	// @Success 204
	// @Router /api/v1/auth/onboard/complete [post]
	g.POST("/onboard/complete", func(c *gin.Context) {
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
			InviteID   string          `json:"invite_id"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		cred, err := wa.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		// Mark invite as redeemed (best-effort)
		if in.InviteID != "" {
			if invID, err := uuid.Parse(in.InviteID); err == nil {
				_ = deps.Invites.Redeem(c.Request.Context(), invID, time.Now().UTC())
			}
		}
		// Establish session: set refresh cookie and CSRF cookie
		if cred != nil {
			if user, uerr := deps.Users.GetByID(c.Request.Context(), cred.UserID); uerr == nil && user != nil {
				if cookieValue, _, _, ierr := deps.Refresh.Issue(c.Request.Context(), user, time.Now().UTC()); ierr == nil {
					authhttp.SetRefreshCookie(c.Writer, cookieValue, refreshTTL, cookieCfg)
				}
			}
		}
		if csrf := apiauth.BuildDepsCached().CSRF; csrf != nil {
			if tok, terr := csrf.Generate(); terr == nil {
				csrf.SetCookie(c.Writer, tok)
			}
		}
		c.Status(http.StatusNoContent)
	})
}

func sliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
