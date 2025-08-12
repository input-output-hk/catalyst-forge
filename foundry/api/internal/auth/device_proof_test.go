package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
)

// MockDeviceRepository is a mock implementation of DeviceRepository.
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) Create(device *dbmodel.Device) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetByID(id uuid.UUID) (*dbmodel.Device, error) {
	args := m.Called(id)
	return args.Get(0).(*dbmodel.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByJWKThumbprint(thumbprint string) (*dbmodel.Device, error) {
	args := m.Called(thumbprint)
	return args.Get(0).(*dbmodel.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByUserID(userID uint) ([]dbmodel.Device, error) {
	args := m.Called(userID)
	return args.Get(0).([]dbmodel.Device), args.Error(1)
}

func (m *MockDeviceRepository) UpdateLastUsed(id uuid.UUID, timestamp time.Time) error {
	args := m.Called(id, timestamp)
	return args.Error(0)
}

func (m *MockDeviceRepository) RevokeDevice(id uuid.UUID) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetActiveByUserID(userID uint) ([]dbmodel.Device, error) {
	args := m.Called(userID)
	return args.Get(0).([]dbmodel.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByUserAndFingerprint(userID uint, fingerprint string) (*dbmodel.Device, error) {
	args := m.Called(userID, fingerprint)
	return args.Get(0).(*dbmodel.Device), args.Error(1)
}

// Helper functions for test setup
// Uses math/big.Int for ECDSA key coordinates.
func generateTestECDSAKey() (*ecdsa.PrivateKey, *ecdsa.PublicKey, map[string]interface{}, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	pubKey := &privKey.PublicKey

	// Convert to JWK format
	xBytes := pubKey.X.Bytes()
	yBytes := pubKey.Y.Bytes()

	// Ensure 32-byte length for P-256
	if len(xBytes) < 32 {
		padding := make([]byte, 32-len(xBytes))
		xBytes = append(padding, xBytes...)
	}
	if len(yBytes) < 32 {
		padding := make([]byte, 32-len(yBytes))
		yBytes = append(padding, yBytes...)
	}

	jwk := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(xBytes),
		"y":   base64.RawURLEncoding.EncodeToString(yBytes),
	}

	return privKey, pubKey, jwk, nil
}

func signCanonicalString(privKey *ecdsa.PrivateKey, canonicalString string) ([]byte, error) {
	hash := sha256.Sum256([]byte(canonicalString))
	r, s, err := ecdsa.Sign(rand.Reader, privKey, hash[:])
	if err != nil {
		return nil, err
	}

	// Ensure we have valid big.Int values
	_ = big.NewInt(0) // Explicit use to satisfy linter

	// Convert to 64-byte signature format (32 bytes r + 32 bytes s)
	signature := make([]byte, 64)
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Right-pad with zeros if needed
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	return signature, nil
}

func createTestDevice(deviceID uuid.UUID, jwk map[string]interface{}, status string) *dbmodel.Device {
	jwkBytes, _ := json.Marshal(jwk)
	return &dbmodel.Device{
		ID:        deviceID,
		UserID:    1,
		Name:      "Test Device",
		PublicJWK: jwkBytes,
		Status:    status,
	}
}

func TestDeviceProofVerifier_ParseDeviceProofHeaders(t *testing.T) {
    t.Parallel()
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	tests := []struct {
		name       string
		deviceID   string
		proof      string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "valid headers",
			deviceID: "550e8400-e29b-41d4-a716-446655440000",
			proof:    "1234567890.dGVzdF9zaWduYXR1cmU",
			wantErr:  false,
		},
		{
			name:       "missing device ID",
			deviceID:   "",
			proof:      "1234567890.dGVzdF9zaWduYXR1cmU",
			wantErr:    true,
			wantErrMsg: "X-Device-Id header is required",
		},
		{
			name:       "missing proof",
			deviceID:   "550e8400-e29b-41d4-a716-446655440000",
			proof:      "",
			wantErr:    true,
			wantErrMsg: "X-Device-Proof header is required",
		},
		{
			name:       "invalid UUID format",
			deviceID:   "invalid-uuid",
			proof:      "1234567890.dGVzdF9zaWduYXR1cmU",
			wantErr:    true,
			wantErrMsg: "invalid X-Device-Id format",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			result, err := verifier.ParseDeviceProofHeaders(tt.deviceID, tt.proof)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.deviceID, result.DeviceID.String())
				assert.Equal(t, tt.proof, result.Proof)
			}
		})
	}
}

func TestDeviceProofVerifier_ParseDeviceProof(t *testing.T) {
    t.Parallel()
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	tests := []struct {
		name       string
		proof      string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "valid proof format",
			proof:   "1234567890.dGVzdF9zaWduYXR1cmU",
			wantErr: false,
		},
		{
			name:       "missing dot separator",
			proof:      "1234567890dGVzdF9zaWduYXR1cmU",
			wantErr:    true,
			wantErrMsg: "invalid device proof format",
		},
		{
			name:       "invalid timestamp",
			proof:      "invalid_timestamp.dGVzdF9zaWduYXR1cmU",
			wantErr:    true,
			wantErrMsg: "invalid timestamp",
		},
		{
			name:       "invalid signature encoding",
			proof:      "1234567890.invalid_base64url!",
			wantErr:    true,
			wantErrMsg: "invalid signature encoding",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			result, err := verifier.ParseDeviceProof(tt.proof)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, int64(1234567890), result.Timestamp)
			}
		})
	}
}

