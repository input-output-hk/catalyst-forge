package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apimodels "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/models/auth"
	akcrypto "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/crypto"
	akservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/service"
	akstore "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
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
		user, err := deps.Users.Create(c.Request.Context(), inv.Email, inv.Roles)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
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
			SessionKey string      `json:"session_key"`
			Credential interface{} `json:"credential"`
			InviteID   string      `json:"invite_id"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := wa.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		if in.InviteID != "" {
			if invID, err := uuid.Parse(in.InviteID); err == nil {
				_ = deps.Invites.Redeem(c.Request.Context(), invID, time.Now().UTC())
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
