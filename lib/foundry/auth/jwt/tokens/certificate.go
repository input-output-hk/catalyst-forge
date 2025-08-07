package tokens

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	foundryJWT "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
)

// CertificateSigningOptions contains optional parameters for certificate signing tokens
type CertificateSigningOptions struct {
	// CommonName overrides the subject from the token subject claim
	CommonName string
	// Email to include in the certificate
	Email string
	// TTL overrides the default token lifetime (default: 5 minutes)
	TTL time.Duration
}

// GenerateCertificateSigningToken creates a JWT token for certificate signing with a CA like step-ca
// The token has a configurable TTL (default 5 minutes) for security
func GenerateCertificateSigningToken(
	signer foundryJWT.JWTSigner,
	subject string,
	audience string,
	sans []string,
	csrPEM []byte,
	opts ...CertificateSigningOption,
) (string, error) {
	if signer == nil {
		return "", fmt.Errorf("signer cannot be nil")
	}

	if subject == "" {
		return "", fmt.Errorf("subject cannot be empty")
	}

	if audience == "" {
		return "", fmt.Errorf("audience cannot be empty")
	}

	// Apply certificate options with defaults
	options := &CertificateSigningOptions{
		TTL: 5 * time.Minute, // Default TTL
	}
	for _, opt := range opts {
		opt(options)
	}

	// Validate TTL against signer's max
	if options.TTL <= 0 {
		return "", fmt.Errorf("TTL must be positive")
	}

	maxTTL := signer.MaxCertificateTTL()
	if maxTTL > 0 && options.TTL > maxTTL {
		return "", fmt.Errorf("TTL (%v) exceeds maximum allowed TTL (%v)", options.TTL, maxTTL)
	}

	// Calculate CSR fingerprint if provided
	var csrSHA string
	if len(csrPEM) > 0 {
		hash := sha256.Sum256(csrPEM)
		csrSHA = base64.RawURLEncoding.EncodeToString(hash[:])
	}

	// Build claims with configured TTL
	now := time.Now()
	claims := &CertificateClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    signer.Issuer(),
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(options.TTL)),
			NotBefore: jwt.NewNumericDate(now),
		},
		SANs:       sans,
		SHA:        csrSHA,
		Email:      options.Email,
		CommonName: getOrDefault(options.CommonName, subject),
	}

	// Sign the token
	return signer.SignToken(claims)
}

// GenerateCertificateSigningTokenWithTTL creates a certificate signing token with custom TTL
// This is a convenience function that sets the TTL option and calls GenerateCertificateSigningToken
func GenerateCertificateSigningTokenWithTTL(
	signer foundryJWT.JWTSigner,
	subject string,
	audience string,
	sans []string,
	csrPEM []byte,
	ttl time.Duration,
	opts ...CertificateSigningOption,
) (string, error) {
	// Prepend the TTL option to the other options
	allOpts := append([]CertificateSigningOption{WithTTL(ttl)}, opts...)

	// Use the main function with the TTL option
	return GenerateCertificateSigningToken(signer, subject, audience, sans, csrPEM, allOpts...)
}

// VerifyCertificateSigningToken validates a certificate signing token
func VerifyCertificateSigningToken(
	verifier foundryJWT.JWTVerifier,
	tokenString string,
) (*CertificateClaims, error) {
	if verifier == nil {
		return nil, fmt.Errorf("verifier cannot be nil")
	}

	if tokenString == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}

	claims := &CertificateClaims{}
	if err := verifier.VerifyToken(tokenString, claims); err != nil {
		return nil, fmt.Errorf("failed to verify certificate token: %w", err)
	}

	// Validate required fields
	if claims.Subject == "" {
		return nil, fmt.Errorf("certificate token missing subject")
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("certificate token has expired")
	}

	// Verify audience contains step-ca or similar
	hasValidAudience := false
	for _, aud := range claims.Audience {
		if aud == "step-ca" || aud == "ca" {
			hasValidAudience = true
			break
		}
	}
	if !hasValidAudience {
		return nil, fmt.Errorf("certificate token has invalid audience")
	}

	return claims, nil
}

// ValidateCSRFingerprint checks if the provided CSR matches the token's fingerprint
func ValidateCSRFingerprint(claims *CertificateClaims, csrPEM []byte) error {
	if claims == nil {
		return fmt.Errorf("claims cannot be nil")
	}

	// If no fingerprint in token, skip validation
	if claims.SHA == "" {
		return nil
	}

	if len(csrPEM) == 0 {
		return fmt.Errorf("CSR required when token contains fingerprint")
	}

	// Calculate CSR fingerprint
	hash := sha256.Sum256(csrPEM)
	csrSHA := base64.RawURLEncoding.EncodeToString(hash[:])

	if csrSHA != claims.SHA {
		return fmt.Errorf("CSR fingerprint mismatch")
	}

	return nil
}

// CertificateSigningOption allows customization of certificate signing tokens
type CertificateSigningOption func(*CertificateSigningOptions)

// WithCommonName sets a custom common name for the certificate
func WithCommonName(cn string) CertificateSigningOption {
	return func(opts *CertificateSigningOptions) {
		opts.CommonName = cn
	}
}

// WithEmail sets the email for the certificate
func WithEmail(email string) CertificateSigningOption {
	return func(opts *CertificateSigningOptions) {
		opts.Email = email
	}
}

// WithTTL sets the TTL for the certificate token
// The TTL will be validated against the signer's MaxCertificateTTL
func WithTTL(ttl time.Duration) CertificateSigningOption {
	return func(opts *CertificateSigningOptions) {
		opts.TTL = ttl
	}
}
