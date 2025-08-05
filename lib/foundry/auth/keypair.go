package auth

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"path/filepath"

	"github.com/golang-jwt/jwt/v5"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

// KeyPair holds an Ed25519 key pair.
type KeyPair struct {
	fs         fs.Filesystem
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// LoginRequest represents the request body for authentication
type LoginRequest struct {
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

// EncodePrivateKey returns the base64 encoded private key
func (k *KeyPair) EncodePrivateKey() string {
	return base64.StdEncoding.EncodeToString(k.PrivateKey)
}

// EncodePublicKey returns the base64 encoded public key
func (k *KeyPair) EncodePublicKey() string {
	return base64.StdEncoding.EncodeToString(k.PublicKey)
}

// Kid returns the Key ID of the key pair.
func (k *KeyPair) Kid() string {
	sum := sha256.Sum256(k.PublicKey)
	return "sha256:" + base64.RawURLEncoding.EncodeToString(sum[:])
}

// Save saves the key pair to the filesystem.
func (k *KeyPair) Save(dir string) error {
	pubPEM, err := k.encodePublicPEM()
	if err != nil {
		return err
	}
	privPEM, err := k.encodePrivatePEM()
	if err != nil {
		return err
	}

	if err := k.fs.WriteFile(filepath.Join(dir, "public.pem"), pubPEM, 0600); err != nil {
		return err
	}
	if err := k.fs.WriteFile(filepath.Join(dir, "private.pem"), privPEM, 0600); err != nil {
		return err
	}
	return nil
}

// SignChallenge signs a challenge with the key pair.
func (k *KeyPair) SignChallenge(tokenString string) (*LoginRequest, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse challenge token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to get claims from challenge token")
	}

	kid, ok := claims["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("kid not found in token claims")
	}
	nonce, ok := claims["nonce"].(string)
	if !ok {
		return nil, fmt.Errorf("nonce not found in token claims")
	}

	// 1. Validate that the kid matches the key being used to sign
	if kid != k.Kid() {
		return nil, fmt.Errorf("kid mismatch: token kid is %s, keypair kid is %s", kid, k.Kid())
	}

	// 2. Sign the Nonce from the JWT claims
	signature, err := k.PrivateKey.Sign(nil, []byte(nonce), crypto.Hash(0))
	if err != nil {
		return nil, err
	}

	// Return a type matching the expected LoginRequest
	return &LoginRequest{
		Challenge: tokenString,
		Signature: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// VerifySignature verifies a signature with the key pair.
func (k *KeyPair) VerifySignature(message, signature string) error {
	sig, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return err
	}

	verified := ed25519.Verify(k.PublicKey, []byte(message), sig)
	if !verified {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// encodePublicPEM encodes the public key to PEM format.
func (k *KeyPair) encodePublicPEM() ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(k.PublicKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

// encodePrivatePEM encodes the private key to PEM format.
func (k *KeyPair) encodePrivatePEM() ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(k.PrivateKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

// generateRandomString generates a random string by generating 32 random bytes
// and base64 encoding them.
func (a *KeyPair) generateRandomString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}
