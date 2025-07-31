package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

const (
	ISSUER   = "foundry.projectcatalyst.io"
	AUDIENCE = "catalyst-forge"
)

// AuthManager handles authentication operations including JWT token management and key generation
type AuthManager struct {
	audiences  []string
	fs         fs.Filesystem
	issuer     string
	logger     *slog.Logger
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

// Claims represents the JWT claims structure
type Claims struct {
	UserID      string       `json:"user_id"`
	Permissions []Permission `json:"permissions"`
	jwt.RegisteredClaims
}

// ES256KeyPair represents a pair of ES256 keys with their raw contents
type ES256KeyPair struct {
	PrivateKeyPEM []byte
	PublicKeyPEM  []byte
}

// AuthManagerOption is a function that configures an AuthManager
type AuthManagerOption func(*AuthManager)

// WithAudiences sets the audiences for the AuthManager
func WithAudiences(audiences []string) AuthManagerOption {
	return func(am *AuthManager) {
		am.audiences = audiences
	}
}

// WithFilesystem sets the filesystem for the AuthManager
func WithFilesystem(fs fs.Filesystem) AuthManagerOption {
	return func(am *AuthManager) {
		am.fs = fs
	}
}

// WithIssuer sets the issuer for the AuthManager
func WithIssuer(issuer string) AuthManagerOption {
	return func(am *AuthManager) {
		am.issuer = issuer
	}
}

// WithLogger sets the logger for the AuthManager
func WithLogger(logger *slog.Logger) AuthManagerOption {
	return func(am *AuthManager) {
		am.logger = logger
	}
}

// NewAuthManager creates a new auth manager with the provided keys
// The public key is optional, if not provided, only validation is supported
// The private key is also optional, if not provided, only signing is supported
func NewAuthManager(privateKeyPath, publicKeyPath string, opts ...AuthManagerOption) (*AuthManager, error) {
	am := &AuthManager{
		audiences: []string{AUDIENCE},
		fs:        billy.NewBaseOsFS(),
		issuer:    ISSUER,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	for _, opt := range opts {
		opt(am)
	}

	var privateKey *ecdsa.PrivateKey
	if privateKeyPath != "" {
		privateKeyBytes, err := am.loadPrivateKey(privateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load private key: %w", err)
		}

		privateKey, err = x509.ParseECPrivateKey(privateKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	} else {
		am.logger.Warn("no private key provided, only validation is supported")
	}

	var ecdsaPublicKey *ecdsa.PublicKey
	var ok bool
	if publicKeyPath != "" {
		publicKeyBytes, err := am.loadPublicKey(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load public key: %w", err)
		}

		publicKey, err := x509.ParsePKIXPublicKey(publicKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}

		ecdsaPublicKey, ok = publicKey.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not an ECDSA key")
		}
	} else {
		am.logger.Warn("no public key provided, only signing is supported")
	}

	return &AuthManager{
		privateKey: privateKey,
		publicKey:  ecdsaPublicKey,
		issuer:     am.issuer,
		audiences:  am.audiences,
	}, nil
}

// GenerateToken creates a new JWT token for the given user ID
func (am *AuthManager) GenerateToken(
	userID string,
	permissions []Permission,
	expiration time.Duration) (string, error) {
	if am.privateKey == nil {
		return "", fmt.Errorf("no private key provided, only validation is supported")
	}

	now := time.Now()
	claims := &Claims{
		UserID:      userID,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    am.issuer,
			Audience:  am.audiences,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	return token.SignedString(am.privateKey)
}

// HasPermission checks if the token has the given permission
func (am *AuthManager) HasPermission(tokenString string, permission Permission) (bool, error) {
	claims, err := am.ValidateToken(tokenString)
	if err != nil {
		return false, fmt.Errorf("failed to validate token: %w", err)
	}

	for _, perm := range claims.Permissions {
		if perm == permission {
			return true, nil
		}
	}

	return false, nil
}

// ValidateToken validates and parses a JWT token
func (am *AuthManager) ValidateToken(tokenString string) (*Claims, error) {
	if am.publicKey == nil {
		return nil, fmt.Errorf("no public key provided, only signing is supported")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// loadPrivateKey loads a ECDSA private key from a PEM file
func (am *AuthManager) loadPrivateKey(path string) ([]byte, error) {
	block, err := am.loadPEMFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load PEM file: %w", err)
	}

	if block.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", block.Type)
	}

	return block.Bytes, nil
}

// loadPublicKey loads a public key from a PEM file
func (am *AuthManager) loadPublicKey(path string) ([]byte, error) {
	block, err := am.loadPEMFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load PEM file: %w", err)
	}

	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", block.Type)
	}

	return block.Bytes, nil
}

// loadPEMFile loads a PEM file from disk
func (am *AuthManager) loadPEMFile(path string) (*pem.Block, error) {
	data, err := am.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return block, nil
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
