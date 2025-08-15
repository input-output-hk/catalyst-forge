package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/crypto"
)

// AccessClaims represents the claims in an access token JWT.
type AccessClaims struct {
	Sub            string   `json:"sub"`                   // User ID
	Email          string   `json:"email"`                 // User email
	Roles          []string `json:"roles,omitempty"`       // User roles
	Permissions    []string `json:"permissions,omitempty"` // User permissions
	SessionVersion int64    `json:"sv"`                    // Session version for invalidation
	JTI            string   `json:"jti"`                   // JWT ID for tracking
	Issuer         string   `json:"iss"`                   // Token issuer
	Audience       string   `json:"aud"`                   // Token audience
	IssuedAt       int64    `json:"iat"`                   // Unix timestamp
	ExpiresAt      int64    `json:"exp"`                   // Unix timestamp
	StepUpUntil    int64    `json:"su,omitempty"`          // Step-up valid until (Unix timestamp)
	AMR            []string `json:"amr,omitempty"`         // Authentication Methods References (e.g., ["webauthn"], ["device_link"])
	DeviceID       string   `json:"device_id,omitempty"`   // Device ID for CLI sessions
}

// TokenService handles JWT token operations.
type TokenService interface {
	// SignAccess creates and signs an access token with the given claims.
	SignAccess(ctx context.Context, claims AccessClaims) (string, error)

	// ParseAccess parses and validates an access token.
	ParseAccess(ctx context.Context, token string) (*AccessClaims, error)

	// JWKS returns the JSON Web Key Set for public key verification.
	JWKS() interface{}
}

// tokenService implements TokenService.
type tokenService struct {
	keys   crypto.KeyManager
	rand   crypto.Rand
	issuer string
	ttl    time.Duration
}

const (
	// clockSkewLeeway is the allowed clock skew for token validation (60 seconds).
	clockSkewLeeway = 60 * time.Second
)

// NewTokenService creates a new token service.
func NewTokenService(keys crypto.KeyManager, rand crypto.Rand, issuer string, ttl time.Duration) TokenService {
	if ttl <= 0 {
		panic("token TTL must be > 0")
	}
	return &tokenService{
		keys:   keys,
		rand:   rand,
		issuer: issuer,
		ttl:    ttl,
	}
}

// SignAccess creates and signs an access token with the given claims.
func (s *tokenService) SignAccess(ctx context.Context, claims AccessClaims) (string, error) {
	now := time.Now().UTC()

	// Require session version to be set by caller
	if claims.SessionVersion == 0 {
		return "", errors.New("session version is required")
	}

	// Set standard claims if not already set
	if claims.JTI == "" {
		jti, err := s.rand.String(16)
		if err != nil {
			return "", err
		}
		claims.JTI = jti
	}

	if claims.Issuer == "" {
		claims.Issuer = s.issuer
	}

	if claims.Audience == "" {
		claims.Audience = s.issuer
	}

	if claims.IssuedAt == 0 {
		claims.IssuedAt = now.Unix()
	}

	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = now.Add(s.ttl).Unix()
	}

	// Convert claims to JSON for signing
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	// Sign the token
	token, err := s.keys.SignJWT(ctx, claimsJSON, s.keys.CurrentKID())
	if err != nil {
		return "", err
	}

	return token, nil
}

// ParseAccess parses and validates an access token.
func (s *tokenService) ParseAccess(ctx context.Context, token string) (*AccessClaims, error) {
	// Verify and parse the token
	claimsMap, err := s.keys.VerifyJWT(ctx, token)
	if err != nil {
		return nil, err
	}

	// Convert map to AccessClaims
	claims := &AccessClaims{}

	// Parse standard claims
	if sub, ok := claimsMap["sub"].(string); ok {
		claims.Sub = sub
	} else {
		return nil, errors.New("missing subject claim")
	}

	if email, ok := claimsMap["email"].(string); ok {
		claims.Email = email
	}

	// Parse arrays
	if roles, ok := claimsMap["roles"].([]interface{}); ok {
		claims.Roles = make([]string, len(roles))
		for i, role := range roles {
			if r, ok := role.(string); ok {
				claims.Roles[i] = r
			}
		}
	}

	if permissions, ok := claimsMap["permissions"].([]interface{}); ok {
		claims.Permissions = make([]string, len(permissions))
		for i, perm := range permissions {
			if p, ok := perm.(string); ok {
				claims.Permissions[i] = p
			}
		}
	}

	// Parse numeric claims
	if sv, ok := claimsMap["sv"].(float64); ok {
		claims.SessionVersion = int64(sv)
	}

	if jti, ok := claimsMap["jti"].(string); ok {
		claims.JTI = jti
	}

	if iss, ok := claimsMap["iss"].(string); ok {
		claims.Issuer = iss
	}

	// Handle audience as string or array
	var foundAudience bool
	switch v := claimsMap["aud"].(type) {
	case string:
		if v == s.issuer {
			claims.Audience = v
			foundAudience = true
		}
	case []interface{}:
		// Check if our expected audience is in the array
		for _, aud := range v {
			if audStr, ok := aud.(string); ok && audStr == s.issuer {
				claims.Audience = audStr
				foundAudience = true
				break
			}
		}
	}

	if !foundAudience {
		return nil, errors.New("token audience does not match expected audience")
	}

	if iat, ok := claimsMap["iat"].(float64); ok {
		claims.IssuedAt = int64(iat)
	}

	if exp, ok := claimsMap["exp"].(float64); ok {
		claims.ExpiresAt = int64(exp)
	}

	if su, ok := claimsMap["su"].(float64); ok {
		claims.StepUpUntil = int64(su)
	}

	// Parse AMR (Authentication Methods References)
	if amr, ok := claimsMap["amr"].([]interface{}); ok {
		claims.AMR = make([]string, len(amr))
		for i, method := range amr {
			if m, ok := method.(string); ok {
				claims.AMR[i] = m
			}
		}
	}

	// Parse device ID for CLI sessions
	if deviceID, ok := claimsMap["device_id"].(string); ok {
		claims.DeviceID = deviceID
	}

	// Validate timing claims with clock skew tolerance
	now := time.Now().UTC()

	// Check not-before (issued-at with leeway)
	if claims.IssuedAt > 0 {
		iatTime := time.Unix(claims.IssuedAt, 0)
		if iatTime.After(now.Add(clockSkewLeeway)) {
			return nil, errors.New("token not yet valid (iat in future)")
		}
	}

	// Validate expiration
	if claims.ExpiresAt > 0 && time.Unix(claims.ExpiresAt, 0).Before(now.Add(-clockSkewLeeway)) {
		return nil, errors.New("token expired")
	}

	// Validate issuer
	if claims.Issuer != s.issuer {
		return nil, errors.New("invalid issuer")
	}

	// Require session version presence
	if claims.SessionVersion == 0 {
		return nil, errors.New("missing session version")
	}

	return claims, nil
}

// JWKS returns the JSON Web Key Set for public key verification.
func (s *tokenService) JWKS() interface{} {
	return s.keys.JWKS()
}

// ValidateAccessToken is a helper to validate an access token without parsing.
func ValidateAccessToken(ctx context.Context, service TokenService, token string) error {
	_, err := service.ParseAccess(ctx, token)
	return err
}
