package handlers

import (
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/clients/hydra"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/clients/kratos"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/mapping"
)

// buildConsentInput constructs a mapping.ConsentInput from Kratos whoami and Hydra consent request.
func buildConsentInput(whoami *kratos.WhoAmIResponse, consentReq *hydra.ConsentRequest) mapping.ConsentInput {
	kr := map[string]any{
		"identity": map[string]any{
			"id":     whoami.Identity.ID,
			"traits": whoami.Identity.Traits,
		},
	}
	hy := map[string]any{
		"consent": map[string]any{
			"requested_scope":                 consentReq.RequestedScope,
			"requested_access_token_audience": consentReq.RequestedAccessTokenAudience,
		},
	}
	return mapping.ConsentInput{Kratos: kr, Hydra: hy, Req: map[string]any{}}
}

// buildTokenHookInput constructs a mapping.TokenHookInput with minimal request context.
// Safe fields only; avoids sensitive payload echoing.
func buildTokenHookInput(jwtClaims map[string]any, c RequestContext, grantType, clientID string, audience, scopes []string) mapping.TokenHookInput {
	req := map[string]any{
		"grant_type": grantType,
	}
	if c != nil {
		if v, ok := c.Get("request_id"); ok {
			req["request_id"] = v
		}
		if ip := c.ClientIP(); ip != "" {
			req["ip"] = ip
		}
		if ua := c.GetHeader("User-Agent"); ua != "" {
			req["user_agent"] = ua
		}
	}
	if clientID != "" {
		req["client_id"] = clientID
	}
	if len(audience) > 0 {
		req["audience"] = audience
	}
	if len(scopes) > 0 {
		req["scopes"] = scopes
	}
	return mapping.TokenHookInput{JWT: jwtClaims, Req: req}
}

// RequestContext is the subset of gin.Context we use for building request info.
type RequestContext interface {
	Get(string) (any, bool)
	ClientIP() string
	GetHeader(string) string
}
