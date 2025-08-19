package crypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"sync"

	"github.com/golang-jwt/jwt/v5"
)

// es256KeyManager implements KeyManager using ES256 (ECDSA with P-256).
type es256KeyManager struct {
	mu         sync.RWMutex
	keys       map[string]*ecdsa.PrivateKey
	currentKID string
	publicKeys map[string]*ecdsa.PublicKey
}

// NewES256KeyManager creates a new ES256 key manager without pre-generated keys.
//
// Thread-safety: the returned manager is safe for concurrent use. Call AddKey or
// GenerateKey and optionally SetCurrentKID before signing.
func NewES256KeyManager() KeyManager {
	km := &es256KeyManager{
		keys:       make(map[string]*ecdsa.PrivateKey),
		publicKeys: make(map[string]*ecdsa.PublicKey),
	}
	return km
}

// AddKey adds a key to the manager.
//
// The key should be PEM encoded in either SEC1 ("EC PRIVATE KEY") or PKCS#8
// ("PRIVATE KEY") format, and MUST use the P-256 curve (ES256).
func (m *es256KeyManager) AddKey(kid string, pemKey []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	block, _ := pem.Decode(pemKey)
	if block == nil {
		return errors.New("failed to parse PEM block")
	}

	var key *ecdsa.PrivateKey
	var err error

	switch block.Type {
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return err
		}
		var ok bool
		key, ok = parsedKey.(*ecdsa.PrivateKey)
		if !ok {
			return errors.New("not an ECDSA private key")
		}
	default:
		return fmt.Errorf("unsupported key type: %s", block.Type)
	}

	if err != nil {
		return err
	}

	// Verify it's P-256
	if key.Curve != elliptic.P256() {
		return errors.New("key must use P-256 curve for ES256")
	}

	m.keys[kid] = key
	m.publicKeys[kid] = &key.PublicKey

	// Set as current if it's the first key
	if m.currentKID == "" {
		m.currentKID = kid
	}

	return nil
}

// GenerateKey generates a new ES256 (P-256) key pair and makes it available for
// verification and (if it is the first key) signing.
func (m *es256KeyManager) GenerateKey(kid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	m.keys[kid] = key
	m.publicKeys[kid] = &key.PublicKey

	// Set as current if it's the first key
	if m.currentKID == "" {
		m.currentKID = kid
	}

	return nil
}

// SetCurrentKID sets the current key ID used for signing.
func (m *es256KeyManager) SetCurrentKID(kid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.keys[kid]; !exists {
		return errors.New("key not found")
	}

	m.currentKID = kid
	return nil
}

// SignJWT signs a JWT payload with the specified key ID (or the current key if
// kid is empty). The token is signed with ES256 and includes the "kid" header.
func (m *es256KeyManager) SignJWT(ctx context.Context, payload []byte, kid string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Use current key if kid is empty
	if kid == "" {
		kid = m.currentKID
	}

	key, exists := m.keys[kid]
	if !exists {
		return "", errors.New("signing key not found")
	}

	// Parse the payload JSON into claims
	var claims jwt.MapClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("invalid payload JSON: %w", err)
	}

	// Create token with ES256 signing method
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = kid

	// Sign the token
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// VerifyJWT verifies and parses a JWT token.
//
// Security: this rejects algorithm confusion by requiring ES256 explicitly and
// requires a present and known "kid" header.
func (m *es256KeyManager) VerifyJWT(ctx context.Context, tokenString string) (map[string]interface{}, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method is exactly ES256
		if token.Method != jwt.SigningMethodES256 {
			return nil, fmt.Errorf("unexpected alg: %v", token.Header["alg"])
		}

		// Get the key ID from the header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		m.mu.RLock()
		defer m.mu.RUnlock()

		// Get the public key
		pubKey, exists := m.publicKeys[kid]
		if !exists {
			return nil, errors.New("verification key not found")
		}

		return pubKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// Return claims as map
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		return claims, nil
	}

	return nil, errors.New("invalid claims format")
}

// JWKS returns the JSON Web Key Set for public key verification.
func (m *es256KeyManager) JWKS() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]map[string]interface{}, 0, len(m.publicKeys))
	for kid, pubKey := range m.publicKeys {
		x, y := pubKey.X.Bytes(), pubKey.Y.Bytes()
		keys = append(keys, map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"alg": "ES256",
			"use": "sig",
			"kid": kid,
			"x":   base64.RawURLEncoding.EncodeToString(pad32(x)),
			"y":   base64.RawURLEncoding.EncodeToString(pad32(y)),
		})
	}

	return map[string]interface{}{"keys": keys}
}

// pad32 left-pads a big-endian integer byte slice to 32 bytes.
func pad32(b []byte) []byte {
	if len(b) >= 32 {
		return b
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// CurrentKID returns the current key ID for signing new tokens.
func (m *es256KeyManager) CurrentKID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentKID
}
