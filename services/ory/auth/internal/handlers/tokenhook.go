package handlers

import (
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"strings"

	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/logging"
)

// hydraTokenHookRequest is a reduced shape of Hydra's token hook payload.
type hydraTokenHookRequest struct {
	Request struct {
		ClientID        string   `json:"client_id"`
		GrantedScopes   []string `json:"granted_scopes"`
		GrantedAudience []string `json:"granted_audience"`
		GrantTypes      []string `json:"grant_types"`
		Payload         struct {
			Assertion json.RawMessage `json:"assertion"`
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

func extractAssertion(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil && len(arr) > 0 {
		return arr[0]
	}
	return ""
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
		logger.Warn("token_hook bind error", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload", "correlation": corrFields(c)})
		return
	}

	ext := map[string]any{}

	// JWT-Bearer enrichment
	assertion := extractAssertion(req.Request.Payload.Assertion)
	if assertion != "" {
		claims := parseUnverifiedJWT(assertion)
		if claims != nil {
			if h.mapper != nil {
				var grantTypeForInput string

				audience := req.Request.GrantedAudience

				scopes := req.Request.GrantedScopes

				if len(req.Request.GrantTypes) > 0 {
					grantTypeForInput = req.Request.GrantTypes[0]
				}
				input := buildTokenHookInput(
					claims,
					c,
					grantTypeForInput,
					req.Request.ClientID,
					audience,
					scopes,
				)
				// Issuer-scoped evaluation
				iss, _ := claims["iss"].(string)
				mapped, err := h.mapper.EvaluateTokenHookByIssuer(iss, input)
				if err != nil {
					// Unknown issuer or evaluation error
					if strings.EqualFold(h.cfg.Mapping.OnError, "deny") {
						tokenHookFailureTotal.Inc()
						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "mapping error", "correlation": corrFields(c)})
						return
					}
					logging.L().Warn("token_hook mapping error; proceeding without claims", slog.String("error", err.Error()))
				} else {
					maps.Copy(ext, mapped)
				}
			}
		}
	}

	// If no enrichment is present, avoid overriding consent-provided claims.
	if len(ext) == 0 {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": gin.H{
			"access_token": ext,
		},
	})
}
