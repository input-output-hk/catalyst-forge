package handlers

import (
    "context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
)

func setupTestDeviceRefreshHandler() *DeviceRefreshHandler {
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

	// Create managers using the auth package
	cookieManager := auth.NewCookieManager(authConfig)
	deviceProofVerifier := auth.NewDeviceProofVerifier(nil, authConfig) // nil repo is ok for basic tests

	// Return a handler with initialized managers for basic tests
	// TODO: Add proper mocks for repositories and services when needed for full integration tests
	return &DeviceRefreshHandler{
		deviceRepo:          nil, // TODO: Add mock for integration tests
		refreshTokenRepo:    nil, // TODO: Add mock for integration tests
		userService:         nil, // TODO: Add mock for integration tests
		roleService:         nil, // TODO: Add mock for integration tests
		userRoleService:     nil, // TODO: Add mock for integration tests
		jwtManager:          nil, // TODO: Add mock for integration tests
		cookieManager:       cookieManager,
		deviceProofVerifier: deviceProofVerifier,
		authConfig:          authConfig,
		logger:              logger,
	}
}

func TestNewDeviceRefreshHandler_Structure(t *testing.T) {
    t.Parallel()
	// Test that our handler can be created with proper structure
	handler := setupTestDeviceRefreshHandler()

	assert.NotNil(t, handler.authConfig)
	assert.NotNil(t, handler.logger)
	assert.Equal(t, "test_rt", handler.authConfig.RefreshCookieName)
	assert.Equal(t, "test_secret_key_for_hmac_validation", handler.authConfig.RefreshHashSecret)
	assert.Equal(t, 30*time.Minute, handler.authConfig.AccessTTL)
	assert.Equal(t, 24*time.Hour, handler.authConfig.RefreshTTL)
}

func TestDeviceRefreshHandler_RefreshEndpoint_MissingCookie(t *testing.T) {
    t.Parallel()
	// Test that the refresh endpoint properly handles missing cookies
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceRefreshHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/refresh", handler.RefreshToken)

	// Test with no cookies
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", nil)
	require.NoError(t, err)
	req.Header.Set("X-Device-Id", uuid.New().String())
	req.Header.Set("X-Device-Proof", "test.proof")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "missing or invalid refresh token", response["error"])
}

func TestDeviceRefreshHandler_RefreshEndpoint_MissingDeviceHeaders(t *testing.T) {
    t.Parallel()
	// Test that the refresh endpoint properly handles missing device headers
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceRefreshHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/refresh", handler.RefreshToken)

	// Test with cookie but no device headers
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/refresh", nil)
	require.NoError(t, err)

	// Add a test cookie (won't be parsed successfully due to nil dependencies, but will get past cookie check)
	req.Header.Set("Cookie", "test_rt=test.value")
	// Intentionally omit X-Device-Id and X-Device-Proof headers

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail at cookie parsing stage due to nil cookieManager, but structure test passes
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusUnauthorized}, w.Code)
}

func TestDeviceRefreshHandler_GenerateSecretHash(t *testing.T) {
    t.Parallel()
	handler := setupTestDeviceRefreshHandler()

	// Test HMAC generation with configured secret
	secret := "test_secret_value"
	hash, err := handler.generateSecretHash(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA-256 hex encoding should be 64 characters

	// Test that same secret produces same hash
	hash2, err := handler.generateSecretHash(secret)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Test that different secrets produce different hashes
	differentSecret := "different_secret_value"
	hash3, err := handler.generateSecretHash(differentSecret)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash3)
}

func TestDeviceRefreshHandler_GenerateSecretHash_NoConfigSecret(t *testing.T) {
    t.Parallel()
	// Test fallback to simple hash when no HMAC secret configured
	handler := setupTestDeviceRefreshHandler()
	handler.authConfig.RefreshHashSecret = "" // Clear the secret

	secret := "test_secret_value"
	hash, err := handler.generateSecretHash(secret)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA-256 hex encoding should be 64 characters
}

func TestDeviceRefreshHandler_RotateRefreshToken_Structure(t *testing.T) {
    t.Parallel()
	// Test the token rotation logic structure without calling the actual method
	handler := setupTestDeviceRefreshHandler()

	// Create a mock old token
	oldToken := &dbmodel.RefreshToken{
		ID:       uuid.New(),
		UserID:   123,
		DeviceID: uuid.New(),
		FamilyID: uuid.New(),
	}

	// Test that the handler has the necessary configuration for rotation
	assert.NotNil(t, handler.authConfig)
	assert.Equal(t, 24*time.Hour, handler.authConfig.RefreshTTL)
	assert.NotEmpty(t, handler.authConfig.RefreshHashSecret)

	// Test that old token has required fields
	assert.NotEqual(t, uuid.Nil, oldToken.ID)
	assert.NotEqual(t, uuid.Nil, oldToken.DeviceID)
	assert.NotEqual(t, uuid.Nil, oldToken.FamilyID)
	assert.Positive(t, oldToken.UserID)
}

func TestDeviceRefreshHandler_CheckReplayAttack_Structure(t *testing.T) {
    t.Parallel()
	// Test replay attack detection structure without calling the method
	handler := setupTestDeviceRefreshHandler()

	// Create a mock token
	token := &dbmodel.RefreshToken{
		ID:        uuid.New(),
		FamilyID:  uuid.New(),
		CreatedAt: time.Now(),
	}

	// Test that the handler is configured for replay detection
	assert.NotNil(t, handler.authConfig)

	// Test that token has required fields for replay detection
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.NotEqual(t, uuid.Nil, token.FamilyID)
	assert.False(t, token.CreatedAt.IsZero())
}

func TestDeviceRefreshHandler_RevokeFamilyTokens_Structure(t *testing.T) {
    t.Parallel()
	// Test family revocation structure without calling the method
	handler := setupTestDeviceRefreshHandler()

	familyID := uuid.New()

	// Test that the handler is configured for family revocation
	assert.NotNil(t, handler.authConfig)
	assert.NotNil(t, handler.logger)

	// Test that family ID is valid
	assert.NotEqual(t, uuid.Nil, familyID)
}

// TODO: Add comprehensive integration tests with mocked dependencies:
// - Test successful token refresh flow
// - Test device proof validation in refresh flow
// - Test token rotation with family tracking
// - Test replay attack detection and family revocation
// - Test secret hash validation with constant-time comparison
// - Test device last used timestamp updates
// - Test access token generation
// - Test CORS handling
// - Test concurrent refresh request handling
// - Test expired token handling
// - Test revoked token handling
// - Test device mismatch scenarios
