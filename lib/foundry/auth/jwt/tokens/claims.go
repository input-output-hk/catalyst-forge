package tokens

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

// AuthClaims represents the JWT claims structure for authentication tokens
type AuthClaims struct {
	Permissions []auth.Permission `json:"perms"`
	AKID        string            `json:"akid,omitempty"`
	UserVer     int               `json:"user_ver,omitempty"`
	jwt.RegisteredClaims
}

// CertificateClaims represents the JWT claims for certificate signing tokens
// These tokens are used with step-ca or other certificate authorities
type CertificateClaims struct {
	jwt.RegisteredClaims
	// SANs are the Subject Alternative Names for the certificate
	SANs []string `json:"sans,omitempty"`
	// SHA is the SHA256 fingerprint of the CSR to bind the token to a specific request
	SHA string `json:"sha,omitempty"`
	// Email can be included as additional identity information
	Email string `json:"email,omitempty"`
	// CommonName override for the certificate subject
	CommonName string `json:"cn,omitempty"`
}

// ChallengeClaims represents the JWT claims structure for challenge tokens
// These are used in challenge-response authentication flows
type ChallengeClaims struct {
	Email string `json:"email"`
	Kid   string `json:"kid"`
	Nonce string `json:"nonce"`
	jwt.RegisteredClaims
}

// TokenType constants for different token types
const (
	TokenTypeAuth        = "auth+jwt"
	TokenTypeChallenge   = "challenge+jwt"
	TokenTypeCertificate = "certificate+jwt"
)
