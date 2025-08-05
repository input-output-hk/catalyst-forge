package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

const (
	ISSUER         = "foundry.projectcatalyst.io"
	AUDIENCE       = "catalyst-forge"
	CHALLENGE_TYPE = "challenge+jwt"
)

// JWTManager handles authentication operations including JWT token management and key generation
type JWTManager struct {
	audiences  []string
	fs         fs.Filesystem
	issuer     string
	logger     *slog.Logger
	privateKey *ecdsa.PrivateKey
	publicKey  *ecdsa.PublicKey
}

// Claims represents the JWT claims structure
type Claims struct {
	UserID      string            `json:"user_id"`
	Permissions []auth.Permission `json:"permissions"`
	jwt.RegisteredClaims
}

// ChallengeClaims represents the JWT claims structure for challenge tokens
type ChallengeClaims struct {
	Email string `json:"email"`
	Kid   string `json:"kid"`
	Nonce string `json:"nonce"`
	jwt.RegisteredClaims
}

// ES256KeyPair represents a pair of ES256 keys with their raw contents
type ES256KeyPair struct {
	PrivateKeyPEM []byte
	PublicKeyPEM  []byte
}

// JWTManagerOption is a function that configures a JWTManager
type JWTManagerOption func(*JWTManager)

// WithAudiences sets the audiences for the JWTManager
func WithAudiences(audiences []string) JWTManagerOption {
	return func(am *JWTManager) {
		am.audiences = audiences
	}
}

// WithFilesystem sets the filesystem for the JWTManager
func WithFilesystem(fs fs.Filesystem) JWTManagerOption {
	return func(am *JWTManager) {
		am.fs = fs
	}
}

// WithIssuer sets the issuer for the JWTManager
func WithIssuer(issuer string) JWTManagerOption {
	return func(am *JWTManager) {
		am.issuer = issuer
	}
}

// WithLogger sets the logger for the JWTManager
func WithLogger(logger *slog.Logger) JWTManagerOption {
	return func(am *JWTManager) {
		am.logger = logger
	}
}

// NewJWTManager creates a new JWT manager with the provided keys
// The public key is optional, if not provided, only validation is supported
// The private key is also optional, if not provided, only signing is supported
func NewJWTManager(privateKeyPath, publicKeyPath string, opts ...JWTManagerOption) (*JWTManager, error) {
	am := &JWTManager{
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

	return &JWTManager{
		privateKey: privateKey,
		publicKey:  ecdsaPublicKey,
		issuer:     am.issuer,
		audiences:  am.audiences,
	}, nil
}

// GenerateChallengeJWT generates a challenge JWT token for the given email and kid
// The token is signed with the private key and can be used to verify the user's identity
// The token is valid for the given TTL
// The token is a JWT with the following claims:
// - email: the email of the user trying to log in
// - kid: the kid of the user key that must sign the nonce
// - nonce: a 128-bit random nonce
// - exp: the expiration time of the token
func (am *JWTManager) GenerateChallengeJWT(
	email, kid string,
	ttl time.Duration,
) (string, string, error) {
	if am.privateKey == nil {
		return "", "", fmt.Errorf("signing key not loaded")
	}

	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)

	now := time.Now()
	claims := ChallengeClaims{
		Email: email,
		Kid:   kid,
		Nonce: nonce,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    am.issuer,
			Audience:  am.audiences,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        nonce, // jti = nonce (single-use)
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["typ"] = CHALLENGE_TYPE

	compact, err := token.SignedString(am.privateKey)
	return compact, nonce, err
}

// GenerateToken creates a new JWT token for the given user ID
func (am *JWTManager) GenerateToken(
	userID string,
	permissions []auth.Permission,
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
func (am *JWTManager) HasPermission(tokenString string, permission auth.Permission) (bool, error) {
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

// ValidateChallengeJWT validates a challenge JWT token
// The token is a JWT with the following claims:
// - email: the email of the user trying to log in
// - kid: the kid of the user key that must sign the nonce
// - nonce: a 128-bit random nonce
// - exp: the expiration time of the token
func (am *JWTManager) ValidateChallengeJWT(compact string) (*ChallengeClaims, error) {
	token, err := jwt.ParseWithClaims(compact, &ChallengeClaims{}, func(t *jwt.Token) (any, error) {
		if t.Header["typ"] != CHALLENGE_TYPE {
			return nil, fmt.Errorf("not a challenge token")
		}
		return am.publicKey, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims := token.Claims.(*ChallengeClaims)
	if time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("challenge expired")
	}

	return claims, nil
}

// ValidateToken validates and parses a JWT token
func (am *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
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
func (am *JWTManager) loadPrivateKey(path string) ([]byte, error) {
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
func (am *JWTManager) loadPublicKey(path string) ([]byte, error) {
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
func (am *JWTManager) loadPEMFile(path string) (*pem.Block, error) {
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
