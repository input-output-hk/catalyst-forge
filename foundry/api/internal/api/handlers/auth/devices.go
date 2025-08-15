package auth

import (
	"net/http"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit/store"
)

// DeviceInfo represents a single device in the response
type DeviceInfo struct {
	ID         string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DeviceName string    `json:"device_name" example:"My CLI"`
	CreatedAt  time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	LastUsedAt time.Time `json:"last_used_at" example:"2024-01-02T15:04:05Z"`
	IsCurrent  bool      `json:"is_current" example:"false"`
}

// DeviceListResponse represents the response for listing devices
type DeviceListResponse struct {
	Devices []DeviceInfo `json:"devices"`
}

// DeviceResponse represents the response for a single device
type DeviceResponse struct {
	ID         string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	DeviceName string    `json:"device_name" example:"My CLI"`
	CreatedAt  time.Time `json:"created_at" example:"2024-01-01T00:00:00Z"`
	LastUsedAt time.Time `json:"last_used_at" example:"2024-01-02T15:04:05Z"`
	IsCurrent  bool      `json:"is_current" example:"false"`
}

// DeviceManagementDeps contains dependencies for device management handlers.
type DeviceManagementDeps struct {
	DeviceStore  store.DeviceStore
	RefreshStore store.RefreshStore
	AuditStore   store.AuditStore
}

// RegisterDeviceManagement registers device management endpoints.
func RegisterDeviceManagement(r *gin.Engine, deps DeviceManagementDeps) {
	api := r.Group("/api/v1/auth/devices")
	
	// @Summary List user devices
	// @Description Lists all devices registered for the authenticated user
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Security BearerAuth
	// @Success 200 {object} DeviceListResponse "List of user's devices"
	// @Failure 401 {object} httpkit.ErrorResponse "Authentication required"
	// @Failure 500 {object} httpkit.ErrorResponse "Internal server error"
	// @Router /api/v1/auth/devices [get]
	api.GET("", listDevicesHandler(deps.DeviceStore))
	
	// @Summary Get device details
	// @Description Gets details of a specific device
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Security BearerAuth
	// @Param id path string true "Device ID" format(uuid)
	// @Success 200 {object} DeviceResponse "Device details"
	// @Failure 400 {object} httpkit.ErrorResponse "Invalid device ID"
	// @Failure 401 {object} httpkit.ErrorResponse "Authentication required"
	// @Failure 403 {object} httpkit.ErrorResponse "Access denied"
	// @Failure 404 {object} httpkit.ErrorResponse "Device not found"
	// @Failure 500 {object} httpkit.ErrorResponse "Internal server error"
	// @Router /api/v1/auth/devices/{id} [get]
	api.GET("/:id", getDeviceHandler(deps.DeviceStore))
	
	// @Summary Revoke device
	// @Description Revokes a device and all associated refresh tokens
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Security BearerAuth
	// @Param id path string true "Device ID" format(uuid)
	// @Success 204 "Device revoked successfully"
	// @Failure 400 {object} httpkit.ErrorResponse "Invalid device ID or cannot revoke current device"
	// @Failure 401 {object} httpkit.ErrorResponse "Authentication required"
	// @Failure 403 {object} httpkit.ErrorResponse "Access denied"
	// @Failure 404 {object} httpkit.ErrorResponse "Device not found"
	// @Failure 500 {object} httpkit.ErrorResponse "Internal server error"
	// @Router /api/v1/auth/devices/{id} [delete]
	api.DELETE("/:id", revokeDeviceHandler(deps.DeviceStore, deps.RefreshStore, deps.AuditStore))
}

// listDevicesHandler handles GET /api/v1/auth/devices
func listDevicesHandler(deviceStore store.DeviceStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check authentication
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = httpkit.NewUnauthorizedError("authentication required").Write(c.Writer)
			return
		}
		
		// Get user's devices
		devices, err := deviceStore.GetByUser(c.Request.Context(), ctx.UserID)
		if err != nil {
			_ = httpkit.NewInternalError().Write(c.Writer)
			return
		}
		
		// Format response
		deviceList := make([]map[string]interface{}, 0, len(devices))
		for _, device := range devices {
			deviceList = append(deviceList, map[string]interface{}{
				"id":           device.ID.String(),
				"device_name":  device.DeviceName,
				"created_at":   device.CreatedAt,
				"last_used_at": device.LastUsedAt,
				"is_current":   isCurrentDevice(c, device.ID), // Check if this is the requesting device
			})
		}
		
		_ = httpkit.WriteJSON(c.Writer, http.StatusOK, map[string]interface{}{
			"devices": deviceList,
		})
	}
}

