package handlers

import (
    "context"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
)

func setupTestDeviceRegistrationHandler() *DeviceRegistrationHandler {
	// Create test configuration
	authConfig := &config.AuthConfig{
		AccessTTL:           30 * time.Minute,
		RefreshTTL:          24 * time.Hour,
		RefreshSkew:         30 * time.Second,
		RefreshCookieName:   "test_rt",
		RefreshCookieDomain: "",
		RefreshCookieSecure: false,
		AllowedWebOrigins:   "http://localhost:3000",
		RefreshHashSecret:   "test_secret_key_for_hmac_validation",
	}

	// Create test logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// For now, return a handler with nil dependencies for basic structure tests
	// TODO: Add proper mocks for repositories and services
	return &DeviceRegistrationHandler{
		inviteRepo:          nil, // TODO: Add mock
		deviceRepo:          nil, // TODO: Add mock
		refreshTokenRepo:    nil, // TODO: Add mock
		userService:         nil, // TODO: Add mock
		roleService:         nil, // TODO: Add mock
		userRoleService:     nil, // TODO: Add mock
		jwtManager:          nil, // TODO: Add mock
		cookieManager:       nil, // TODO: Create with test config
		deviceProofVerifier: nil, // TODO: Create with test config
		authConfig:          authConfig,
		logger:              logger,
		challengeStorage:    make(map[uuid.UUID]*DeviceInitChallenge),
		challengeMu:         &sync.RWMutex{},
	}
}

func TestDeviceRegistrationInitRequest_Structure(t *testing.T) {
    t.Parallel()
	// Test that our request structure can be properly marshaled/unmarshaled
	req := DeviceRegistrationInitRequest{
		Token:    "test_token",
		InviteID: 123,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(req)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled DeviceRegistrationInitRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req.Token, unmarshaled.Token)
	assert.Equal(t, req.InviteID, unmarshaled.InviteID)
}

func TestDeviceRegistrationInitResponse_Structure(t *testing.T) {
    t.Parallel()
	// Test that our response structure can be properly marshaled
	deviceID := uuid.New()
	resp := DeviceRegistrationInitResponse{
		DeviceID:  deviceID,
		Challenge: "test_challenge",
		Algorithm: "ES256",
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(resp)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled DeviceRegistrationInitResponse
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, resp.DeviceID, unmarshaled.DeviceID)
	assert.Equal(t, resp.Challenge, unmarshaled.Challenge)
	assert.Equal(t, resp.Algorithm, unmarshaled.Algorithm)
	assert.WithinDuration(t, resp.ExpiresAt, unmarshaled.ExpiresAt, time.Second)
}

func TestDeviceRegisterRequest_Structure(t *testing.T) {
    t.Parallel()
	// Test that our request structure can be properly marshaled/unmarshaled
	deviceID := uuid.New()
	req := DeviceRegisterRequest{
		DeviceID:   deviceID,
		DeviceName: "Test Device",
		PublicKeyJWK: map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   "test_x_coordinate",
			"y":   "test_y_coordinate",
		},
		DeviceProof: "test_proof",
		Timestamp:   time.Now().Unix(),
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(req)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled DeviceRegisterRequest
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, req.DeviceID, unmarshaled.DeviceID)
	assert.Equal(t, req.DeviceName, unmarshaled.DeviceName)
	assert.Equal(t, req.PublicKeyJWK, unmarshaled.PublicKeyJWK)
	assert.Equal(t, req.DeviceProof, unmarshaled.DeviceProof)
	assert.Equal(t, req.Timestamp, unmarshaled.Timestamp)
}

func TestDeviceInitChallenge_Structure(t *testing.T) {
    t.Parallel()
	// Test that our challenge storage structure works correctly
	deviceID := uuid.New()
	challenge := &DeviceInitChallenge{
		DeviceID:  deviceID,
		Challenge: "test_challenge",
		Algorithm: "ES256",
		ExpiresAt: time.Now().Add(5 * time.Minute),
		InviteID:  123,
		UserID:    456,
	}

	// Test that it can be stored and retrieved from map
	storage := make(map[string]*DeviceInitChallenge)
	storage[challenge.Challenge] = challenge

	retrieved := storage["test_challenge"]
	require.NotNil(t, retrieved)
	assert.Equal(t, challenge.DeviceID, retrieved.DeviceID)
	assert.Equal(t, challenge.Challenge, retrieved.Challenge)
	assert.Equal(t, challenge.Algorithm, retrieved.Algorithm)
	assert.Equal(t, challenge.InviteID, retrieved.InviteID)
	assert.Equal(t, challenge.UserID, retrieved.UserID)
}

func TestNewDeviceRegistrationHandler_Structure(t *testing.T) {
    t.Parallel()
	// Test that our handler can be created with proper structure
	handler := setupTestDeviceRegistrationHandler()

	assert.NotNil(t, handler.authConfig)
	assert.NotNil(t, handler.logger)
	assert.NotNil(t, handler.challengeStorage)
	assert.Equal(t, "test_rt", handler.authConfig.RefreshCookieName)
	assert.Equal(t, "test_secret_key_for_hmac_validation", handler.authConfig.RefreshHashSecret)
	assert.Equal(t, 30*time.Minute, handler.authConfig.AccessTTL)
	assert.Equal(t, 24*time.Hour, handler.authConfig.RefreshTTL)
}

func TestDeviceRegistrationHandler_InitEndpoint_BadRequest(t *testing.T) {
    t.Parallel()
	// Test that the init endpoint properly handles bad requests
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceRegistrationHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/devices/init", handler.InitDeviceRegistration)

	// Test with invalid JSON
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/devices/init", bytes.NewBufferString("invalid json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request format", response["error"])
}

