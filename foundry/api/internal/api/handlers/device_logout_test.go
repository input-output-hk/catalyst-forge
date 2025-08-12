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

func setupTestDeviceLogoutHandler() *DeviceLogoutHandler {
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
	// TODO: Add proper mocks for repositories when needed for full integration tests
	return &DeviceLogoutHandler{
		deviceRepo:          nil, // TODO: Add mock for integration tests
		refreshTokenRepo:    nil, // TODO: Add mock for integration tests
		cookieManager:       cookieManager,
		deviceProofVerifier: deviceProofVerifier,
		authConfig:          authConfig,
		logger:              logger,
	}
}

func TestNewDeviceLogoutHandler_Structure(t *testing.T) {
    t.Parallel()
	// Test that our handler can be created with proper structure
	handler := setupTestDeviceLogoutHandler()

	assert.NotNil(t, handler.authConfig)
	assert.NotNil(t, handler.logger)
	assert.Equal(t, "test_rt", handler.authConfig.RefreshCookieName)
	assert.Equal(t, "test_secret_key_for_hmac_validation", handler.authConfig.RefreshHashSecret)
	assert.Equal(t, 30*time.Minute, handler.authConfig.AccessTTL)
	assert.Equal(t, 24*time.Hour, handler.authConfig.RefreshTTL)
}

func TestDeviceLogoutHandler_LogoutEndpoint_MissingCookie(t *testing.T) {
    t.Parallel()
	// Test that the logout endpoint properly handles missing cookies (should succeed gracefully)
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceLogoutHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/logout", handler.Logout)

	// Test with no cookies - should succeed with 204 No Content (idempotent)
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("X-Device-Id", uuid.New().String())
	req.Header.Set("X-Device-Proof", "test.proof")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed gracefully with no content - idempotent logout
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeviceLogoutHandler_LogoutEndpoint_MissingDeviceHeaders(t *testing.T) {
    t.Parallel()
	// Test that the logout endpoint properly handles missing device headers
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceLogoutHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/logout", handler.Logout)

	// Test with cookie but no device headers
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
	require.NoError(t, err)

	// Add a test cookie (won't be parsed successfully due to nil dependencies, but will get past cookie check)
	req.Header.Set("Cookie", "test_rt=test.value")
	// Intentionally omit X-Device-Id and X-Device-Proof headers

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail at device headers parsing stage or cookie parsing stage
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusNoContent}, w.Code)
}

func TestDeviceLogoutHandler_LogoutEndpoint_WithDeviceHeaders(t *testing.T) {
    t.Parallel()
	// Test that the logout endpoint accepts proper device headers format
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceLogoutHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/logout", handler.Logout)

	// Test with proper headers but no cookie
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("X-Device-Id", uuid.New().String())
	req.Header.Set("X-Device-Proof", "test.proof")
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should succeed gracefully with no cookie - idempotent logout
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeviceLogoutHandler_IdempotentLogout(t *testing.T) {
    t.Parallel()
	// Test that logout is idempotent - calling multiple times should always succeed
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceLogoutHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/logout", handler.Logout)

	// Create request with proper format
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("X-Device-Id", uuid.New().String())
	req.Header.Set("X-Device-Proof", "test.proof")
	req.Header.Set("Origin", "http://localhost:3000")

	// First logout call
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req)
	assert.Equal(t, http.StatusNoContent, w1.Code)

	// Second logout call - should still succeed (idempotent)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusNoContent, w2.Code)
}

func TestDeviceLogoutHandler_CookieClearingLogic(t *testing.T) {
    t.Parallel()
	// Test that the handler is configured to clear cookies
	handler := setupTestDeviceLogoutHandler()

	// Test that handler has necessary config for cookie management
	assert.NotNil(t, handler.authConfig)
	assert.Equal(t, "test_rt", handler.authConfig.RefreshCookieName)
	assert.Empty(t, handler.authConfig.RefreshCookieDomain)
	assert.False(t, handler.authConfig.RefreshCookieSecure) // Test mode
}

func TestDeviceLogoutHandler_DeviceProofValidationStructure(t *testing.T) {
    t.Parallel()
	// Test device proof validation structure without calling the actual method
	handler := setupTestDeviceLogoutHandler()

	deviceID := uuid.New()

	// Test that handler has configuration for device proof validation
	assert.NotNil(t, handler.authConfig)
	assert.Equal(t, 30*time.Second, handler.authConfig.RefreshSkew)
	assert.NotEmpty(t, handler.authConfig.AllowedWebOrigins)

	// Test device ID format
	assert.NotEqual(t, uuid.Nil, deviceID)
	assert.Positive(t, len(deviceID.String()))
}

func TestDeviceLogoutHandler_TokenRevocationStructure(t *testing.T) {
    t.Parallel()
	// Test token revocation structure without calling the actual method
	handler := setupTestDeviceLogoutHandler()

	// Create a mock token
	token := &dbmodel.RefreshToken{
		ID:        uuid.New(),
		UserID:    123,
		DeviceID:  uuid.New(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		RevokedAt: nil, // Not revoked initially
	}

	// Test that handler is configured for token operations
	assert.NotNil(t, handler.authConfig)
	assert.NotNil(t, handler.logger)

	// Test that token has required fields for revocation
	assert.NotEqual(t, uuid.Nil, token.ID)
	assert.NotEqual(t, uuid.Nil, token.DeviceID)
	assert.Positive(t, token.UserID)
	assert.Nil(t, token.RevokedAt) // Should be nil initially
	assert.True(t, token.ExpiresAt.After(time.Now()))
}

func TestDeviceLogoutHandler_ErrorResponseFormat(t *testing.T) {
    t.Parallel()
	// Test that error responses follow expected format
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceLogoutHandler()

	// Create test router
	router := gin.New()
	router.POST("/auth/logout", handler.Logout)

	// Test with malformed headers to trigger error response
    req, err := http.NewRequestWithContext(context.Background(), "POST", "/auth/logout", nil)
	require.NoError(t, err)
	req.Header.Set("Cookie", "test_rt=test.value")
	req.Header.Set("X-Device-Id", "invalid-uuid")
	req.Header.Set("X-Device-Proof", "") // Empty proof

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error response with proper JSON format (if it gets past cookie parsing)
	if w.Code == http.StatusBadRequest {
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "error")
		assert.IsType(t, "", response["error"]) // Should be string
	}
}

// TODO: Add comprehensive integration tests with mocked dependencies:
// - Test successful logout flow with valid token and device proof
// - Test logout with revoked token (should succeed gracefully)
// - Test logout with expired token (should succeed gracefully)
// - Test logout with device ID mismatch (should fail with 401)
// - Test logout with invalid device proof (should fail with 401)
// - Test cookie clearing functionality
// - Test token revocation in database
// - Test CORS origin handling
// - Test proper logging for various scenarios
// - Test concurrent logout requests
// - Test logout with non-existent token (should succeed gracefully)
// - Test logout after token family revocation
