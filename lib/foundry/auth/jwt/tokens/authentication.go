package tokens

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	foundryJWT "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
)

// GenerateAuthToken creates a new authentication JWT token for the given user
// The expiration is capped by the signer's MaxAuthTokenTTL
func GenerateAuthToken(
	signer foundryJWT.JWTSigner,
	subject string,
	permissions []auth.Permission,
	expiration time.Duration,
	opts ...foundryJWT.TokenOption,
) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("signer cannot be nil")
	}

	if subject == "" {
		return "", fmt.Errorf("subject cannot be empty")
	}

	if expiration <= 0 {
		return "", fmt.Errorf("expiration must be positive")
	}

	// Validate expiration against signer's max
	maxTTL := signer.MaxAuthTokenTTL()
	if maxTTL > 0 && expiration > maxTTL {
		// Cap the expiration at the maximum allowed
		expiration = maxTTL
	}

	// Apply token options
	options := &foundryJWT.TokenOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Build claims
	now := time.Now()
	claims := &AuthClaims{
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    getOrDefault(options.Issuer, signer.Issuer()),
			Audience:  getOrDefaultSlice(options.Audiences, signer.DefaultAudiences()),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        options.ID,
		},
	}

	if options.AdditionalClaims != nil {
		if v, ok := options.AdditionalClaims["akid"]; ok {
			if s, ok2 := v.(string); ok2 {
				claims.AKID = s
			}
		}
		if v, ok := options.AdditionalClaims["user_ver"]; ok {
			switch t := v.(type) {
			case int:
				claims.UserVer = t
			case int32:
				claims.UserVer = int(t)
			case int64:
				claims.UserVer = int(t)
			}
		}
	}

	// Sign the token
	token := jwt.NewWithClaims(signer.SigningMethod(), claims)
	token.Header["typ"] = TokenTypeAuth

	return signer.SignToken(claims)
}

// VerifyAuthToken validates an authentication token and returns the claims
func VerifyAuthToken(
	verifier foundryJWT.JWTVerifier,
	tokenString string,
) (*AuthClaims, error) {
	if verifier == nil {
		return nil, fmt.Errorf("verifier cannot be nil")
	}

	if tokenString == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	claims := &AuthClaims{}
	if err := verifier.VerifyToken(tokenString, claims); err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	// Additional validation
	if claims.Subject == "" {
		return nil, fmt.Errorf("token missing sub claim")
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

// HasPermission checks if the auth claims contain a specific permission
func HasPermission(claims *AuthClaims, permission auth.Permission) bool {
	if claims == nil {
		return false
	}

	for _, p := range claims.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasAnyPermission checks if the auth claims contain any of the specified permissions
func HasAnyPermission(claims *AuthClaims, permissions ...auth.Permission) bool {
	if claims == nil || len(permissions) == 0 {
		return false
	}

	for _, required := range permissions {
		if HasPermission(claims, required) {
			return true
		}
	}
	return false
}

// HasAllPermissions checks if the auth claims contain all of the specified permissions
func HasAllPermissions(claims *AuthClaims, permissions ...auth.Permission) bool {
	if claims == nil || len(permissions) == 0 {
		return false
	}

	for _, required := range permissions {
		if !HasPermission(claims, required) {
			return false
		}
	}
	return true
}

// HasAnyCertificateSignPermission checks if the user has any certificate signing permissions
func HasAnyCertificateSignPermission(claims *AuthClaims) bool {
	if claims == nil {
		return false
	}

	for _, perm := range claims.Permissions {
		if auth.IsCertificateSignPermission(perm) {
			return true
		}
	}
	return false
}

// GetCertificateSignPermissions returns all certificate signing permissions from the claims
func GetCertificateSignPermissions(claims *AuthClaims) []auth.Permission {
	if claims == nil {
		return nil
	}

	var certPerms []auth.Permission
	for _, perm := range claims.Permissions {
		if auth.IsCertificateSignPermission(perm) {
			certPerms = append(certPerms, perm)
		}
	}
	return certPerms
}

// Helper functions
func getOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

func getOrDefaultSlice(value, defaultValue []string) []string {
	if len(value) > 0 {
		return value
	}
	return defaultValue
}
