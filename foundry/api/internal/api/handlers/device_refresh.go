package handlers

import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/sha256"
    "crypto/subtle"
    "encoding/base64"
    "encoding/hex"
    "errors"
    "fmt"
    "net/http"
    "time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	usersvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	libauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	authjwt "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	tokens "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
	"gorm.io/gorm"
	"log/slog"
)

// DeviceRefreshHandler handles device-bound token refresh with family-based rotation.
type DeviceRefreshHandler struct {
	deviceRepo          userrepo.DeviceRepository
	refreshTokenRepo    userrepo.RefreshTokenRepository
	userService         usersvc.UserService
	roleService         usersvc.RoleService
	userRoleService     usersvc.UserRoleService
	jwtManager          authjwt.JWTManager
	cookieManager       *auth.CookieManager
	deviceProofVerifier *auth.DeviceProofVerifier
	authConfig          *config.AuthConfig
	logger              *slog.Logger
}

// NewDeviceRefreshHandler creates a new device refresh handler.
func NewDeviceRefreshHandler(
	deviceRepo userrepo.DeviceRepository,
	refreshTokenRepo userrepo.RefreshTokenRepository,
	userService usersvc.UserService,
	roleService usersvc.RoleService,
	userRoleService usersvc.UserRoleService,
	jwtManager authjwt.JWTManager,
	authConfig *config.AuthConfig,
	logger *slog.Logger,
) *DeviceRefreshHandler {
	cookieManager := auth.NewCookieManager(authConfig)
	deviceProofVerifier := auth.NewDeviceProofVerifier(deviceRepo, authConfig)

	return &DeviceRefreshHandler{
		deviceRepo:          deviceRepo,
		refreshTokenRepo:    refreshTokenRepo,
		userService:         userService,
		roleService:         roleService,
		userRoleService:     userRoleService,
		jwtManager:          jwtManager,
		cookieManager:       cookieManager,
		deviceProofVerifier: deviceProofVerifier,
		authConfig:          authConfig,
		logger:              logger,
	}
}

