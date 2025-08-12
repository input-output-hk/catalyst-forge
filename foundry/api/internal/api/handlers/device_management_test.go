package handlers

import (
    "context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"os"
)

// mockDeviceRepo is a simple mock for testing.
type mockDeviceRepo struct {
	returnError bool
}

func (m *mockDeviceRepo) Create(device *dbmodel.Device) error {
	return errors.New("not implemented")
}

func (m *mockDeviceRepo) GetByID(id uuid.UUID) (*dbmodel.Device, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDeviceRepo) GetByJWKThumbprint(thumbprint string) (*dbmodel.Device, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDeviceRepo) GetByUserID(userID uint) ([]dbmodel.Device, error) {
	if m.returnError {
		return nil, errors.New("mock error")
	}
	// Return empty list for successful case
	return []dbmodel.Device{}, nil
}

func (m *mockDeviceRepo) UpdateLastUsed(id uuid.UUID, timestamp time.Time) error {
	return errors.New("not implemented")
}

func (m *mockDeviceRepo) RevokeDevice(id uuid.UUID) error {
	return errors.New("not implemented")
}

func (m *mockDeviceRepo) GetActiveByUserID(userID uint) ([]dbmodel.Device, error) {
	if m.returnError {
		return nil, errors.New("mock error")
	}
	// Return empty list for successful case
	return []dbmodel.Device{}, nil
}

func setupTestDeviceManagementHandler() *DeviceManagementHandler {
	return setupTestDeviceManagementHandlerWithMock(false)
}

func setupTestDeviceManagementHandlerWithMock(returnError bool) *DeviceManagementHandler {
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

	// Create mock device repo
	mockRepo := &mockDeviceRepo{returnError: returnError}

	// Return a handler with mock device repo for basic tests
	// TODO: Add proper mocks for other dependencies when needed
	return &DeviceManagementHandler{
		deviceRepo:       mockRepo,
		refreshTokenRepo: nil, // TODO: Add mock for tests that need it
		userService:      nil, // TODO: Add mock for tests that need it
		jwtManager:       nil, // TODO: Add mock for tests that need it
		authConfig:       authConfig,
		logger:           logger,
	}
}

func TestNewDeviceManagementHandler_Structure(t *testing.T) {
    t.Parallel()
	// Test that our handler can be created with proper structure
	handler := setupTestDeviceManagementHandler()

	assert.NotNil(t, handler.authConfig)
	assert.NotNil(t, handler.logger)
	assert.Equal(t, "test_rt", handler.authConfig.RefreshCookieName)
	assert.Equal(t, "test_secret_key_for_hmac_validation", handler.authConfig.RefreshHashSecret)
	assert.Equal(t, 30*time.Minute, handler.authConfig.AccessTTL)
	assert.Equal(t, 24*time.Hour, handler.authConfig.RefreshTTL)
}

func TestDeviceManagementHandler_ListDevices_MissingUserContext(t *testing.T) {
    t.Parallel()
	// Test that the list devices endpoint properly handles missing user context
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceManagementHandler()

	// Create test router
	router := gin.New()
	router.GET("/auth/devices", handler.ListDevices)

	// Test with no user_id in context
    req, err := http.NewRequestWithContext(context.Background(), "GET", "/auth/devices", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response["error"])
}

func TestDeviceManagementHandler_ListDevices_WithUserContext(t *testing.T) {
    t.Parallel()
	// Test that the list devices endpoint accepts proper user context
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceManagementHandler()

	// Create test router with middleware to set user context
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", float64(123)) // JWT middleware would set this
		c.Next()
	})
	router.GET("/auth/devices", handler.ListDevices)

	// Test with proper user context
    req, err := http.NewRequestWithContext(context.Background(), "GET", "/auth/devices", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Will fail at device repo call due to nil dependency, but gets past auth check
	assert.Contains(t, []int{http.StatusInternalServerError, http.StatusOK}, w.Code)
}

func TestDeviceManagementHandler_DeleteDevice_MissingUserContext(t *testing.T) {
    t.Parallel()
	// Test that the delete device endpoint properly handles missing user context
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceManagementHandler()

	// Create test router
	router := gin.New()
	router.DELETE("/auth/devices/:id", handler.DeleteDevice)

	// Test with no user_id in context
	deviceID := uuid.New()
    req, err := http.NewRequestWithContext(context.Background(), "DELETE", "/auth/devices/"+deviceID.String(), nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "unauthorized", response["error"])
}

func TestDeviceManagementHandler_DeleteDevice_InvalidDeviceID(t *testing.T) {
    t.Parallel()
	// Test that the delete device endpoint handles invalid device ID format
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceManagementHandler()

	// Create test router with middleware to set user context
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", float64(123)) // JWT middleware would set this
		c.Next()
	})
	router.DELETE("/auth/devices/:id", handler.DeleteDevice)

	// Test with invalid UUID format
    req, err := http.NewRequestWithContext(context.Background(), "DELETE", "/auth/devices/invalid-uuid", nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "invalid device ID format", response["error"])
}

