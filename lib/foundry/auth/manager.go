package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/redis/go-redis/v9"
)

// AuthManager exposes registration and authentication helpers.
type AuthManager struct {
	fs  fs.Filesystem
	rdb *redis.Client
}

// AuthManagerOption is a function that can be used to configure the AuthManager.
type AuthManagerOption func(*AuthManager)

// WithFilesystem sets the filesystem to use for the AuthManager.
func WithFilesystem(fs fs.Filesystem) AuthManagerOption {
	return func(am *AuthManager) {
		am.fs = fs
	}
}

// WithRedis sets the Redis client to use for the AuthManager.
func WithRedis(rdb *redis.Client) AuthManagerOption {
	return func(am *AuthManager) {
		am.rdb = rdb
	}
}

// NewAuthManager returns a new AuthManager.
func NewAuthManager(opts ...AuthManagerOption) *AuthManager {
	am := &AuthManager{
		fs: billy.NewBaseOsFS(),
	}

	for _, opt := range opts {
		opt(am)
	}

	return am
}

// GenerateKey creates a new Ed25519 key pair.
func (a *AuthManager) GenerateKeypair() (*KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &KeyPair{
		fs:         a.fs,
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// LoadKeyPair loads a KeyPair from the given path.
func (a *AuthManager) LoadKeyPair(path string) (*KeyPair, error) {
	publicKeyPath := filepath.Join(path, "public.pem")
	privateKeyPath := filepath.Join(path, "private.pem")

	publicKeyBytes, err := a.fs.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	privateKeyBytes, err := a.fs.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	publicKeyBlock, _ := pem.Decode(publicKeyBytes)
	if publicKeyBlock == nil {
		return nil, fmt.Errorf("failed to decode public key PEM")
	}

	publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	ed25519PublicKey, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an Ed25519 key")
	}

	privateKeyBlock, _ := pem.Decode(privateKeyBytes)
	if privateKeyBlock == nil {
		return nil, fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ed25519PrivateKey, ok := privateKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an Ed25519 key")
	}

	return &KeyPair{
		fs:         a.fs,
		PublicKey:  ed25519PublicKey,
		PrivateKey: ed25519PrivateKey,
	}, nil
}