// getDeviceHandler handles GET /api/v1/auth/devices/:id
func getDeviceHandler(deviceStore store.DeviceStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check authentication
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = httpkit.NewUnauthorizedError("authentication required").Write(c.Writer)
			return
		}
		
		// Parse device ID
		deviceIDStr := c.Param("id")
		deviceID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			_ = httpkit.NewBadRequestError("invalid device ID").Write(c.Writer)
			return
		}
		
		// Get the device
		device, err := deviceStore.GetByID(c.Request.Context(), deviceID)
		if err != nil || device == nil {
			_ = httpkit.NewNotFoundError("device not found").Write(c.Writer)
			return
		}
		
		// Check ownership
		if device.UserID != ctx.UserID {
			_ = httpkit.NewForbiddenError("access denied").Write(c.Writer)
			return
		}
		
		_ = httpkit.WriteJSON(c.Writer, http.StatusOK, map[string]interface{}{
			"id":           device.ID.String(),
			"device_name":  device.DeviceName,
			"created_at":   device.CreatedAt,
			"last_used_at": device.LastUsedAt,
			"is_current":   isCurrentDevice(c, device.ID),
		})
	}
}

// revokeDeviceHandler handles DELETE /api/v1/auth/devices/:id
func revokeDeviceHandler(deviceStore store.DeviceStore, refreshStore store.RefreshStore, auditStore store.AuditStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check authentication
		ctx, ok := authkit.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = httpkit.NewUnauthorizedError("authentication required").Write(c.Writer)
			return
		}
		
		// Parse device ID
		deviceIDStr := c.Param("id")
		deviceID, err := uuid.Parse(deviceIDStr)
		if err != nil {
			_ = httpkit.NewBadRequestError("invalid device ID").Write(c.Writer)
			return
		}
		
		// Get the device to verify ownership
		device, err := deviceStore.GetByID(c.Request.Context(), deviceID)
		if err != nil || device == nil {
			_ = httpkit.NewNotFoundError("device not found").Write(c.Writer)
			return
		}
		
		// Check ownership
		if device.UserID != ctx.UserID {
			_ = httpkit.NewForbiddenError("access denied").Write(c.Writer)
			return
		}
		
		// Don't allow revoking the current device
		if isCurrentDevice(c, deviceID) {
			_ = httpkit.NewBadRequestError("cannot revoke current device").Write(c.Writer)
			return
		}
		
		now := time.Now().UTC()
		
		// Revoke all refresh tokens for this device
		if err := refreshStore.RevokeDeviceTokens(c.Request.Context(), deviceID, "device revoked", now); err != nil {
			_ = httpkit.NewInternalError().Write(c.Writer)
			return
		}
		
		// Mark device as revoked
		if err := deviceStore.Revoke(c.Request.Context(), deviceID, now); err != nil {
			_ = httpkit.NewInternalError().Write(c.Writer)
			return
		}
		
		// Audit the revocation
		if auditStore != nil {
			event := domain.Event{
				ID:        uuid.New(),
				Type:      domain.EventDeviceRevoked,
				UserID:    &ctx.UserID,
				CreatedAt: now,
				Metadata: map[string]interface{}{
					"device_id":   deviceID.String(),
					"device_name": device.DeviceName,
					"revoked_by":  "user",
				},
			}
			_ = auditStore.Record(c.Request.Context(), event)
		}
		
		c.Status(http.StatusNoContent)
	}
}

// isCurrentDevice checks if the given device ID matches the current request's device.
func isCurrentDevice(c *gin.Context, deviceID uuid.UUID) bool {
	// Extract device ID from the authentication context
	ctx, ok := authkit.From(c)
	if !ok {
		return false
	}
	
	// Check if the device ID matches
	if ctx.DeviceID != nil && *ctx.DeviceID == deviceID {
		return true
	}
	
	return false
}