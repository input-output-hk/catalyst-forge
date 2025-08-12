package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	usersvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	authjwt "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"log/slog"
)

// DeviceManagementHandler handles device management endpoints for user account settings.
type DeviceManagementHandler struct {
	deviceRepo       userrepo.DeviceRepository
	refreshTokenRepo userrepo.RefreshTokenRepository
	userService      usersvc.UserService
	jwtManager       authjwt.JWTManager
	authConfig       *config.AuthConfig
	logger           *slog.Logger
}

// DeviceListResponse represents a device in the list response.
type DeviceListResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	LastUsedAt *string   `json:"last_used_at"`
	Status     string    `json:"status"`
	CreatedAt  string    `json:"created_at"`
}

// NewDeviceManagementHandler creates a new device management handler.
func NewDeviceManagementHandler(
	deviceRepo userrepo.DeviceRepository,
	refreshTokenRepo userrepo.RefreshTokenRepository,
	userService usersvc.UserService,
	jwtManager authjwt.JWTManager,
	authConfig *config.AuthConfig,
	logger *slog.Logger,
) *DeviceManagementHandler {
	return &DeviceManagementHandler{
		deviceRepo:       deviceRepo,
		refreshTokenRepo: refreshTokenRepo,
		userService:      userService,
		jwtManager:       jwtManager,
		authConfig:       authConfig,
		logger:           logger,
	}
}

// ListDevices handles GET /auth/devices - List user's registered devices
// @Summary List user's devices
// @Description Get list of all devices registered to the authenticated user
// @Tags auth,devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} DeviceListResponse "List of user's devices"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/devices [get].
func (h *DeviceManagementHandler) ListDevices(c *gin.Context) {
	// Extract user ID from JWT context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Convert user ID to uint
	userIDFloat, ok := userIDStr.(float64)
	if !ok {
		// Try string conversion as fallback
		if userIDStrVal, ok := userIDStr.(string); ok {
			userIDInt, err := strconv.ParseUint(userIDStrVal, 10, 32)
			if err != nil {
				h.logger.Error("invalid user_id format in context", "user_id", userIDStr)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			userIDFloat = float64(userIDInt)
		} else {
			h.logger.Error("user_id type assertion failed", "user_id", userIDStr, "type", typeof(userIDStr))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
	}

	userID := uint(userIDFloat)

	// Get user's devices from repository
	devices, err := h.deviceRepo.GetByUserID(userID)
	if err != nil {
		h.logger.Error("failed to get user devices", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve devices"})
		return
	}

	// Convert to response format
	response := make([]DeviceListResponse, len(devices))
	for i, device := range devices {
		var lastUsedAt *string
		if device.LastUsedAt != nil {
			lastUsedAtStr := device.LastUsedAt.Format("2006-01-02T15:04:05Z")
			lastUsedAt = &lastUsedAtStr
		}

		response[i] = DeviceListResponse{
			ID:         device.ID,
			Name:       device.Name,
			LastUsedAt: lastUsedAt,
			Status:     string(device.Status),
			CreatedAt:  device.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	h.logger.Info("devices listed successfully", "user_id", userID, "device_count", len(devices))
	c.JSON(http.StatusOK, response)
}

// DeleteDevice handles DELETE /auth/devices/:id - Revoke specific device and its token family
// @Summary Delete/revoke a device
// @Description Revoke a specific device and invalidate all its refresh tokens
// @Tags auth,devices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Device ID (UUID)"
// @Success 204 "No Content - device revoked successfully"
// @Failure 400 {object} map[string]interface{} "Invalid device ID"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Device not found"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/devices/{id} [delete].
func (h *DeviceManagementHandler) DeleteDevice(c *gin.Context) {
	// Extract user ID from JWT context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// Convert user ID to uint
	userIDFloat, ok := userIDStr.(float64)
	if !ok {
		// Try string conversion as fallback
		if userIDStrVal, ok := userIDStr.(string); ok {
			userIDInt, err := strconv.ParseUint(userIDStrVal, 10, 32)
			if err != nil {
				h.logger.Error("invalid user_id format in context", "user_id", userIDStr)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			userIDFloat = float64(userIDInt)
		} else {
			h.logger.Error("user_id type assertion failed", "user_id", userIDStr, "type", typeof(userIDStr))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
	}

	userID := uint(userIDFloat)

	// Parse device ID from URL parameter
	deviceIDStr := c.Param("id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		h.logger.Error("invalid device ID format", "device_id", deviceIDStr, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID format"})
		return
	}

	// Get the device to verify ownership and existence
	device, err := h.deviceRepo.GetByID(deviceID)
	if err != nil {
		h.logger.Error("failed to get device", "error", err, "device_id", deviceID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve device"})
		return
	}

	if device == nil {
		h.logger.Warn("device not found", "device_id", deviceID, "user_id", userID)
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}

	// Verify device belongs to the authenticated user
	if device.UserID != userID {
		h.logger.Warn("attempt to delete device belonging to different user",
			"device_id", deviceID,
			"device_user_id", device.UserID,
			"requesting_user_id", userID)
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"}) // Don't reveal existence
		return
	}

	// Check if device is already revoked
	if device.Status == dbmodel.DeviceStatusRevoked {
		h.logger.Info("device already revoked", "device_id", deviceID, "user_id", userID)
		c.Status(http.StatusNoContent) // Idempotent - already revoked
		return
	}

	// Revoke all refresh token families for this device
	activeTokens, err := h.refreshTokenRepo.GetActiveTokensByDevice(deviceID)
	if err != nil {
		h.logger.Error("failed to get device tokens for revocation", "error", err, "device_id", deviceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke device"})
		return
	}

	// Revoke each token family (multiple families possible if device was used multiple times)
	familiesRevoked := make(map[uuid.UUID]bool)
	for _, token := range activeTokens {
		if !familiesRevoked[token.FamilyID] {
			if err := h.refreshTokenRepo.RevokeTokenFamily(token.FamilyID); err != nil {
				h.logger.Error("failed to revoke token family",
					"error", err,
					"family_id", token.FamilyID,
					"device_id", deviceID)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke device tokens"})
				return
			}
			familiesRevoked[token.FamilyID] = true
			h.logger.Info("token family revoked", "family_id", token.FamilyID, "device_id", deviceID)
		}
	}

	// Mark device as revoked using repository method
	if err := h.deviceRepo.RevokeDevice(deviceID); err != nil {
		h.logger.Error("failed to revoke device", "error", err, "device_id", deviceID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke device"})
		return
	}

	h.logger.Info("device revoked successfully",
		"device_id", deviceID,
		"user_id", userID,
		"token_families_revoked", len(familiesRevoked))

	// Return 204 No Content for successful deletion
	c.Status(http.StatusNoContent)
}

// Helper function for debugging type issues.
func typeof(v interface{}) string {
	switch v.(type) {
	case int:
		return "int"
	case float64:
		return "float64"
	case string:
		return "string"
	default:
		return "unknown"
	}
}
