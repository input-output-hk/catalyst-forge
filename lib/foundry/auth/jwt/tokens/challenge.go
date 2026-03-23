package tokens

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	foundryJWT "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
)

// GenerateChallengeJWT generates a challenge JWT token for authentication flows
// Returns the token and the nonce that must be signed by the user
func GenerateChallengeJWT(
	signer foundryJWT.JWTSigner,
	email string,
	kid string,
	ttl time.Duration,
	opts ...foundryJWT.TokenOption,
) (string, string, error) {
	if signer == nil {
		return "", "", fmt.Errorf("signer cannot be nil")
	}

	if email == "" {
		return "", "", fmt.Errorf("email cannot be empty")
	}

	if kid == "" {
		return "", "", fmt.Errorf("kid cannot be empty")
	}

	if ttl <= 0 {
		return "", "", fmt.Errorf("ttl must be positive")
	}

	// Generate a cryptographically secure nonce
	nonceBytes := make([]byte, 16) // 128-bit nonce
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

	// Apply token options
	options := &foundryJWT.TokenOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Build claims
	now := time.Now()
	claims := &ChallengeClaims{
		Email: email,
		Kid:   kid,
		Nonce: nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   email, // Use email as subject
			Issuer:    getOrDefault(options.Issuer, signer.Issuer()),
			Audience:  getOrDefaultSlice(options.Audiences, signer.DefaultAudiences()),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        nonce, // Use nonce as JWT ID for single-use validation
		},
	}

	// Sign the token using the signer interface
	// Note: We can't set custom headers with the current interface
	// The SignToken method handles the actual signing
	tokenString, err := signer.SignToken(claims)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign challenge token: %w", err)
	}

	return tokenString, nonce, nil
}

// VerifyChallengeJWT validates a challenge JWT token
func VerifyChallengeJWT(
	verifier foundryJWT.JWTVerifier,
	tokenString string,
) (*ChallengeClaims, error) {
	if verifier == nil {
		return nil, fmt.Errorf("verifier cannot be nil")
	}

	if tokenString == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	claims := &ChallengeClaims{}
	if err := verifier.VerifyToken(tokenString, claims); err != nil {
		return nil, fmt.Errorf("failed to verify challenge token: %w", err)
	}

	// Additional validation
	if claims.Email == "" {
		return nil, fmt.Errorf("challenge token missing email")
	}

	if claims.Kid == "" {
		return nil, fmt.Errorf("challenge token missing kid")
	}

	if claims.Nonce == "" {
		return nil, fmt.Errorf("challenge token missing nonce")
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("challenge token has expired")
	}

	return claims, nil
}