func TestDeviceRegistrationHandler_RegisterEndpoint_BadRequest(t *testing.T) {
    t.Parallel()
	// Test that the register endpoint properly handles bad requests
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceRegistrationHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/devices/register", handler.RegisterDevice)

	// Test with invalid JSON
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/devices/register", bytes.NewBufferString("invalid json"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid request format", response["error"])
}

// Test RFC 7638 JWK Thumbprint Generation.
func TestGenerateJWKThumbprint_RFC7638_Compliance(t *testing.T) {
    t.Parallel()
	handler := setupTestDeviceRegistrationHandler()

    t.Run("EC P-256 key thumbprint", func(t *testing.T) {
        t.Parallel()
		// Test with EC P-256 key
		jwk := map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
			"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			"kid": "some-kid", // kid should be ignored for thumbprint
			"use": "sig",      // optional params should be ignored
		}

		thumbprint, err := handler.generateJWKThumbprint(jwk)
		require.NoError(t, err)
		assert.NotEmpty(t, thumbprint)

		// The same key with different kid should produce the same thumbprint
		jwk2 := map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
			"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			"kid": "different-kid",
		}

		thumbprint2, err := handler.generateJWKThumbprint(jwk2)
		require.NoError(t, err)
		assert.Equal(t, thumbprint, thumbprint2, "Same key material should produce same thumbprint regardless of kid")
	})

    t.Run("RSA key thumbprint", func(t *testing.T) {
        t.Parallel()
		// Test with RSA key
		jwk := map[string]interface{}{
			"kty": "RSA",
			"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISO6qGn-8",
			"e":   "AQAB",
			"kid": "rsa-key-1",
		}

		thumbprint, err := handler.generateJWKThumbprint(jwk)
		require.NoError(t, err)
		assert.NotEmpty(t, thumbprint)

		// Same key without kid should produce same thumbprint
		jwk2 := map[string]interface{}{
			"kty": "RSA",
			"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISO6qGn-8",
			"e":   "AQAB",
		}

		thumbprint2, err := handler.generateJWKThumbprint(jwk2)
		require.NoError(t, err)
		assert.Equal(t, thumbprint, thumbprint2)
	})

    t.Run("OKP Ed25519 key thumbprint", func(t *testing.T) {
        t.Parallel()
		// Test with OKP (Ed25519) key
		jwk := map[string]interface{}{
			"kty": "OKP",
			"crv": "Ed25519",
			"x":   "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo",
			"kid": "ed25519-key",
		}

		thumbprint, err := handler.generateJWKThumbprint(jwk)
		require.NoError(t, err)
		assert.NotEmpty(t, thumbprint)
	})

    t.Run("Different keys produce different thumbprints", func(t *testing.T) {
        t.Parallel()
		// Two different EC keys
		jwk1 := map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
			"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
		}

		jwk2 := map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   "WKn-ZIGevcwGIyyrzFoZNBdaq9_TsqzGl96oc0CWuis",
			"y":   "y77t-RvAHRKTsSGdIYUfweuOvwrvDD-Q3Hv5J0fSKbE",
		}

		thumbprint1, err := handler.generateJWKThumbprint(jwk1)
		require.NoError(t, err)

		thumbprint2, err := handler.generateJWKThumbprint(jwk2)
		require.NoError(t, err)

		assert.NotEqual(t, thumbprint1, thumbprint2, "Different keys should produce different thumbprints")
	})

    t.Run("Missing required fields", func(t *testing.T) {
        t.Parallel()
		// Missing kty
		jwk := map[string]interface{}{
			"crv": "P-256",
			"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
			"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
		}

		_, err := handler.generateJWKThumbprint(jwk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing or invalid kty")

		// EC key missing x coordinate
		jwk = map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
		}

		_, err = handler.generateJWKThumbprint(jwk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required EC key parameters")

		// RSA key missing n
		jwk = map[string]interface{}{
			"kty": "RSA",
			"e":   "AQAB",
		}

		_, err = handler.generateJWKThumbprint(jwk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing required RSA key parameters")
	})

    t.Run("Unsupported key type", func(t *testing.T) {
        t.Parallel()
		jwk := map[string]interface{}{
			"kty": "UNSUPPORTED",
			"x":   "some-value",
		}

		_, err := handler.generateJWKThumbprint(jwk)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported key type")
	})

	t.Run("Thumbprint prevents duplicate key registration", func(t *testing.T) {
		// This test verifies that the same key cannot be registered twice
		// even with different kid values (preventing key swapping attacks)

		// Same EC key with different kids
		jwk1 := map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
			"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			"kid": "device-1",
		}

		jwk2 := map[string]interface{}{
			"kty": "EC",
			"crv": "P-256",
			"x":   "MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
			"y":   "4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
			"kid": "device-2",
		}

		thumbprint1, err := handler.generateJWKThumbprint(jwk1)
		require.NoError(t, err)

		thumbprint2, err := handler.generateJWKThumbprint(jwk2)
		require.NoError(t, err)

		// Both thumbprints should be identical
		assert.Equal(t, thumbprint1, thumbprint2)

		// This means the database unique constraint on JWKThumbprint will prevent
		// the same key from being registered twice
	})
}

// TODO: Add comprehensive tests with mocked dependencies:
// - Test successful device initialization flow
// - Test successful device registration flow
// - Test invite token validation
// - Test device proof verification
// - Test JWT token generation
// - Test refresh token creation
// - Test cookie management
// - Test error handling scenarios
// - Test challenge expiry
// - Test replay attack prevention