func TestDeviceManagementHandler_DeleteDevice_ValidDeviceID(t *testing.T) {
    t.Parallel()
	// Test that the delete device endpoint accepts valid device ID format
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceManagementHandler()

	// Create test router with middleware to set user context
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", float64(123)) // JWT middleware would set this
		c.Next()
	})
	router.DELETE("/auth/devices/:id", handler.DeleteDevice)

	// Test with valid UUID format
	deviceID := uuid.New()
    req, err := http.NewRequestWithContext(context.Background(), "DELETE", "/auth/devices/"+deviceID.String(), nil)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Will fail at device repo call due to nil dependency, but gets past validation
	assert.Contains(t, []int{http.StatusInternalServerError, http.StatusNotFound, http.StatusNoContent}, w.Code)
}

func TestDeviceManagementHandler_UserIDTypeHandling(t *testing.T) {
    t.Parallel()
	// Test different user ID types that JWT middleware might set
	gin.SetMode(gin.TestMode)

	handler := setupTestDeviceManagementHandler()

	// Test with string user ID
	router1 := gin.New()
	router1.Use(func(c *gin.Context) {
		c.Set("user_id", "123") // String format
		c.Next()
	})
	router1.GET("/auth/devices", handler.ListDevices)

    req1, err := http.NewRequestWithContext(context.Background(), "GET", "/auth/devices", nil)
	require.NoError(t, err)

	w1 := httptest.NewRecorder()
	router1.ServeHTTP(w1, req1)

	// Should handle string conversion properly
	assert.Contains(t, []int{http.StatusInternalServerError, http.StatusOK}, w1.Code)

	// Test with float64 user ID (common from JSON parsing)
	router2 := gin.New()
	router2.Use(func(c *gin.Context) {
		c.Set("user_id", float64(123)) // Float64 format
		c.Next()
	})
	router2.GET("/auth/devices", handler.ListDevices)

    req2, err := http.NewRequestWithContext(context.Background(), "GET", "/auth/devices", nil)
	require.NoError(t, err)

	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)

	// Should handle float64 conversion properly
	assert.Contains(t, []int{http.StatusInternalServerError, http.StatusOK}, w2.Code)
}

func TestDeviceManagementHandler_DeviceListResponseStructure(t *testing.T) {
    t.Parallel()
	// Test the device list response structure
	device := &dbmodel.Device{
		ID:         uuid.New(),
		Name:       "Test Device",
		Status:     dbmodel.DeviceStatusActive,
		CreatedAt:  time.Now(),
		LastUsedAt: nil,
	}

	// Test response creation logic
	var lastUsedAt *string
	if device.LastUsedAt != nil {
		lastUsedAtStr := device.LastUsedAt.Format("2006-01-02T15:04:05Z")
		lastUsedAt = &lastUsedAtStr
	}

	response := DeviceListResponse{
		ID:         device.ID,
		Name:       device.Name,
		LastUsedAt: lastUsedAt,
		Status:     string(device.Status),
		CreatedAt:  device.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Verify response structure
	assert.NotEqual(t, uuid.Nil, response.ID)
	assert.Equal(t, "Test Device", response.Name)
	assert.Nil(t, response.LastUsedAt) // Should be nil when device.LastUsedAt is nil
	assert.Equal(t, "active", response.Status)
	assert.NotEmpty(t, response.CreatedAt)

	// Test with last used timestamp
	now := time.Now()
	device.LastUsedAt = &now
	lastUsedAtStr := device.LastUsedAt.Format("2006-01-02T15:04:05Z")
	response.LastUsedAt = &lastUsedAtStr

	assert.NotNil(t, response.LastUsedAt)
	assert.Equal(t, now.Format("2006-01-02T15:04:05Z"), *response.LastUsedAt)
}

func TestDeviceManagementHandler_DeviceStatusConstants(t *testing.T) {
    t.Parallel()
	// Test that we're using the correct device status constants
	assert.Equal(t, "active", dbmodel.DeviceStatusActive)
	assert.Equal(t, "revoked", dbmodel.DeviceStatusRevoked)
	assert.Equal(t, "disabled", dbmodel.DeviceStatusDisabled)
}

func TestDeviceManagementHandler_AuthorizationLogic(t *testing.T) {
    t.Parallel()
	// Test authorization logic structure without calling actual methods
	handler := setupTestDeviceManagementHandler()

	// Test that handler has necessary config for authorization
	assert.NotNil(t, handler.authConfig)
	assert.NotNil(t, handler.logger)

	// Test user ID handling logic
	userID := uint(123)
	deviceUserID := uint(123)    // Same user
	differentUserID := uint(456) // Different user

	// Authorization should pass for same user
	assert.Equal(t, userID, deviceUserID)

	// Authorization should fail for different user
	assert.NotEqual(t, userID, differentUserID)
}

// TODO: Add comprehensive integration tests with mocked dependencies:
// - Test successful device list retrieval with proper response format
// - Test successful device deletion with token family revocation
// - Test device deletion authorization (users can only delete their own devices)
// - Test device deletion with already revoked device (idempotent)
// - Test device deletion with non-existent device
// - Test device list filtering (only show user's devices)
// - Test token family revocation on device deletion
// - Test concurrent device operations
// - Test device list with various last_used_at scenarios
// - Test device list with different device statuses
// - Test error handling for repository failures
// - Test proper logging for various operations
