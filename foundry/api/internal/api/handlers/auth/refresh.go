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

type RefreshDeps struct {
	CSRF    basehttp.CSRF
	Service akservice.RefreshService
	Cookie  basehttp.CookieConfig
}

func RegisterRefresh(r *gin.Engine, deps RefreshDeps) {
	g := r.Group("/api/v1/auth")

	// @Summary Refresh access token
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Success 200 {object} apimodels.AccessTokenResponse
	// @Router /api/v1/auth/refresh [post]
	g.POST("/refresh", func(c *gin.Context) {
		// Check if this is a CLI request
		isCLI := c.GetHeader("X-CLI") == "1"

		var refreshToken string

		if isCLI {
			// CLI mode: get token from Authorization header or JSON body
			// No CSRF validation required for CLI mode

			// Try Authorization header first
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && len(authHeader) > 8 && authHeader[:8] == "Refresh " {
				refreshToken = authHeader[8:]
			} else {
				// Try JSON body
				var req struct {
					RefreshToken string `json:"refresh_token"`
				}
				if err := basehttp.ParseJSON(c.Writer, c.Request, &req); err != nil {
					return
				}
				refreshToken = req.RefreshToken
			}

			if refreshToken == "" {
				_ = basehttp.NewUnauthorizedError("refresh token required").Write(c.Writer)
				return
			}
		} else {
			// Browser mode: get token from cookie with CSRF validation
			if err := deps.CSRF.Validate(c.Request); err != nil {
				_ = basehttp.NewCSRFError("").Write(c.Writer)
				return
			}

			cookie, err := authhttp.GetRefreshCookie(c.Request)
			if err != nil || cookie == "" {
				_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
				return
			}
			refreshToken = cookie
		}

		// Rotate the refresh token
		newRefreshToken, accessJWT, _, err := deps.Service.Rotate(c.Request.Context(), refreshToken, time.Now().UTC())
		if err != nil {
			_ = basehttp.NewUnauthorizedError("session expired").Write(c.Writer)
			return
		}

		if isCLI {
			// CLI mode: return both tokens in response body
			_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]interface{}{
				"access_token":  accessJWT,
				"refresh_token": newRefreshToken,
			})
		} else {
			// Browser mode: set cookie and return access token
			authhttp.SetRefreshCookie(c.Writer, newRefreshToken, 24*time.Hour, deps.Cookie)
			_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.AccessTokenResponse{AccessToken: accessJWT})
		}
	})
}