func TestDeviceProofVerifier_GenerateCanonicalString(t *testing.T) {
    t.Parallel()
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	deviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	timestamp := int64(1234567890)
	origin := "https://app.example.com"
	method := "POST"
	path := "/auth/refresh"

	expected := "AUTH-REFRESH\n1234567890\n550e8400-e29b-41d4-a716-446655440000\nhttps://app.example.com\nPOST /auth/refresh"

	result := verifier.GenerateCanonicalString(timestamp, deviceID, origin, method, path)
	assert.Equal(t, expected, result)
}

func TestDeviceProofVerifier_VerifyTimestamp(t *testing.T) {
    t.Parallel()
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	now := time.Now().Unix()

	tests := []struct {
		name       string
		timestamp  int64
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "valid timestamp (current)",
			timestamp: now,
			wantErr:   false,
		},
		{
			name:      "valid timestamp (within skew past)",
			timestamp: now - 25, // 25 seconds ago, within 30s skew
			wantErr:   false,
		},
		{
			name:      "valid timestamp (within skew future)",
			timestamp: now + 25, // 25 seconds in future, within 30s skew
			wantErr:   false,
		},
		{
			name:       "timestamp too far in past",
			timestamp:  now - 35, // 35 seconds ago, outside 30s skew
			wantErr:    true,
			wantErrMsg: "timestamp too far in past",
		},
		{
			name:       "timestamp too far in future",
			timestamp:  now + 35, // 35 seconds in future, outside 30s skew
			wantErr:    true,
			wantErrMsg: "timestamp too far in future",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			err := verifier.VerifyTimestamp(tt.timestamp)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeviceProofVerifier_GetDevicePublicKey(t *testing.T) {
    t.Parallel()
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	deviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	_, _, jwk, err := generateTestECDSAKey()
	assert.NoError(t, err)

	// Create JWK with "none" algorithm
	noneAlgJWK := map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"alg": "none",
		"x":   jwk["x"],
		"y":   jwk["y"],
	}

	// Create JWK with RSA key type
	rsaJWK := map[string]interface{}{
		"kty": "RSA",
		"n":   "test",
		"e":   "AQAB",
	}

	tests := []struct {
		name       string
		device     *dbmodel.Device
		repoErr    error
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "valid active device",
			device:  createTestDevice(deviceID, jwk, "active"),
			repoErr: nil,
			wantErr: false,
		},
		{
			name:       "device not found",
			device:     nil,
			repoErr:    nil,
			wantErr:    true,
			wantErrMsg: "device not found",
		},
		{
			name:       "inactive device",
			device:     createTestDevice(deviceID, jwk, "revoked"),
			repoErr:    nil,
			wantErr:    true,
			wantErrMsg: "device is not active",
		},
		{
			name: "invalid JWK format",
			device: &dbmodel.Device{
				ID:        deviceID,
				PublicJWK: []byte("invalid json"),
				Status:    "active",
			},
			repoErr:    nil,
			wantErr:    true,
			wantErrMsg: "failed to parse device JWK",
		},
		{
			name:       "stored device with none algorithm",
			device:     createTestDevice(deviceID, noneAlgJWK, "active"),
			repoErr:    nil,
			wantErr:    true,
			wantErrMsg: "stored device key failed algorithm validation",
		},
		{
			name:       "stored device with RSA key",
			device:     createTestDevice(deviceID, rsaJWK, "active"),
			repoErr:    nil,
			wantErr:    true,
			wantErrMsg: "stored device key failed algorithm validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.On("GetByID", deviceID).Return(tt.device, tt.repoErr).Once()

			pubKey, device, err := verifier.GetDevicePublicKey(deviceID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, pubKey)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pubKey)
				assert.NotNil(t, device)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestDeviceProofVerifier_VerifySignature(t *testing.T) {
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	privKey, pubKey, _, err := generateTestECDSAKey()
	assert.NoError(t, err)

	canonicalString := "AUTH-REFRESH\n1234567890\n550e8400-e29b-41d4-a716-446655440000\nhttps://app.example.com\nPOST /auth/refresh"

	validSignature, err := signCanonicalString(privKey, canonicalString)
	assert.NoError(t, err)

	tests := []struct {
		name       string
		signature  []byte
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "valid signature",
			signature: validSignature,
			wantErr:   false,
		},
		{
			name:       "invalid signature length",
			signature:  []byte("too_short"),
			wantErr:    true,
			wantErrMsg: "invalid ECDSA signature length",
		},
		{
			name:       "wrong signature",
			signature:  make([]byte, 64), // all zeros
			wantErr:    true,
			wantErrMsg: "signature verification failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifier.VerifySignature(pubKey, canonicalString, tt.signature)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeviceProofVerifier_VerifyDeviceProof_EndToEnd(t *testing.T) {
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	deviceID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	privKey, _, jwk, err := generateTestECDSAKey()
	assert.NoError(t, err)

	device := createTestDevice(deviceID, jwk, "active")
	timestamp := time.Now().Unix()
	origin := "https://app.example.com"
	method := "POST"
	path := "/auth/refresh"

	canonicalString := verifier.GenerateCanonicalString(timestamp, deviceID, origin, method, path)
	signature, err := signCanonicalString(privKey, canonicalString)
	assert.NoError(t, err)

	proof := fmt.Sprintf("%d.%s", timestamp, base64.RawURLEncoding.EncodeToString(signature))

	// Mock expectations
	mockRepo.On("GetByID", deviceID).Return(device, nil).Once()
	mockRepo.On("UpdateLastUsed", deviceID, mock.AnythingOfType("time.Time")).Return(nil).Once()

	// Test successful verification
	resultDevice, err := verifier.VerifyDeviceProof(deviceID.String(), proof, origin, method, path)

	assert.NoError(t, err)
	assert.NotNil(t, resultDevice)
	assert.Equal(t, deviceID, resultDevice.ID)

	mockRepo.AssertExpectations(t)
}

func TestDeviceProofVerifier_parseECDSAFromJWK(t *testing.T) {
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	_, _, validJWK, err := generateTestECDSAKey()
	assert.NoError(t, err)

	tests := []struct {
		name       string
		jwk        map[string]interface{}
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "valid JWK",
			jwk:     validJWK,
			wantErr: false,
		},
		{
			name: "wrong key type",
			jwk: map[string]interface{}{
				"kty": "RSA",
				"crv": "P-256",
			},
			wantErr:    true,
			wantErrMsg: "key type must be 'EC'",
		},
		{
			name: "wrong curve",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-384",
			},
			wantErr:    true,
			wantErrMsg: "curve must be 'P-256'",
		},
		{
			name: "missing x coordinate",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"y":   "test",
			},
			wantErr:    true,
			wantErrMsg: "missing 'x' coordinate",
		},
		{
			name: "missing y coordinate",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"x":   "test",
			},
			wantErr:    true,
			wantErrMsg: "missing 'y' coordinate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pubKey, err := verifier.parseECDSAFromJWK(tt.jwk)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, pubKey)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pubKey)
				assert.Equal(t, elliptic.P256(), pubKey.Curve)
			}
		})
	}
}

