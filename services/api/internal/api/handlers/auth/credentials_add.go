package auth

import (
	"encoding/json"
	"fmt"
	"net/http"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

type CredentialsAddDeps struct {
	Users    store.UserStore
	WebAuthn akservice.WebAuthnService
}

func RegisterCredentialsAdd(r *gin.Engine, deps CredentialsAddDeps) {
	g := r.Group("/api/v1/auth")

	// @Summary Add credential (begin)
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.CredentialsAddBeginRequest true "Begin credential add"
	// @Success 200 {object} apimodels.PublicKeyOptionsResponse
	// @Router /api/v1/auth/credentials/add/begin [post]
	g.POST("/credentials/add/begin", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := deps.Users.GetByID(c.Request.Context(), ctx.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		requireHW := false
		for _, r := range user.Roles {
			if r == "admin" {
				requireHW = true
				break
			}
		}
		options, sessionKey, err := deps.WebAuthn.BeginRegistration(c.Request.Context(), user, in.DeviceName, requireHW)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.PublicKeyOptionsResponse{PublicKey: options, SessionKey: sessionKey})
	})

	// @Summary Add credential (complete)
	// @Tags auth
	// @Accept json
	// @Param request body apimodels.CredentialsAddCompleteRequest true "Complete credential add"
	// @Success 204
	// @Router /api/v1/auth/credentials/add/complete [post]
	g.POST("/credentials/add/complete", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := deps.WebAuthn.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			// Include error detail during development to diagnose 400s from WebAuthn verification
			_ = basehttp.NewBadRequestError(fmt.Sprintf("registration failed: %v", err)).Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
