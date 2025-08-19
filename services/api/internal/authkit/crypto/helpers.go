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

// EncodePrivateKeyPEM encodes an ECDSA private key to SEC1 (EC PRIVATE KEY) PEM format.
// Use EncodePrivateKeyPEMPKCS8 to produce PKCS#8 if required by external tools.
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

// EncodePrivateKeyPEMPKCS8 encodes an ECDSA private key to PKCS#8 PEM format.
func EncodePrivateKeyPEMPKCS8(key *ecdsa.PrivateKey) ([]byte, error) {
	x509Encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
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
// Supports SEC1 (EC PRIVATE KEY) and PKCS#8 (PRIVATE KEY) formats.
// Enforces ES256 by rejecting keys not on the P-256 curve.
func DecodePrivateKeyPEM(pemData []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	var k *ecdsa.PrivateKey
	switch block.Type {
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		k = key
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		ecKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an ECDSA private key")
		}
		k = ecKey
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}
	if k.Curve != elliptic.P256() {
		return nil, errors.New("key must use P-256 curve for ES256")
	}
	return k, nil
}

// GenerateKeyID generates a key ID based on the current time.
// Format: YYYY-QN where N is the quarter number.
func GenerateKeyID() string {
	return GenerateKeyIDAt(time.Now())
}

// GenerateKeyIDAt is a testable variant that derives a key ID for the provided time.
func GenerateKeyIDAt(t time.Time) string {
	quarter := (t.Month()-1)/3 + 1
	return fmt.Sprintf("%d-q%d", t.Year(), quarter)
}