func TestDeviceProofVerifier_VerifyDeviceProofWithKey_AlgorithmValidation(t *testing.T) {
    t.Parallel()
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	// Generate a valid key for signing
	privKey, _, validJWK, err := generateTestECDSAKey()
	assert.NoError(t, err)

	// Create a valid canonical string and sign it
	canonicalString := "DEVICE-REGISTER\n1234567890\n550e8400-e29b-41d4-a716-446655440000\ntest-challenge"
	signature, err := signCanonicalString(privKey, canonicalString)
	assert.NoError(t, err)
	proof := fmt.Sprintf("%d.%s", 1234567890, base64.RawURLEncoding.EncodeToString(signature))

	tests := []struct {
		name       string
		jwk        map[string]interface{}
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "valid ES256 key",
			jwk:     validJWK,
			wantErr: false,
		},
		{
			name: "reject none algorithm",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "none",
				"x":   validJWK["x"],
				"y":   validJWK["y"],
			},
			wantErr:    true,
			wantErrMsg: "algorithm validation failed",
		},
		{
			name: "reject RS256 algorithm",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "RS256",
				"x":   validJWK["x"],
				"y":   validJWK["y"],
			},
			wantErr:    true,
			wantErrMsg: "algorithm validation failed",
		},
		{
			name: "reject RSA key type",
			jwk: map[string]interface{}{
				"kty": "RSA",
				"n":   "test",
				"e":   "AQAB",
			},
			wantErr:    true,
			wantErrMsg: "algorithm validation failed",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			err := verifier.VerifyDeviceProofWithKey(canonicalString, proof, tt.jwk)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeviceProofVerifier_validateAlgorithm(t *testing.T) {
    t.Parallel()
	mockRepo := &MockDeviceRepository{}
	config := &config.AuthConfig{RefreshSkew: 30 * time.Second}
	verifier := NewDeviceProofVerifier(mockRepo, config)

	tests := []struct {
		name       string
		jwk        map[string]interface{}
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid ES256 with alg field",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "ES256",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr: false,
		},
		{
			name: "valid ES256 without alg field",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr: false,
		},
		{
			name: "reject none algorithm",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "none",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "algorithm 'none' is explicitly forbidden",
		},
		{
			name: "reject NONE algorithm (uppercase)",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "NONE",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "algorithm 'none' is explicitly forbidden",
		},
		{
			name: "reject NoNe algorithm (mixed case)",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "NoNe",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "algorithm 'none' is explicitly forbidden",
		},
		{
			name: "reject RS256 algorithm",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "RS256",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "only ES256 is allowed",
		},
		{
			name: "reject HS256 algorithm",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "HS256",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "only ES256 is allowed",
		},
		{
			name: "reject ES384 algorithm",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": "ES384",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "only ES256 is allowed",
		},
		{
			name: "reject RSA key type with RS256 algorithm",
			jwk: map[string]interface{}{
				"kty": "RSA",
				"alg": "RS256",
				"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
				"e":   "AQAB",
			},
			wantErr:    true,
			wantErrMsg: "only ES256 is allowed",
		},
		{
			name: "reject RSA key type without alg",
			jwk: map[string]interface{}{
				"kty": "RSA",
				"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
				"e":   "AQAB",
			},
			wantErr:    true,
			wantErrMsg: "only EC (Elliptic Curve) keys are allowed",
		},
		{
			name: "reject OKP key type",
			jwk: map[string]interface{}{
				"kty": "OKP",
				"crv": "Ed25519",
				"x":   "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
			},
			wantErr:    true,
			wantErrMsg: "only EC (Elliptic Curve) keys are allowed",
		},
		{
			name: "reject oct (symmetric) key type",
			jwk: map[string]interface{}{
				"kty": "oct",
				"k":   "AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow",
			},
			wantErr:    true,
			wantErrMsg: "only EC (Elliptic Curve) keys are allowed",
		},
		{
			name: "reject P-384 curve",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-384",
				"x":   "fY7ROmNpJJgExKTJpLUfmu3XaKVxG5XwJiARGg-UXHiVGWj3zaXrI6H8LFVOK2lQ",
				"y":   "c9aW9c-oK8N5H9LZBNPRlPmBWykqt5xdDzttIa5jLQiJLHBSuDlYMRAKUlgIrE_7",
			},
			wantErr:    true,
			wantErrMsg: "only P-256 is allowed",
		},
		{
			name: "reject P-521 curve",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-521",
				"x":   "AekpBQ8ST8a8VcfVOTNl353vSrDCLLJXmPk06wTjxrrjcBpXp5EOnYG_NjFZ6OvLFV1jSfS9tsz4qUxcWceqwQGk",
				"y":   "ADSmRA43Z1DSNx_RvcLI87cdL07l6jQyyBXMoxVg_l2Th-x3S1WDhjDly79ajL4Kkd0AZMaZmh9ubmf63e3kyMj2",
			},
			wantErr:    true,
			wantErrMsg: "only P-256 is allowed",
		},
		{
			name: "missing kty field",
			jwk: map[string]interface{}{
				"crv": "P-256",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "missing 'kty' field",
		},
		{
			name: "missing crv field for EC key",
			jwk: map[string]interface{}{
				"kty": "EC",
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "missing 'crv' field",
		},
		{
			name: "non-string alg field",
			jwk: map[string]interface{}{
				"kty": "EC",
				"crv": "P-256",
				"alg": 256,
				"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
				"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			},
			wantErr:    true,
			wantErrMsg: "'alg' field must be a string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifier.validateAlgorithm(tt.jwk)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