// RefreshToken handles device-bound token refresh with rotation and replay detection
// @Summary Refresh access token
// @Description Refresh access token using device proof and cookie-bound refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Header 200 {string} X-Device-Id "Device ID for proof verification"
// @Header 200 {string} X-Device-Proof "Device proof signature"
// @Success 200 {object} map[string]interface{} "New access token"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid token or proof"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/refresh [post].
func (h *DeviceRefreshHandler) RefreshToken(c *gin.Context) {
	h.logger.Info("refresh: start")
	// Parse refresh token cookie
	refreshCookie, err := h.cookieManager.ParseRefreshTokenCookie(c)
	if err != nil {
		h.logger.Error("failed to parse refresh token cookie", "error", err, "ip", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid refresh token"})
		return
	}
	h.logger.Info("refresh: cookie parsed", "jti", refreshCookie.JTI)

	// Validate device proof headers
	deviceIDHeader := c.GetHeader("X-Device-Id")
	proofHeader := c.GetHeader("X-Device-Proof")
	deviceHeaders, err := h.deviceProofVerifier.ParseDeviceProofHeaders(deviceIDHeader, proofHeader)
	if err != nil {
		h.logger.Error("failed to parse device proof headers", "error", err, "ip", c.ClientIP())
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or invalid device proof headers"})
		return
	}

	// Get refresh token from database
	refreshToken, err := h.refreshTokenRepo.GetByID(refreshCookie.JTI)
	if err != nil || refreshToken == nil {
		h.logger.Error("refresh token not found", "jti", refreshCookie.JTI, "ip", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
		return
	}
	h.logger.Info("refresh: token loaded", "user_id", refreshToken.UserID)

	// Check if token is expired or revoked
	if refreshToken.RevokedAt != nil {
		h.logger.Warn("attempt to use revoked refresh token", "jti", refreshCookie.JTI, "user_id", refreshToken.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
		return
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		h.logger.Warn("attempt to use expired refresh token", "jti", refreshCookie.JTI, "user_id", refreshToken.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token has expired"})
		return
	}

	// Validate device ID matches token
	if refreshToken.DeviceID != deviceHeaders.DeviceID {
		h.logger.Warn("device ID mismatch in refresh request",
			"token_device_id", refreshToken.DeviceID,
			"header_device_id", deviceHeaders.DeviceID,
			"user_id", refreshToken.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "device mismatch"})
		return
	}

	// Validate secret hash using constant-time comparison
	expectedHash, err := h.generateSecretHash(refreshCookie.Secret)
	if err != nil {
		h.logger.Error("failed to generate secret hash", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	if subtle.ConstantTimeCompare([]byte(refreshToken.SecretHash), []byte(expectedHash)) != 1 {
		h.logger.Warn("secret hash mismatch in refresh request",
			"jti", refreshCookie.JTI,
			"user_id", refreshToken.UserID,
			"ip", c.ClientIP())
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token secret"})
		return
	}
	h.logger.Info("refresh: secret validated")

	// Check for replay attack - if token has already been replaced/used
	if err := h.checkReplayAttack(refreshToken); err != nil {
		h.logger.Error("replay attack detected", "error", err, "jti", refreshCookie.JTI, "family_id", refreshToken.FamilyID)
		// Revoke entire token family on replay detection
		if err := h.revokeFamilyTokens(refreshToken.FamilyID, "replay_attack"); err != nil {
			h.logger.Error("failed to revoke family tokens after replay", "error", err, "family_id", refreshToken.FamilyID)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token reuse detected - session invalidated"})
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
		"/auth/refresh",
	)
	if err != nil {
		h.logger.Error("device proof verification failed",
			"error", err,
			"device_id", deviceHeaders.DeviceID,
			"user_id", refreshToken.UserID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid device proof"})
		return
	}
	h.logger.Info("refresh: proof ok")

	// Update device last used timestamp
	if err := h.deviceRepo.UpdateLastUsed(refreshToken.DeviceID, time.Now()); err != nil {
		h.logger.Error("failed to update device last used", "error", err, "device_id", refreshToken.DeviceID)
		// Don't fail the request for this
	}

	// Create new refresh token (rotation) - MUST succeed in production
	newRefreshToken, newSecret, err := h.rotateRefreshTokenAtomic(refreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		// In dev mode only, allow proceeding without rotation
		if h.authConfig.DevMode {
			h.logger.Warn("DEVMODE: failed to rotate refresh token; proceeding without rotation",
				"error", err,
				"jti", refreshCookie.JTI,
				"WARNING", "This behavior is ONLY for development and is a security risk in production")
		} else {
			// In production, fail the request if rotation fails
			h.logger.Error("failed to rotate refresh token", "error", err, "jti", refreshCookie.JTI)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "token rotation failed"})
			return
		}
	} else {
		h.logger.Info("refresh: rotated", "new_jti", newRefreshToken.ID)
		// Set new refresh token cookie
		h.cookieManager.SetRefreshTokenCookie(c, newRefreshToken.ID, newSecret, h.authConfig.RefreshTTL)
	}

	// Generate new access token
	user, err := h.userService.GetUserByID(refreshToken.UserID)
	if err != nil {
		h.logger.Error("failed to get user for token generation", "error", err, "user_id", refreshToken.UserID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}
	h.logger.Info("refresh: user loaded")

	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		h.logger.Error("failed to generate access token", "error", err, "user_id", refreshToken.UserID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}
	h.logger.Info("refresh: access minted")

	if newRefreshToken != nil {
		h.logger.Info("token refresh successful with rotation",
			"user_id", refreshToken.UserID,
			"device_id", refreshToken.DeviceID,
			"old_jti", refreshCookie.JTI,
			"new_jti", newRefreshToken.ID,
			"family_id", refreshToken.FamilyID)
	} else {
		h.logger.Warn("token refresh successful without rotation (dev mode)",
			"user_id", refreshToken.UserID,
			"device_id", refreshToken.DeviceID,
			"jti", refreshCookie.JTI,
			"family_id", refreshToken.FamilyID)
	}

	// Return new access token
	c.JSON(http.StatusOK, gin.H{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(h.authConfig.AccessTTL.Seconds()),
	})
}

// Helper methods

func (h *DeviceRefreshHandler) generateSecretHash(secret string) (string, error) {
	if h.authConfig.RefreshHashSecret == "" {
		// Fallback to simple hash for development
		hash := sha256.Sum256([]byte(secret))
		return hex.EncodeToString(hash[:]), nil
	}

	// Use HMAC with configured secret
	mac := hmac.New(sha256.New, []byte(h.authConfig.RefreshHashSecret))
	mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (h *DeviceRefreshHandler) checkReplayAttack(token *dbmodel.RefreshToken) error {
	// Check if token has already been rotated
	if token.RotatedAt != nil {
		return fmt.Errorf("token has already been used - replay attack detected")
	}

	// Also check if there are any newer tokens in the same family (belt and suspenders)
	// This catches cases where rotation succeeded but marking failed
	newer, err := h.refreshTokenRepo.GetNewerTokensInFamily(token.FamilyID, token.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to check for newer tokens: %w", err)
	}

	if len(newer) > 0 {
		return fmt.Errorf("token has already been replaced - replay attack detected")
	}

	return nil
}

func (h *DeviceRefreshHandler) revokeFamilyTokens(familyID uuid.UUID, reason string) error {
	// Get all tokens in family
	tokens, err := h.refreshTokenRepo.GetTokensByFamily(familyID)
	if err != nil {
		return fmt.Errorf("failed to get family tokens: %w", err)
	}

	// Revoke all non-revoked tokens in family
	now := time.Now()
	for _, token := range tokens {
		if token.RevokedAt == nil {
			token.RevokedAt = &now
			if err := h.refreshTokenRepo.Update(&token); err != nil {
				h.logger.Error("failed to revoke token in family",
					"error", err,
					"jti", token.ID,
					"family_id", familyID,
					"reason", reason)
			}
		}
	}

	return nil
}

func (h *DeviceRefreshHandler) rotateRefreshTokenAtomic(oldToken *dbmodel.RefreshToken, ip, userAgent string) (*dbmodel.RefreshToken, string, error) {
	// Generate new secret for the rotated token
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate secret: %w", err)
	}
	newSecret := base64.RawURLEncoding.EncodeToString(secretBytes)

	// Generate secret hash
	secretHash, err := h.generateSecretHash(newSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate secret hash: %w", err)
	}

	// Create new refresh token with same family but new parent
	newToken := &dbmodel.RefreshToken{
		ID:         uuid.New(),
		UserID:     oldToken.UserID,
		DeviceID:   oldToken.DeviceID,
		FamilyID:   oldToken.FamilyID, // Same family for tracking
		ParentID:   &oldToken.ID,      // Track replacement chain
		SecretHash: secretHash,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(h.authConfig.RefreshTTL),
		RevokedAt:  nil,
		IP:         &ip,
		UserAgent:  &userAgent,
	}

	// Atomically mark old token as rotated and create new token
	// This prevents race conditions where two parallel requests could both succeed
	if err := h.refreshTokenRepo.RotateTokenAtomic(oldToken, newToken); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Token was already rotated by another request
			return nil, "", fmt.Errorf("token already rotated - possible race condition")
		}
		return nil, "", fmt.Errorf("failed to rotate token atomically: %w", err)
	}

	return newToken, newSecret, nil
}

func (h *DeviceRefreshHandler) generateAccessToken(user *dbmodel.User) (string, error) {
	// Aggregate permissions from user's roles
	permsMap := map[libauth.Permission]struct{}{}
	userRoles, err := h.userRoleService.GetUserRoles(user.ID)
	if err == nil {
		for _, ur := range userRoles {
			role, err := h.roleService.GetRoleByID(ur.RoleID)
			if err == nil && role != nil {
				for _, p := range role.GetPermissions() {
					permsMap[p] = struct{}{}
				}
			}
		}
	}
	perms := make([]libauth.Permission, 0, len(permsMap))
	for p := range permsMap {
		perms = append(perms, p)
	}

	// Generate an auth token with permissions
	return tokens.GenerateAuthToken(h.jwtManager, fmt.Sprintf("%d", user.ID), perms, h.authConfig.AccessTTL)
}
