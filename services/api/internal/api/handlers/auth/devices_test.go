package auth

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
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/testing/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDevicesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	deviceID1 := uuid.New()
	deviceID2 := uuid.New()
	currentDeviceID := uuid.New()

	t.Run("ok/list_devices", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context with device ID
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:   userID,
				Email:    "user@example.com",
				DeviceID: &currentDeviceID,
			}
			authCtx.Set(c)
			c.Next()
		})

		// Create device store with test data
		deviceStore := inmemory.NewDeviceStore()
		ctx := context.Background()

		// Add devices
		device1 := &domain.Device{
			ID:         deviceID1,
			UserID:     userID,
			DeviceName: "Device 1",
			CreatedAt:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		}
		device2 := &domain.Device{
			ID:         deviceID2,
			UserID:     userID,
			DeviceName: "Device 2",
			CreatedAt:  time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 14, 10, 0, 0, 0, time.UTC),
		}
		currentDevice := &domain.Device{
			ID:         currentDeviceID,
			UserID:     userID,
			DeviceName: "Current Device",
			CreatedAt:  time.Date(2025, 1, 10, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
		}

		require.NoError(t, deviceStore.Create(ctx, device1))
		require.NoError(t, deviceStore.Create(ctx, device2))
		require.NoError(t, deviceStore.Create(ctx, currentDevice))

		handler := listDevicesHandler(deviceStore)
		r.GET("/devices", handler)

		req := httptest.NewRequest("GET", "/devices", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		devices := resp["devices"].([]interface{})
		assert.Len(t, devices, 3)

		// Check that one device is marked as current
		var foundCurrent bool
		for _, d := range devices {
			device := d.(map[string]interface{})
			if device["id"] == currentDeviceID.String() {
				assert.True(t, device["is_current"].(bool))
				foundCurrent = true
			}
		}
		assert.True(t, foundCurrent, "Should have found current device")
	})

	t.Run("ok/empty_list", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID: uuid.New(), // Different user with no devices
				Email:  "user@example.com",
			}
			authCtx.Set(c)
			c.Next()
		})

		deviceStore := inmemory.NewDeviceStore()
		handler := listDevicesHandler(deviceStore)
		r.GET("/devices", handler)

		req := httptest.NewRequest("GET", "/devices", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)

		devices := resp["devices"].([]interface{})
		assert.Len(t, devices, 0)
	})

	t.Run("error/not_authenticated", func(t *testing.T) {
		r := gin.New()
		deviceStore := inmemory.NewDeviceStore()
		handler := listDevicesHandler(deviceStore)
		r.GET("/devices", handler)

		req := httptest.NewRequest("GET", "/devices", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetDeviceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	deviceID := uuid.New()
	otherUserID := uuid.New()

	t.Run("ok/get_device", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:   userID,
				Email:    "user@example.com",
				DeviceID: &deviceID,
			}
			authCtx.Set(c)
			c.Next()
		})

		// Create device store with test data
		deviceStore := inmemory.NewDeviceStore()
		ctx := context.Background()

		device := &domain.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceName: "Test Device",
			CreatedAt:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		}
		require.NoError(t, deviceStore.Create(ctx, device))

		handler := getDeviceHandler(deviceStore)
		r.GET("/devices/:id", handler)

		req := httptest.NewRequest("GET", "/devices/"+deviceID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, deviceID.String(), resp["id"])
		assert.Equal(t, "Test Device", resp["device_name"])
		assert.True(t, resp["is_current"].(bool))
	})

	t.Run("error/not_authenticated", func(t *testing.T) {
		r := gin.New()
		deviceStore := inmemory.NewDeviceStore()
		handler := getDeviceHandler(deviceStore)
		r.GET("/devices/:id", handler)

		req := httptest.NewRequest("GET", "/devices/"+deviceID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error/invalid_device_id", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID: userID,
				Email:  "user@example.com",
			}
			authCtx.Set(c)
			c.Next()
		})

		deviceStore := inmemory.NewDeviceStore()
		handler := getDeviceHandler(deviceStore)
		r.GET("/devices/:id", handler)

		req := httptest.NewRequest("GET", "/devices/invalid-uuid", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error/device_not_found", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID: userID,
				Email:  "user@example.com",
			}
			authCtx.Set(c)
			c.Next()
		})

		deviceStore := inmemory.NewDeviceStore()
		handler := getDeviceHandler(deviceStore)
		r.GET("/devices/:id", handler)

		req := httptest.NewRequest("GET", "/devices/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error/access_denied", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context as different user
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID: otherUserID,
				Email:  "other@example.com",
			}
			authCtx.Set(c)
			c.Next()
		})

		// Create device store with test data
		deviceStore := inmemory.NewDeviceStore()
		ctx := context.Background()

		device := &domain.Device{
			ID:         deviceID,
			UserID:     userID, // Different user owns this device
			DeviceName: "Test Device",
			CreatedAt:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		}
		require.NoError(t, deviceStore.Create(ctx, device))

		handler := getDeviceHandler(deviceStore)
		r.GET("/devices/:id", handler)

		req := httptest.NewRequest("GET", "/devices/"+deviceID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestRevokeDeviceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	deviceID := uuid.New()
	currentDeviceID := uuid.New()

	t.Run("ok/revoke_device", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context with device ID
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:   userID,
				Email:    "user@example.com",
				DeviceID: &currentDeviceID,
			}
			authCtx.Set(c)
			c.Next()
		})

		// Create stores with test data
		deviceStore := inmemory.NewDeviceStore()
		refreshStore := inmemory.NewRefreshStore()
		auditStore := inmemory.NewAuditStore()
		ctx := context.Background()

		// Add device to revoke
		device := &domain.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceName: "Device to Revoke",
			CreatedAt:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		}
		require.NoError(t, deviceStore.Create(ctx, device))

		// Add refresh token for the device
		tokenHash := []byte("test-token-hash")
		_, _, err := refreshStore.CreateFamilyWithDevice(ctx, userID, deviceID, 1, tokenHash, time.Now().Add(time.Hour))
		require.NoError(t, err)

		handler := revokeDeviceHandler(deviceStore, refreshStore, auditStore)
		r.DELETE("/devices/:id", handler)

		req := httptest.NewRequest("DELETE", "/devices/"+deviceID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify device was revoked
		revokedDevice, err := deviceStore.GetByID(ctx, deviceID)
		assert.NoError(t, err)
		assert.Nil(t, revokedDevice) // Should return nil for revoked devices

		// Verify audit event was recorded
		events := auditStore.GetEvents()
		require.Len(t, events, 1)
		assert.Equal(t, domain.EventDeviceRevoked, events[0].Type)
		assert.Equal(t, deviceID.String(), events[0].Metadata["device_id"])
	})

	t.Run("error/not_authenticated", func(t *testing.T) {
		r := gin.New()
		deviceStore := inmemory.NewDeviceStore()
		refreshStore := inmemory.NewRefreshStore()
		auditStore := inmemory.NewAuditStore()
		handler := revokeDeviceHandler(deviceStore, refreshStore, auditStore)
		r.DELETE("/devices/:id", handler)

		req := httptest.NewRequest("DELETE", "/devices/"+deviceID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error/invalid_device_id", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID: userID,
				Email:  "user@example.com",
			}
			authCtx.Set(c)
			c.Next()
		})

		deviceStore := inmemory.NewDeviceStore()
		refreshStore := inmemory.NewRefreshStore()
		auditStore := inmemory.NewAuditStore()
		handler := revokeDeviceHandler(deviceStore, refreshStore, auditStore)
		r.DELETE("/devices/:id", handler)

		req := httptest.NewRequest("DELETE", "/devices/invalid-uuid", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error/cannot_revoke_current_device", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context with same device ID
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:   userID,
				Email:    "user@example.com",
				DeviceID: &deviceID, // Same as device being revoked
			}
			authCtx.Set(c)
			c.Next()
		})

		// Create stores with test data
		deviceStore := inmemory.NewDeviceStore()
		refreshStore := inmemory.NewRefreshStore()
		auditStore := inmemory.NewAuditStore()
		ctx := context.Background()

		// Add current device
		device := &domain.Device{
			ID:         deviceID,
			UserID:     userID,
			DeviceName: "Current Device",
			CreatedAt:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		}
		require.NoError(t, deviceStore.Create(ctx, device))

		handler := revokeDeviceHandler(deviceStore, refreshStore, auditStore)
		r.DELETE("/devices/:id", handler)

		req := httptest.NewRequest("DELETE", "/devices/"+deviceID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp["message"], "cannot revoke")
	})

	t.Run("error/device_not_found", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:   userID,
				Email:    "user@example.com",
				DeviceID: &currentDeviceID,
			}
			authCtx.Set(c)
			c.Next()
		})

		deviceStore := inmemory.NewDeviceStore()
		refreshStore := inmemory.NewRefreshStore()
		auditStore := inmemory.NewAuditStore()
		handler := revokeDeviceHandler(deviceStore, refreshStore, auditStore)
		r.DELETE("/devices/:id", handler)

		req := httptest.NewRequest("DELETE", "/devices/"+uuid.New().String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error/access_denied", func(t *testing.T) {
		r := gin.New()

		otherUserID := uuid.New()

		// Add authenticated context as different user
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID:   otherUserID,
				Email:    "other@example.com",
				DeviceID: &currentDeviceID,
			}
			authCtx.Set(c)
			c.Next()
		})

		// Create stores with test data
		deviceStore := inmemory.NewDeviceStore()
		refreshStore := inmemory.NewRefreshStore()
		auditStore := inmemory.NewAuditStore()
		ctx := context.Background()

		// Add device owned by different user
		device := &domain.Device{
			ID:         deviceID,
			UserID:     userID, // Different user owns this
			DeviceName: "Other User Device",
			CreatedAt:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			LastUsedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		}
		require.NoError(t, deviceStore.Create(ctx, device))

		handler := revokeDeviceHandler(deviceStore, refreshStore, auditStore)
		r.DELETE("/devices/:id", handler)

		req := httptest.NewRequest("DELETE", "/devices/"+deviceID.String(), nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// mockDeviceStore implements a minimal store.DeviceStore for error testing
type mockDeviceStore struct {
	getByUserFunc func(ctx context.Context, userID uuid.UUID) ([]*domain.Device, error)
	getByIDFunc   func(ctx context.Context, id uuid.UUID) (*domain.Device, error)
	revokeFunc    func(ctx context.Context, id uuid.UUID, revokedAt time.Time) error
}

func (m *mockDeviceStore) Create(ctx context.Context, device *domain.Device) error {
	return nil
}

func (m *mockDeviceStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.Device, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockDeviceStore) GetByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Device, error) {
	if m.getByUserFunc != nil {
		return m.getByUserFunc(ctx, userID)
	}
	return []*domain.Device{}, nil
}

func (m *mockDeviceStore) UpdateLastUsed(ctx context.Context, id uuid.UUID, lastUsedAt time.Time) error {
	return nil
}

func (m *mockDeviceStore) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, id, revokedAt)
	}
	return nil
}

func (m *mockDeviceStore) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func TestListDevicesHandler_WithMockError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	t.Run("error/service_error", func(t *testing.T) {
		r := gin.New()

		// Add authenticated context
		r.Use(func(c *gin.Context) {
			authCtx := authkit.AuthContext{
				UserID: userID,
				Email:  "user@example.com",
			}
			authCtx.Set(c)
			c.Next()
		})

		deviceStore := &mockDeviceStore{
			getByUserFunc: func(ctx context.Context, uid uuid.UUID) ([]*domain.Device, error) {
				return nil, errors.New("database error")
			},
		}

		handler := listDevicesHandler(deviceStore)
		r.GET("/devices", handler)

		req := httptest.NewRequest("GET", "/devices", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
