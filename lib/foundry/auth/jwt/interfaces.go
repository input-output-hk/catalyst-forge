package jwt

import (
	"crypto"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTSigner handles JWT signing operations
type JWTSigner interface {
	// SignToken signs the provided claims and returns a JWT string
	SignToken(claims jwt.Claims) (string, error)

	// SigningMethod returns the JWT signing method used by this signer
	SigningMethod() jwt.SigningMethod

	// Issuer returns the issuer identifier for tokens
	Issuer() string

	// DefaultAudiences returns the default audiences for tokens
	DefaultAudiences() []string

	// MaxAuthTokenTTL returns the maximum allowed TTL for authentication tokens
	// This helps enforce security policies at the signer level
	MaxAuthTokenTTL() time.Duration

	// MaxCertificateTTL returns the maximum allowed TTL for certificate signing tokens
	// This helps enforce security policies at the signer level
	MaxCertificateTTL() time.Duration
}

// JWTVerifier handles JWT verification operations
type JWTVerifier interface {
	// VerifyToken verifies the token string and populates the provided claims
	// Returns an error if the token is invalid or verification fails
	VerifyToken(tokenString string, claims jwt.Claims) error

	// PublicKey returns the public key used for verification
	// This can be used for JWKS endpoints or external verification
	PublicKey() crypto.PublicKey
}

// JWTManager combines signing and verification capabilities
type JWTManager interface {
	JWTSigner
	JWTVerifier
}

// TokenOption allows customization of token generation
type TokenOption func(*TokenOptions)

// TokenOptions contains optional parameters for token generation
type TokenOptions struct {
	// Audiences overrides default audiences
	Audiences []string
	// Issuer overrides default issuer
	Issuer string
	// ID sets a unique token ID (jti claim)
	ID string
	// AdditionalClaims for custom claims not in standard structure
	AdditionalClaims map[string]interface{}
}

// WithAudiences sets custom audiences for the token
func WithAudiences(audiences ...string) TokenOption {
	return func(opts *TokenOptions) {
		opts.Audiences = audiences
	}
}

// WithIssuer sets a custom issuer for the token
func WithIssuer(issuer string) TokenOption {
	return func(opts *TokenOptions) {
		opts.Issuer = issuer
	}
}

// WithTokenID sets a unique ID for the token
func WithTokenID(id string) TokenOption {
	return func(opts *TokenOptions) {
		opts.ID = id
	}
}

// WithAdditionalClaims adds custom claims to the token
func WithAdditionalClaims(claims map[string]interface{}) TokenOption {
	return func(opts *TokenOptions) {
		opts.AdditionalClaims = claims
	}
}
