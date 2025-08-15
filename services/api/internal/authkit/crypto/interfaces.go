package crypto

import (
	"context"
)

// KeyManager handles JWT signing and verification.
type KeyManager interface {
	// SignJWT signs a JWT payload with the specified key ID.
	SignJWT(ctx context.Context, payload []byte, kid string) (token string, err error)
	
	// VerifyJWT verifies and parses a JWT token.
	VerifyJWT(ctx context.Context, token string) (claims map[string]interface{}, err error)
	
	// JWKS returns the JSON Web Key Set for public key verification.
	JWKS() interface{}
	
	// CurrentKID returns the current key ID for signing new tokens.
	CurrentKID() string
}

// Rand provides secure random generation.
type Rand interface {
	// Bytes generates n random bytes.
	Bytes(n int) ([]byte, error)
	
	// String generates a base64url-encoded random string from n bytes.
	String(nBytes int) (string, error)
}