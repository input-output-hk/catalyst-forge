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
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

// KeyPair holds an Ed25519 key pair.
type KeyPair struct {
	fs         fs.Filesystem
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// KeyPairChallenge represents a challenge for a key pair.
type KeyPairChallenge struct {
	Challenge string    `json:"challenge"`
	Email     string    `json:"email"`
	KeyID     string    `json:"kid"`
	Expires   time.Time `json:"expires"`
}

// ID returns the ID of the challenge.
func (k *KeyPairChallenge) ID() string {
	hash := sha256.Sum256([]byte(k.Challenge))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// KeyPairChallengeResponse represents the response to a challenge.
type KeyPairChallengeResponse struct {
	Challenge string    `json:"challenge"`
	Email     string    `json:"email"`
	KeyID     string    `json:"kid"`
	Expires   time.Time `json:"expires"`
	Signature string    `json:"signature"`
}

// ID returns the ID of the challenge response.
func (k *KeyPairChallengeResponse) ID() string {
	hash := sha256.Sum256([]byte(k.Challenge))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// EncodePrivateKey returns the base64 encoded private key
func (k *KeyPair) EncodePrivateKey() string {
	return base64.StdEncoding.EncodeToString(k.PrivateKey)
}

// EncodePublicKey returns the base64 encoded public key
func (k *KeyPair) EncodePublicKey() string {
	return base64.StdEncoding.EncodeToString(k.PublicKey)
}

// GenerateChallenge generates a random challenge for a key pair.
func (k *KeyPair) GenerateChallenge(email string, duration time.Duration) (*KeyPairChallenge, error) {
	challenge, err := k.generateRandomString()
	if err != nil {
		return nil, err
	}

	return &KeyPairChallenge{
		Challenge: challenge,
		Email:     email,
		KeyID:     k.Kid(),
		Expires:   time.Now().Add(duration),
	}, nil
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
func (k *KeyPair) SignChallenge(challenge *KeyPairChallenge) (*KeyPairChallengeResponse, error) {
	signature, err := k.PrivateKey.Sign(nil, []byte(challenge.Challenge), crypto.Hash(0))
	if err != nil {
		return nil, err
	}

	return &KeyPairChallengeResponse{
		Challenge: challenge.Challenge,
		Email:     challenge.Email,
		KeyID:     challenge.KeyID,
		Expires:   challenge.Expires,
		Signature: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// VerifyChallenge verifies a challenge response with the key pair.
func (k *KeyPair) VerifyChallenge(challenge *KeyPairChallengeResponse) error {
	signature, err := base64.StdEncoding.DecodeString(challenge.Signature)
	if err != nil {
		return err
	}

	verified := ed25519.Verify(k.PublicKey, []byte(challenge.Challenge), signature)
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
