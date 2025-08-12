package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"log/slog"
	"sync"

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
)

// DeviceLoginHandler implements returning-device sign-in via signed challenge.
type DeviceLoginHandler struct {
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

	// In-memory challenge store; replace with Redis/DB in production.
	challengeStorage map[uuid.UUID]*DeviceLoginChallenge
	challengeMu      *sync.RWMutex
}

// DeviceLoginChallenge represents a server-issued login challenge.
type DeviceLoginChallenge struct {
	DeviceID  uuid.UUID `json:"device_id"`
	Challenge string    `json:"challenge"`
	Algorithm string    `json:"alg"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    uint      `json:"user_id"`
}

// DeviceLoginInitRequest is the request for initiating login.
type DeviceLoginInitRequest struct {
	DeviceID uuid.UUID `json:"device_id" binding:"required"`
}

// DeviceLoginInitResponse is the login-init response payload.
type DeviceLoginInitResponse struct {
	Challenge string    `json:"challenge"`
	Algorithm string    `json:"alg"`
	ExpiresAt time.Time `json:"expires_at"`
	DeviceID  uuid.UUID `json:"device_id"`
}

// DeviceLoginRequest is the request to complete login using signed challenge.
type DeviceLoginRequest struct {
	DeviceID    uuid.UUID `json:"device_id" binding:"required"`
	Timestamp   int64     `json:"timestamp" binding:"required"`
	DeviceProof string    `json:"device_proof" binding:"required"` // base64url(signature)
}

// DeviceLoginResponse represents successful login response.
type DeviceLoginResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int         `json:"expires_in"`
	User        interface{} `json:"user"`
}

// NewDeviceLoginHandler creates a new login handler.
func NewDeviceLoginHandler(
	deviceRepo userrepo.DeviceRepository,
	refreshTokenRepo userrepo.RefreshTokenRepository,
	userService usersvc.UserService,
	roleService usersvc.RoleService,
	userRoleService usersvc.UserRoleService,
	jwtManager authjwt.JWTManager,
	authConfig *config.AuthConfig,
	logger *slog.Logger,
) *DeviceLoginHandler {
	return &DeviceLoginHandler{
		deviceRepo:          deviceRepo,
		refreshTokenRepo:    refreshTokenRepo,
		userService:         userService,
		roleService:         roleService,
		userRoleService:     userRoleService,
		jwtManager:          jwtManager,
		cookieManager:       auth.NewCookieManager(authConfig),
		deviceProofVerifier: auth.NewDeviceProofVerifier(deviceRepo, authConfig),
		authConfig:          authConfig,
		logger:              logger,
		challengeStorage:    make(map[uuid.UUID]*DeviceLoginChallenge),
		challengeMu:         &sync.RWMutex{},
	}
}

// InitLogin initializes a login challenge for an existing, active device
// @Summary Initialize returning-device login
// @Description Start sign-in for a device that still holds its device key; issues short-lived challenge
// @Tags auth
// @Accept json
// @Produce json
// @Param request body DeviceLoginInitRequest true "Login init request"
// @Success 200 {object} DeviceLoginInitResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Device not found or inactive"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/devices/login/init [post].
func (h *DeviceLoginHandler) InitLogin(c *gin.Context) {
	var req DeviceLoginInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	// Ensure device exists and is active
	device, err := h.deviceRepo.GetByID(req.DeviceID)
	if err != nil || device == nil {
		h.logger.Warn("login init: device not found or inactive", "device_id", req.DeviceID, "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "device not found or inactive"})
		return
	}

	// Create a random challenge valid for 5 minutes
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		h.logger.Error("login init: challenge generation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes)

	expiresAt := time.Now().Add(5 * time.Minute)
	h.challengeMu.Lock()
	h.challengeStorage[req.DeviceID] = &DeviceLoginChallenge{
		DeviceID:  req.DeviceID,
		Challenge: challenge,
		Algorithm: "ES256",
		ExpiresAt: expiresAt,
		UserID:    device.UserID,
	}
	h.challengeMu.Unlock()

	c.JSON(http.StatusOK, DeviceLoginInitResponse{
		Challenge: challenge,
		Algorithm: "ES256",
		ExpiresAt: expiresAt,
		DeviceID:  req.DeviceID,
	})
}

// Login completes login using signed challenge with stored device public key
// @Summary Complete returning-device login
// @Description Verify signed challenge with stored device key, mint new refresh family and access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body DeviceLoginRequest true "Login request"
// @Success 200 {object} DeviceLoginResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid or expired challenge or proof"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/devices/login [post].
func (h *DeviceLoginHandler) Login(c *gin.Context) {
	var req DeviceLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	// Lookup challenge by device_id
	ch := h.getChallenge(req.DeviceID)
	if ch == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired challenge"})
		return
	}
	if time.Now().After(ch.ExpiresAt) {
		h.deleteChallenge(req.DeviceID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "challenge expired"})
		return
	}

	// Verify timestamp skew
	if err := h.deviceProofVerifier.VerifyTimestamp(req.Timestamp); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("invalid timestamp: %v", err)})
		return
	}

	// Get stored device public key and device record (ensures active)
	pubKey, device, err := h.deviceProofVerifier.GetDevicePublicKey(req.DeviceID)
	if err != nil {
		h.logger.Error("login: failed to load device key", "error", err, "device_id", req.DeviceID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "device not found or inactive"})
		return
	}

	// Build canonical string using the same routine as registration
	canonical := h.deviceProofVerifier.GenerateCanonicalStringForChallenge(req.Timestamp, req.DeviceID, ch.Challenge)

	// Decode base64url signature
	sig, err := base64.RawURLEncoding.DecodeString(req.DeviceProof)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device proof encoding"})
		return
	}

	// Verify signature using stored key
	if err := h.deviceProofVerifier.VerifySignature(pubKey, canonical, sig); err != nil {
		h.logger.Warn("login: signature verification failed", "error", err, "device_id", req.DeviceID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid device proof"})
		return
	}

	// Touch last_used_at (best-effort)
	if err := h.deviceRepo.UpdateLastUsed(device.ID, time.Now()); err != nil {
		h.logger.Warn("login: failed to update last_used_at", "error", err, "device_id", device.ID)
	}

	// Create new refresh token family for this device
	familyID := uuid.New()
	refreshToken, secret, err := h.createInitialRefreshToken(device.ID, device.UserID, familyID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		h.logger.Error("login: failed to create refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Set refresh cookie
	h.cookieManager.SetRefreshTokenCookie(c, refreshToken.ID, secret, h.authConfig.RefreshTTL)

	// Mint access token
	user, err := h.userService.GetUserByID(device.UserID)
	if err != nil {
		h.logger.Error("login: failed to load user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}
	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		h.logger.Error("login: failed to mint access token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Clear challenge to prevent reuse
	h.deleteChallenge(req.DeviceID)

	c.JSON(http.StatusOK, DeviceLoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(h.authConfig.AccessTTL.Seconds()),
		User:        h.formatUserResponse(user),
	})
}

// Helpers

func (h *DeviceLoginHandler) getChallenge(deviceID uuid.UUID) *DeviceLoginChallenge {
	h.challengeMu.RLock()
	defer h.challengeMu.RUnlock()
	return h.challengeStorage[deviceID]
}

func (h *DeviceLoginHandler) deleteChallenge(deviceID uuid.UUID) {
	h.challengeMu.Lock()
	delete(h.challengeStorage, deviceID)
	h.challengeMu.Unlock()
}

func (h *DeviceLoginHandler) createInitialRefreshToken(deviceID uuid.UUID, userID uint, familyID uuid.UUID, ip, userAgent string) (*dbmodel.RefreshToken, string, error) {
	// Generate secret for HMAC validation
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, "", err
	}
	secret := base64.RawURLEncoding.EncodeToString(secretBytes)

	// Create secret hash using HMAC with RefreshHashSecret
	secretHash, err := h.generateSecretHash(secret)
	if err != nil {
		return nil, "", err
	}

	refreshToken := &dbmodel.RefreshToken{
		ID:         uuid.New(),
		UserID:     userID,
		DeviceID:   deviceID,
		FamilyID:   familyID,
		ParentID:   nil, // initial token
		SecretHash: secretHash,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(h.authConfig.RefreshTTL),
		RevokedAt:  nil,
		IP:         &ip,
		UserAgent:  &userAgent,
	}

	if err := h.refreshTokenRepo.Create(refreshToken); err != nil {
		return nil, "", err
	}

	return refreshToken, secret, nil
}

func (h *DeviceLoginHandler) generateSecretHash(secret string) (string, error) {
	if h.authConfig.RefreshHashSecret == "" {
		sum := sha256.Sum256([]byte(secret))
		return hex.EncodeToString(sum[:]), nil
	}
	mac := hmac.New(sha256.New, []byte(h.authConfig.RefreshHashSecret))
	mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (h *DeviceLoginHandler) generateAccessToken(user *dbmodel.User) (string, error) {
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
	return tokens.GenerateAuthToken(h.jwtManager, fmt.Sprintf("%d", user.ID), perms, h.authConfig.AccessTTL)
}

func (h *DeviceLoginHandler) formatUserResponse(user *dbmodel.User) interface{} {
	return map[string]interface{}{
		"id":                user.ID,
		"email":             user.Email,
		"status":            user.Status,
		"email_verified_at": user.EmailVerifiedAt,
	}
}
