package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"gopkg.in/square/go-jose.v2"
)

const (
	ISSUER   = "foundry.projectcatalyst.io"
	AUDIENCE = "catalyst-forge"
)

// Ensure ES256Manager implements JWTManager interface
var _ JWTManager = (*ES256Manager)(nil)

// ES256Manager handles ES256 (ECDSA P-256) JWT operations
type ES256Manager struct {
	audiences         []string
	fs                fs.Filesystem
	issuer            string
	logger            *slog.Logger
	maxAuthTokenTTL   time.Duration
	maxCertificateTTL time.Duration
	privateKey        *ecdsa.PrivateKey
	publicKey         *ecdsa.PublicKey
}

// DefaultAudiences implements JWTSigner interface
func (m *ES256Manager) DefaultAudiences() []string {
	return m.audiences
}

// Issuer implements JWTSigner interface
func (m *ES256Manager) Issuer() string {
	return m.issuer
}

// MaxAuthTokenTTL implements JWTSigner interface
func (m *ES256Manager) MaxAuthTokenTTL() time.Duration {
	return m.maxAuthTokenTTL
}

// MaxCertificateTTL implements JWTSigner interface
func (m *ES256Manager) MaxCertificateTTL() time.Duration {
	return m.maxCertificateTTL
}

// NewES256Manager creates a new ES256 JWT manager with the provided keys
// At least one key (private or public) must be provided.
// If only private key is provided, public key will be derived from it.
// If only public key is provided, only verification operations are supported.
// If both are provided, they must form a valid key pair.
func NewES256Manager(privateKeyPath, publicKeyPath string, opts ...ManagerOption) (*ES256Manager, error) {
	m := &ES256Manager{
		audiences:         []string{AUDIENCE},
		fs:                billy.NewBaseOsFS(),
		issuer:            ISSUER,
		logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		maxAuthTokenTTL:   24 * time.Hour,   // Default max TTL for auth tokens
		maxCertificateTTL: 10 * time.Minute, // Default max TTL for certificate tokens
	}

	// Apply configuration options
	for _, opt := range opts {
		opt(m)
	}

	// Load keys
	if err := m.loadKeys(privateKeyPath, publicKeyPath); err != nil {
		return nil, err
	}

	// Validate we have at least one key
	if m.privateKey == nil && m.publicKey == nil {
		return nil, fmt.Errorf("at least one key (private or public) must be provided")
	}

	// Log capabilities
	m.logCapabilities()

	return m, nil
}

// PublicKey implements JWTVerifier interface
func (m *ES256Manager) PublicKey() crypto.PublicKey {
	return m.publicKey
}

// SignToken implements JWTSigner interface
func (m *ES256Manager) SignToken(claims jwt.Claims) (string, error) {
	if m.privateKey == nil {
		return "", fmt.Errorf("no private key available for signing")
	}

	jwkKey := jose.JSONWebKey{Key: &m.privateKey.PublicKey, Algorithm: "ES256"}
	thumb, err := jwkKey.Thumbprint(crypto.SHA256) // RFC-7638 thumbprint
	if err != nil {
		return "", fmt.Errorf("failed to compute JWK thumbprint: %w", err)
	}
	kid := base64.RawURLEncoding.EncodeToString(thumb)

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = kid
	return token.SignedString(m.privateKey)
}

// SigningMethod implements JWTSigner interface
func (m *ES256Manager) SigningMethod() jwt.SigningMethod {
	return jwt.SigningMethodES256
}

// VerifyToken implements JWTVerifier interface
func (m *ES256Manager) VerifyToken(tokenString string, claims jwt.Claims) error {
	if m.publicKey == nil {
		return fmt.Errorf("no public key available for verification")
	}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}

// isValidKeyPair checks if the loaded public key matches the private key
func (m *ES256Manager) isValidKeyPair() bool {
	if m.privateKey == nil || m.publicKey == nil {
		return false
	}

	// Compare the public key from private key with the loaded public key
	return m.privateKey.PublicKey.Equal(m.publicKey)
}

// loadAndSetPrivateKey loads and sets the private key from file
func (m *ES256Manager) loadAndSetPrivateKey(path string) error {
	privateKeyBytes, err := m.loadPrivateKey(path)
	if err != nil {
		return err
	}

	privateKey, err := x509.ParseECPrivateKey(privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	m.privateKey = privateKey
	return nil
}

// loadAndSetPublicKey loads and sets the public key from file
func (m *ES256Manager) loadAndSetPublicKey(path string) error {
	publicKeyBytes, err := m.loadPublicKey(path)
	if err != nil {
		return err
	}

	publicKey, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	ecdsaPublicKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not an ECDSA key")
	}

	m.publicKey = ecdsaPublicKey
	return nil
}

// loadCapabilities logs what operations the manager supports based on loaded keys
func (m *ES256Manager) logCapabilities() {
	switch {
	case m.privateKey != nil && m.publicKey != nil:
		m.logger.Info("ES256Manager initialized with full capabilities (signing and verification)")
	case m.privateKey != nil:
		m.logger.Info("ES256Manager initialized with signing capability only")
	case m.publicKey != nil:
		m.logger.Info("ES256Manager initialized with verification capability only")
	}
}

// loadKeys loads and configures the private and/or public keys
func (m *ES256Manager) loadKeys(privateKeyPath, publicKeyPath string) error {
	// Load private key if provided
	if privateKeyPath != "" {
		if err := m.loadAndSetPrivateKey(privateKeyPath); err != nil {
			return fmt.Errorf("failed to load private key: %w", err)
		}
	}

	// Load or derive public key
	if publicKeyPath != "" {
		if err := m.loadAndSetPublicKey(publicKeyPath); err != nil {
			return fmt.Errorf("failed to load public key: %w", err)
		}

		// Validate key pair if both keys are loaded
		if m.privateKey != nil && !m.isValidKeyPair() {
			return fmt.Errorf("provided public key does not match private key")
		}
	} else if m.privateKey != nil {
		// Derive public key from private key
		m.publicKey = &m.privateKey.PublicKey
		m.logger.Debug("derived public key from private key")
	}

	return nil
}

// loadPEMFile loads a PEM file from disk
func (m *ES256Manager) loadPEMFile(path string) (*pem.Block, error) {
	data, err := m.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	return block, nil
}

// loadPrivateKey loads an ECDSA private key from a PEM file
func (m *ES256Manager) loadPrivateKey(path string) ([]byte, error) {
	block, err := m.loadPEMFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load PEM file: %w", err)
	}

	if block.Type != "EC PRIVATE KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", block.Type)
	}

	return block.Bytes, nil
}

// loadPublicKey loads a public key from a PEM file
func (m *ES256Manager) loadPublicKey(path string) ([]byte, error) {
	block, err := m.loadPEMFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load PEM file: %w", err)
	}

	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("unexpected PEM block type: %s", block.Type)
	}

	return block.Bytes, nil
}
