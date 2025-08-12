package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
)

// secureRand implements the Rand interface using crypto/rand.
type secureRand struct{}

// NewSecureRand creates a new secure random generator.
func NewSecureRand() Rand {
	return &secureRand{}
}

// Bytes generates n random bytes.
func (r *secureRand) Bytes(n int) ([]byte, error) {
	if n <= 0 {
		return nil, errors.New("invalid byte count")
	}

	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}

	return b, nil
}

// String generates a base64url-encoded random string from n bytes.
func (r *secureRand) String(nBytes int) (string, error) {
	b, err := r.Bytes(nBytes)
	if err != nil {
		return "", err
	}

	// Use RawURLEncoding for URL-safe base64 without padding
	return base64.RawURLEncoding.EncodeToString(b), nil
}