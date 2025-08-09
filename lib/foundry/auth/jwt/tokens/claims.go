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
	TokenTypeAuth      = "auth+jwt"
	TokenTypeChallenge = "challenge+jwt"
	// TokenTypeCertificate removed with token-bound CSR feature
)
