package util

import (
	"strings"

	cfgpkg "github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/config"
	"github.com/input-output-hk/catalyst-forge/playground/services/mock-oidc/handlers"
)

// FirstNonEmpty returns the first non-empty string.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// FirstString returns the first element or empty.
func FirstString(values []string) string {
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// BoolOrDefault returns the bool value or default when nil.
func BoolOrDefault(ptr *bool, def bool) bool {
	if ptr == nil {
		return def
	}
	return *ptr
}

// PickPersonaSub returns the first non-empty sub, or a default.
func PickPersonaSub(personas map[string]cfgpkg.PersonaConfig) string {
	if len(personas) == 0 {
		return "00000000-0000-0000-0000-000000000001"
	}
	for _, p := range personas {
		if p.Sub != "" {
			return p.Sub
		}
	}
	return "00000000-0000-0000-0000-000000000001"
}

// PickPersonaClaimString returns the first persona claim value for the key.
func PickPersonaClaimString(personas map[string]cfgpkg.PersonaConfig, key string) string {
	for _, p := range personas {
		if v, ok := p.Claims[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}

// BuildUsersFromPersonas converts persona configs into handler TestUser map.
func BuildUsersFromPersonas(personas map[string]cfgpkg.PersonaConfig) map[string]handlers.TestUser {
	if len(personas) == 0 {
		return nil
	}
	users := make(map[string]handlers.TestUser, len(personas))
	for name, p := range personas {
		u := handlers.TestUser{Sub: p.Sub, Extra: map[string]any{}}
		if v, ok := p.Claims["email"].(string); ok {
			u.Email = v
		}
		if v, ok := p.Claims["email_verified"].(bool); ok {
			u.EmailVerified = v
		}
		if v, ok := p.Claims["name"].(string); ok {
			u.Name = v
		}
		if v, ok := p.Claims["given_name"].(string); ok {
			u.GivenName = v
		}
		if v, ok := p.Claims["family_name"].(string); ok {
			u.FamilyName = v
		}
		if v, ok := p.Claims["picture"].(string); ok {
			u.Picture = v
		}
		// Copy all persona claims into Extra (last-writer wins in merges)
		for k, v := range p.Claims {
			u.Extra[k] = v
		}
		users[name] = u
	}
	return users
}
