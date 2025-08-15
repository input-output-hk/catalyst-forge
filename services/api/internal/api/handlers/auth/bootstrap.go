package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	authhttp "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/httpkit"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
)

type bootstrapRequest struct {
	Email          string `json:"email" binding:"required,email"`
	BootstrapToken string `json:"bootstrap_token" binding:"required,len=32"`
}

// RegisterBootstrap binds the bootstrap endpoint.
func RegisterBootstrap(r *gin.Engine, svc akservice.BootstrapService, tokens akservice.TokenService, refresh akservice.RefreshService) {
	// @Summary Admin bootstrap
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.BootstrapRequest true "Bootstrap request"
	// @Success 201 {object} apimodels.BootstrapResponse
	// @Failure 404 {object} map[string]interface{}
	// @Router /api/v1/auth/bootstrap [post]
	r.POST("/api/v1/auth/bootstrap", func(c *gin.Context) {
		var req bootstrapRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = basehttp.NewBadRequestError("invalid request").Write(c.Writer)
			return
		}
		user, err := svc.Bootstrap(c.Request.Context(), req.BootstrapToken, req.Email)
		if err != nil {
			// Return 404 for invalid/used/disabled to avoid leaking state
			basehttp.ErrorResponse(c.Writer, http.StatusNotFound, "not_found", "resource not found")
			return
		}

		// Establish session: best-effort issue refresh cookie and access token
		var accessToken string
		if tokens != nil {
			claims := akservice.AccessClaims{Sub: user.ID.String(), Email: user.Email, Roles: user.Roles, SessionVersion: user.SessionVersion}
			if t, err := tokens.SignAccess(c.Request.Context(), claims); err == nil {
				accessToken = t
			}
		}

		if refresh != nil {
			if cookieVal, _, _, err := refresh.Issue(c.Request.Context(), user, time.Now().UTC()); err == nil {
				authhttp.SetRefreshCookie(c.Writer, cookieVal, 24*time.Hour, basehttp.DefaultCookieConfig())
			}
		}

		h := sha256.Sum256([]byte(req.BootstrapToken))
		resp := apimodels.BootstrapResponse{
			UserID:    user.ID.String(),
			Email:     user.Email,
			TokenHash: hex.EncodeToString(h[:]),
		}

		// Optionally include access token for tests
		if accessToken != "" {
			c.JSON(http.StatusCreated, gin.H{
				"user_id":      resp.UserID,
				"email":        resp.Email,
				"token_hash":   resp.TokenHash,
				"access_token": accessToken,
			})
			return
		}

		c.JSON(http.StatusCreated, resp)
	})
}
