package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/logging"
)

// hydraTokenHookRequest is a reduced shape of Hydra's token hook payload.
type hydraTokenHookRequest struct {
	GrantType string `json:"grant_type"`
	Request   struct {
		Payload struct {
			Assertion string `json:"assertion"`
		} `json:"payload"`
	} `json:"request"`
}

// parseUnverifiedJWT extracts claims without verifying signature (Hydra has already validated issuer).
func parseUnverifiedJWT(assertion string) map[string]any {
	parts := strings.Split(assertion, ".")
	if len(parts) != 3 {
		return nil
	}
	// Base64url decode middle part
	dec, err := jwtDecodeSegment(parts[1])
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(dec, &m); err != nil {
		return nil
	}
	return m
}

// jwtDecodeSegment is a minimal base64url decoder without padding.
func jwtDecodeSegment(seg string) ([]byte, error) {
	s := seg
	if l := len(s) % 4; l > 0 {
		s += strings.Repeat("=", 4-l)
	}
	return base64.RawURLEncoding.DecodeString(s)
}

// TokenHook handles POST /hydra/token-hook with real enrichment.
func (h *Handlers) TokenHook(c *gin.Context) {
	var req hydraTokenHookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		tokenHookFailureTotal.Inc()
		logger := logging.L()
		if v, ok := c.Get("request_id"); ok {
			logger = logger.With(slog.String("request_id", v.(string)))
		}
		logger.Warn("token_hook bind error")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "correlation": corrFields(c)})
		return
	}

	ext := map[string]any{}

	// JWT-Bearer (GitHub Actions) enrichment
	if strings.EqualFold(req.GrantType, "urn:ietf:params:oauth:grant-type:jwt-bearer") && req.Request.Payload.Assertion != "" {
		claims := parseUnverifiedJWT(req.Request.Payload.Assertion)
		if claims != nil {
			// Normalize common GH claims
			if v, ok := claims["repository"]; ok {
				ext["gh_repository"] = v
			}
			if v, ok := claims["ref"]; ok {
				ext["gh_ref"] = v
			}
			if v, ok := claims["sha"]; ok {
				ext["gh_sha"] = v
			}
			if v, ok := claims["actor"]; ok {
				ext["gh_actor"] = v
			}
			if v, ok := claims["environment"]; ok {
				ext["gh_environment"] = v
			}

			// Validate minimal claims without echoing sensitive values
			if _, ok := ext["gh_repository"]; !ok {
				tokenHookFailureTotal.Inc()
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing required claim", "correlation": corrFields(c)})
				return
			}
			if _, ok := ext["gh_ref"]; !ok {
				tokenHookFailureTotal.Inc()
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing required claim", "correlation": corrFields(c)})
				return
			}
			if _, ok := ext["gh_sha"]; !ok {
				tokenHookFailureTotal.Inc()
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing required claim", "correlation": corrFields(c)})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"session": gin.H{
			"access_token": gin.H{
				"ext": ext,
			},
		},
	})
}
