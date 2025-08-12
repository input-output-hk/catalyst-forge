package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"
)

// GenerateES256KeyPair generates a new ES256 (P-256) key pair.
func GenerateES256KeyPair() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// EncodePrivateKeyPEM encodes an ECDSA private key to PEM format.
func EncodePrivateKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
	x509Encoded, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	
	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: x509Encoded,
	}
	
	return pem.EncodeToMemory(pemBlock), nil
}

// EncodePublicKeyPEM encodes an ECDSA public key to PEM format.
func EncodePublicKeyPEM(key *ecdsa.PublicKey) ([]byte, error) {
	x509Encoded, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}
	
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: x509Encoded,
	}
	
	return pem.EncodeToMemory(pemBlock), nil
}

// DecodePrivateKeyPEM decodes a PEM-encoded ECDSA private key.
func DecodePrivateKeyPEM(pemData []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	
	switch block.Type {
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an ECDSA private key")
		}
		return ecKey, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}
}

// GenerateKeyID generates a key ID based on the current time.
//
// Format: YYYY-QN where N is the quarter number.
func GenerateKeyID() string {
	now := time.Now()
	quarter := (now.Month()-1)/3 + 1
	return fmt.Sprintf("%d-q%d", now.Year(), quarter)
}

// CreateTestKeyManager creates a key manager with a test key for development.
func CreateTestKeyManager() (KeyManager, error) {
	manager := NewES256KeyManager()
	
	// Generate a test key
	key, err := GenerateES256KeyPair()
	if err != nil {
		return nil, err
	}
	
	// Encode to PEM
	pemKey, err := EncodePrivateKeyPEM(key)
	if err != nil {
		return nil, err
	}
	
	// Add to manager with a test KID
	kid := GenerateKeyID()
	if km, ok := manager.(*es256KeyManager); ok {
		if err := km.AddKey(kid, pemKey); err != nil {
			return nil, err
		}
	}
	
	return manager, nil
}