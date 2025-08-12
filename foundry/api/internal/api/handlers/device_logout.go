package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	"log/slog"
)

// DeviceLogoutHandler handles device-bound logout with token revocation.
type DeviceLogoutHandler struct {
	deviceRepo          userrepo.DeviceRepository
	refreshTokenRepo    userrepo.RefreshTokenRepository
	cookieManager       *auth.CookieManager
	deviceProofVerifier *auth.DeviceProofVerifier
	authConfig          *config.AuthConfig
	logger              *slog.Logger
}

// NewDeviceLogoutHandler creates a new device logout handler.
func NewDeviceLogoutHandler(
	deviceRepo userrepo.DeviceRepository,
	refreshTokenRepo userrepo.RefreshTokenRepository,
	authConfig *config.AuthConfig,
	logger *slog.Logger,
) *DeviceLogoutHandler {
	cookieManager := auth.NewCookieManager(authConfig)
	deviceProofVerifier := auth.NewDeviceProofVerifier(deviceRepo, authConfig)

	return &DeviceLogoutHandler{
		deviceRepo:          deviceRepo,
		refreshTokenRepo:    refreshTokenRepo,
		cookieManager:       cookieManager,
		deviceProofVerifier: deviceProofVerifier,
		authConfig:          authConfig,
		logger:              logger,
	}
}

// Logout handles device logout with refresh token revocation
// @Summary Logout and revoke refresh token
// @Description Logout user and revoke the current refresh token using device proof
// @Tags auth
// @Accept json
// @Produce json
// @Header 200 {string} X-Device-Id "Device ID for proof verification"
// @Header 200 {string} X-Device-Proof "Device proof signature"
// @Success 204 "No Content - logout successful"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid token or proof"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/logout [post].
func (h *DeviceLogoutHandler) Logout(c *gin.Context) {
	if os.Getenv("TEST_LOG") == "1" {
		h.logger.Info("[TEST] Logout: begin")
	}
	// Parse refresh token cookie - handle missing cookies gracefully
	refreshCookie, err := h.cookieManager.ParseRefreshTokenCookie(c)
	if err != nil {
		h.logger.Info("logout request without refresh token cookie", "error", err, "ip", c.ClientIP())
		// Clear any existing cookie and return success - idempotent logout
		h.cookieManager.ClearRefreshTokenCookie(c)
		c.Status(http.StatusNoContent)
		return
	}

	// Validate device proof headers
	deviceIDHeader := c.GetHeader("X-Device-Id")
	proofHeader := c.GetHeader("X-Device-Proof")
	deviceHeaders, err := h.deviceProofVerifier.ParseDeviceProofHeaders(deviceIDHeader, proofHeader)
	if err != nil {
		h.logger.Error("failed to parse device proof headers in logout", "error", err, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid device proof headers"})
		return
	}

	// Get refresh token from database
	refreshToken, err := h.refreshTokenRepo.GetByID(refreshCookie.JTI)
	if err != nil || refreshToken == nil {
		h.logger.Info("logout with non-existent refresh token", "jti", refreshCookie.JTI, "ip", c.ClientIP())
		// Clear cookie and return success - token doesn't exist anyway
		h.cookieManager.ClearRefreshTokenCookie(c)
		c.Status(http.StatusNoContent)
		return
	}

	// Check if token is already revoked
	if refreshToken.RevokedAt != nil {
		h.logger.Info("logout with already revoked token", "jti", refreshCookie.JTI, "user_id", refreshToken.UserID)
		// Clear cookie and return success - idempotent logout
		h.cookieManager.ClearRefreshTokenCookie(c)
		c.Status(http.StatusNoContent)
		return
	}

	// Validate device ID matches token
	if refreshToken.DeviceID != deviceHeaders.DeviceID {
		h.logger.Warn("device ID mismatch in logout request",
			"token_device_id", refreshToken.DeviceID,
			"header_device_id", deviceHeaders.DeviceID,
			"user_id", refreshToken.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "device mismatch"})
		return
	}

	// Verify device proof against device stored key
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = c.Request.Host
	}

	_, err = h.deviceProofVerifier.VerifyDeviceProof(
		deviceIDHeader,
		proofHeader,
		origin,
		"POST",
		"/auth/logout",
	)
	if err != nil {
		h.logger.Error("device proof verification failed in logout",
			"error", err,
			"device_id", deviceHeaders.DeviceID,
			"user_id", refreshToken.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid device proof"})
		return
	}
	if os.Getenv("TEST_LOG") == "1" {
		h.logger.Info("[TEST] Logout: proof ok", "device_id", deviceHeaders.DeviceID)
	}

	// Revoke the current refresh token
	if err := h.refreshTokenRepo.RevokeToken(refreshToken.ID); err != nil {
		h.logger.Error("failed to revoke refresh token during logout",
			"error", err,
			"jti", refreshToken.ID,
			"user_id", refreshToken.UserID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete logout"})
		return
	}

	// Clear refresh token cookie
	h.cookieManager.ClearRefreshTokenCookie(c)

	h.logger.Info("logout successful",
		"user_id", refreshToken.UserID,
		"device_id", refreshToken.DeviceID,
		"jti", refreshToken.ID)

	// Return 204 No Content for successful logout
	c.Status(http.StatusNoContent)
}
