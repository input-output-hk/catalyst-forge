package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// ES256KeyPair represents a pair of ES256 keys with their raw contents
type ES256KeyPair struct {
	PrivateKeyPEM []byte
	PublicKeyPEM  []byte
}

// GenerateES256Keys generates a pair of ES256 keys and returns them
func GenerateES256Keys() (*ES256KeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	privateKeyPEMBytes := pem.EncodeToMemory(privateKeyPEM)

	publicKey := &privateKey.PublicKey

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicKeyPEMBytes := pem.EncodeToMemory(publicKeyPEM)

	return &ES256KeyPair{
		PrivateKeyPEM: privateKeyPEMBytes,
		PublicKeyPEM:  publicKeyPEMBytes,
	}, nil
}
