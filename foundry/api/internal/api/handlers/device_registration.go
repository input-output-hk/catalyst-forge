package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// DeviceRegistrationHandler handles device-keypair registration endpoints.
type DeviceRegistrationHandler struct {
	inviteRepo          userrepo.InviteRepository
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
	challengeStorage    map[uuid.UUID]*DeviceInitChallenge // Indexed by DeviceID for O(1) lookup
	challengeMu         *sync.RWMutex
}

// DeviceInitChallenge represents server-side challenge storage.
type DeviceInitChallenge struct {
	DeviceID  uuid.UUID `json:"device_id"`
	Challenge string    `json:"challenge"`
	Algorithm string    `json:"alg"`
	ExpiresAt time.Time `json:"expires_at"`
	InviteID  uint      `json:"invite_id"`
	UserID    uint      `json:"user_id"`
}

// DeviceRegistrationInitRequest represents the request to initialize device registration.
type DeviceRegistrationInitRequest struct {
	Token    string `json:"token" binding:"required"`
	InviteID uint   `json:"invite_id" binding:"required"`
}

// DeviceRegistrationInitResponse represents the response from device initialization.
type DeviceRegistrationInitResponse struct {
	DeviceID  uuid.UUID `json:"device_id"`
	Challenge string    `json:"challenge"`
	Algorithm string    `json:"alg"`
	ExpiresAt time.Time `json:"expires_at"`
}

// DeviceRegisterRequest represents the device registration request.
type DeviceRegisterRequest struct {
	DeviceID     uuid.UUID              `json:"device_id" binding:"required"`
	DeviceName   string                 `json:"device_name" binding:"required"`
	PublicKeyJWK map[string]interface{} `json:"public_key_jwk" binding:"required"`
	DeviceProof  string                 `json:"device_proof" binding:"required"`
	Timestamp    int64                  `json:"timestamp" binding:"required"`
}

// DeviceRegisterResponse represents the successful device registration response.
type DeviceRegisterResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int         `json:"expires_in"`
	User        interface{} `json:"user"`
}

// NewDeviceRegistrationHandler creates a new device registration handler.
func NewDeviceRegistrationHandler(
	inviteRepo userrepo.InviteRepository,
	deviceRepo userrepo.DeviceRepository,
	refreshTokenRepo userrepo.RefreshTokenRepository,
	userService usersvc.UserService,
	roleService usersvc.RoleService,
	userRoleService usersvc.UserRoleService,
	jwtManager authjwt.JWTManager,
	authConfig *config.AuthConfig,
	logger *slog.Logger,
) *DeviceRegistrationHandler {
	cookieManager := auth.NewCookieManager(authConfig)
	deviceProofVerifier := auth.NewDeviceProofVerifier(deviceRepo, authConfig)

	return &DeviceRegistrationHandler{
		inviteRepo:          inviteRepo,
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
		challengeStorage:    make(map[uuid.UUID]*DeviceInitChallenge), // TODO: Use Redis or DB
		challengeMu:         &sync.RWMutex{},
	}
}

// InitDeviceRegistration initializes device registration flow with invite validation
// @Summary Initialize device registration
// @Description Initialize device registration flow for invited users, requires valid invite token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body DeviceRegistrationInitRequest true "Device registration initialization request"
// @Success 200 {object} DeviceRegistrationInitResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid or expired invite"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/devices/init [post].
func (h *DeviceRegistrationHandler) InitDeviceRegistration(c *gin.Context) {
	if os.Getenv("TEST_LOG") == "1" {
		h.logger.Info("[TEST] InitDeviceRegistration: begin")
	}
	var req DeviceRegistrationInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	// Validate invite token (using same logic as invite verification)
	invite, user, err := h.validateInviteToken(req.Token, req.InviteID)
	if err != nil {
		h.logger.Error("invite validation failed", "error", err, "invite_id", req.InviteID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired invite"})
		return
	}

	// Expose invite token to rate limiter for per-invite limiting
	if req.Token != "" {
		c.Set("invite_token", req.Token)
	}

	// Generate device ID and challenge
	deviceID := uuid.New()
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		h.logger.Error("failed to generate challenge", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes)

	// Store challenge with 5-minute TTL
	expiresAt := time.Now().Add(5 * time.Minute)
	challengeData := &DeviceInitChallenge{
		DeviceID:  deviceID,
		Challenge: challenge,
		Algorithm: "ES256", // ECDSA P-256
		ExpiresAt: expiresAt,
		InviteID:  invite.ID,
		UserID:    user.ID,
	}

	h.challengeMu.Lock()
	h.challengeStorage[deviceID] = challengeData
	h.challengeMu.Unlock()

	h.logger.Info("device initialization started",
		"device_id", deviceID,
		"user_id", user.ID,
		"invite_id", invite.ID)

	resp := DeviceRegistrationInitResponse{
		DeviceID:  deviceID,
		Challenge: challenge,
		Algorithm: "ES256",
		ExpiresAt: expiresAt,
	}
	if os.Getenv("TEST_LOG") == "1" {
		h.logger.Info("[TEST] InitDeviceRegistration: ok", "device_id", deviceID, "challenge", challenge)
	}
	c.JSON(http.StatusOK, resp)
}

// RegisterDevice completes device registration with proof verification
// @Summary Complete device registration
// @Description Complete device registration by providing device proof over challenge
// @Tags auth
// @Accept json
// @Produce json
// @Param request body DeviceRegisterRequest true "Device registration request"
// @Success 200 {object} DeviceRegisterResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid proof or expired challenge"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/devices/register [post].
func (h *DeviceRegistrationHandler) RegisterDevice(c *gin.Context) {
	if os.Getenv("TEST_LOG") == "1" {
		h.logger.Info("[TEST] RegisterDevice: begin")
	}
	var req DeviceRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request format"})
		return
	}

	// Look up the challenge by DeviceID (the proper way)
	// The challenge was issued for this specific DeviceID during initialization
	challengeData := h.findChallengeByDeviceID(req.DeviceID)
	if challengeData == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired challenge"})
		return
	}

	// Check challenge expiry
	if time.Now().After(challengeData.ExpiresAt) {
		h.challengeMu.Lock()
		delete(h.challengeStorage, req.DeviceID)
		h.challengeMu.Unlock()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "challenge expired"})
		return
	}

	// CRITICAL: Validate that the provided key matches our expected algorithm (ES256)
	// This provides an additional check at registration time
	if challengeData.Algorithm != "ES256" {
		h.logger.Error("challenge algorithm mismatch",
			"expected", "ES256",
			"challenge_alg", challengeData.Algorithm)
		c.JSON(http.StatusBadRequest, gin.H{"error": "algorithm mismatch"})
		return
	}

	// Ensure the client's JWK specifies ES256 if it has an alg field
	if algValue, hasAlg := req.PublicKeyJWK["alg"]; hasAlg {
		if alg, ok := algValue.(string); ok && alg != "ES256" {
			h.logger.Error("client provided non-ES256 algorithm",
				"provided_alg", alg,
				"device_id", req.DeviceID)
			c.JSON(http.StatusBadRequest, gin.H{"error": "only ES256 algorithm is supported"})
			return
		}
	}

	// Verify device proof over challenge using device proof verifier
	// Generate canonical string for challenge verification
	canonicalString := h.deviceProofVerifier.GenerateCanonicalStringForChallenge(
		req.Timestamp,
		req.DeviceID,
		challengeData.Challenge,
	)
	// Temporary diagnostics to compare client/server canonical during local dev
	h.logger.Debug("device register canonical",
		"canonical", canonicalString,
		"device_id", req.DeviceID,
		"ts", req.Timestamp,
	)

	// Verify signature using the provided public key
	// This will also validate the algorithm internally
	if err := h.deviceProofVerifier.VerifyDeviceProofWithKey(
		canonicalString,
		req.DeviceProof,
		req.PublicKeyJWK,
	); err != nil {
		h.logger.Error("device proof verification failed", "error", err, "device_id", req.DeviceID, "ts", req.Timestamp)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid device proof"})
		return
	}
	if os.Getenv("TEST_LOG") == "1" {
		h.logger.Info("[TEST] RegisterDevice: proof ok", "device_id", req.DeviceID)
	}

	// Create device record
	device, err := h.createDeviceRecord(req.DeviceID, req.DeviceName, req.PublicKeyJWK, challengeData.UserID)
	if err != nil {
		h.logger.Error("failed to create device record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register device"})
		return
	}

	// Create initial refresh token with family
	familyID := uuid.New()
	refreshToken, secret, err := h.createInitialRefreshToken(device.ID, challengeData.UserID, familyID, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		h.logger.Error("failed to create refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	if os.Getenv("TEST_LOG") == "1" {
		h.logger.Info("[TEST] RegisterDevice: refresh created", "jti", refreshToken.ID)
	}

	// Set refresh token cookie
	h.cookieManager.SetRefreshTokenCookie(c, refreshToken.ID, secret, h.authConfig.RefreshTTL)

	// Generate access token
	user, err := h.userService.GetUserByID(challengeData.UserID)
	if err != nil {
		h.logger.Error("failed to get user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user data"})
		return
	}

	accessToken, err := h.generateAccessToken(user)
	if err != nil {
		h.logger.Error("failed to generate access token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
		return
	}

	// Mark invite as redeemed
	if err := h.inviteRepo.MarkRedeemed(challengeData.InviteID); err != nil {
		h.logger.Error("failed to mark invite as redeemed", "error", err)
		// Don't fail the registration for this
	}

	// Clean up challenge
	h.challengeMu.Lock()
	delete(h.challengeStorage, req.DeviceID)
	h.challengeMu.Unlock()

	h.logger.Info("device registration completed successfully",
		"device_id", req.DeviceID,
		"user_id", challengeData.UserID,
		"device_name", req.DeviceName)

	c.JSON(http.StatusOK, DeviceRegisterResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(h.authConfig.AccessTTL.Seconds()),
		User:        h.formatUserResponse(user),
	})
}

// Helper methods

func (h *DeviceRegistrationHandler) validateInviteToken(token string, inviteID uint) (*dbmodel.Invite, *dbmodel.User, error) {
	// Use same HMAC/SHA-256 hex logic as invite handler
	secret := os.Getenv("INVITE_HASH_SECRET")
	var hexHash string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(token))
		hexHash = hex.EncodeToString(mac.Sum(nil))
	} else {
		sum := sha256.Sum256([]byte(token))
		hexHash = hex.EncodeToString(sum[:])
	}

	invite, err := h.inviteRepo.GetByTokenHash(hexHash)
	if err != nil || invite == nil {
		return nil, nil, err
	}

	if invite.ID != inviteID {
		return nil, nil, err
	}

	if invite.RedeemedAt != nil || time.Now().After(invite.ExpiresAt) {
		return nil, nil, err
	}

	// Get or create user
	user, err := h.userService.GetUserByEmail(invite.Email)
	if err != nil || user == nil {
		user = &dbmodel.User{
			Email:  invite.Email,
			Status: dbmodel.UserStatusActive,
		}
		if err := h.userService.CreateUser(user); err != nil {
			return nil, nil, err
		}
	}

	// Activate user and verify email if needed
	now := time.Now()
	user.EmailVerifiedAt = &now
	user.Status = dbmodel.UserStatusActive
    _ = h.userService.UpdateUser(user)

	// Assign roles from invite
	for _, roleName := range invite.Roles {
		role, err := h.roleService.GetRoleByName(roleName)
		if err == nil && role != nil {
            _ = h.userRoleService.AssignUserToRole(user.ID, role.ID)
		}
	}

	return invite, user, nil
}

func (h *DeviceRegistrationHandler) findChallengeByDeviceID(deviceID uuid.UUID) *DeviceInitChallenge {
	h.challengeMu.RLock()
	defer h.challengeMu.RUnlock()

	return h.challengeStorage[deviceID]
}

func (h *DeviceRegistrationHandler) createDeviceRecord(deviceID uuid.UUID, name string, publicKeyJWK map[string]interface{}, userID uint) (*dbmodel.Device, error) {
	// Generate JWK thumbprint for uniqueness constraint
	thumbprint, err := h.generateJWKThumbprint(publicKeyJWK)
	if err != nil {
		return nil, err
	}

	// Convert JWK map to JSON bytes for storage
	jwkBytes, err := json.Marshal(publicKeyJWK)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWK: %w", err)
	}

	now := time.Now()
	device := &dbmodel.Device{
		ID:            deviceID,
		UserID:        userID,
		Name:          name,
		PublicJWK:     jwkBytes,
		JWKThumbprint: thumbprint,
		Status:        dbmodel.DeviceStatusActive,
		CreatedAt:     time.Now(),
		LastUsedAt:    &now,
	}

	if err := h.deviceRepo.Create(device); err != nil {
		return nil, err
	}

	return device, nil
}

func (h *DeviceRegistrationHandler) generateJWKThumbprint(jwk map[string]interface{}) (string, error) {
	// Implement RFC 7638 JWK thumbprint generation
	// This creates a canonical representation of the key material and hashes it
	// to ensure the same key always produces the same thumbprint

	kty, ok := jwk["kty"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid kty field in JWK")
	}

	// Build the canonical JWK representation based on key type
	// RFC 7638 requires only the required members for each key type
	var canonicalJWK map[string]interface{}

	switch kty {
	case "EC":
		// For EC keys: crv, kty, x, y are required (in lexicographic order)
		crv, crvOk := jwk["crv"].(string)
		x, xOk := jwk["x"].(string)
		y, yOk := jwk["y"].(string)

		if !crvOk || !xOk || !yOk {
			return "", fmt.Errorf("missing required EC key parameters")
		}

		canonicalJWK = map[string]interface{}{
			"crv": crv,
			"kty": kty,
			"x":   x,
			"y":   y,
		}

	case "RSA":
		// For RSA keys: e, kty, n are required (in lexicographic order)
		e, eOk := jwk["e"].(string)
		n, nOk := jwk["n"].(string)

		if !eOk || !nOk {
			return "", fmt.Errorf("missing required RSA key parameters")
		}

		canonicalJWK = map[string]interface{}{
			"e":   e,
			"kty": kty,
			"n":   n,
		}

	case "OKP":
		// For OKP (EdDSA) keys: crv, kty, x are required
		crv, crvOk := jwk["crv"].(string)
		x, xOk := jwk["x"].(string)

		if !crvOk || !xOk {
			return "", fmt.Errorf("missing required OKP key parameters")
		}

		canonicalJWK = map[string]interface{}{
			"crv": crv,
			"kty": kty,
			"x":   x,
		}

	default:
		return "", fmt.Errorf("unsupported key type: %s", kty)
	}

	// Create the canonical JSON representation (no whitespace, sorted keys)
	canonicalJSON, err := json.Marshal(canonicalJWK)
	if err != nil {
		return "", fmt.Errorf("failed to marshal canonical JWK: %w", err)
	}

	// Compute SHA-256 hash of the canonical representation
	hash := sha256.Sum256(canonicalJSON)

	// Return base64url-encoded hash (RFC 7638 thumbprint)
	thumbprint := base64.RawURLEncoding.EncodeToString(hash[:])

	return thumbprint, nil
}

func (h *DeviceRegistrationHandler) createInitialRefreshToken(deviceID uuid.UUID, userID uint, familyID uuid.UUID, ip, userAgent string) (*dbmodel.RefreshToken, string, error) {
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
		ParentID:   nil, // Initial token has no parent
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

func (h *DeviceRegistrationHandler) generateSecretHash(secret string) (string, error) {
	if h.authConfig.RefreshHashSecret == "" {
		// Fallback to simple hash for development
		sum := sha256.Sum256([]byte(secret))
		return hex.EncodeToString(sum[:]), nil
	}
	mac := hmac.New(sha256.New, []byte(h.authConfig.RefreshHashSecret))
	mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (h *DeviceRegistrationHandler) generateAccessToken(user *dbmodel.User) (string, error) {
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

func (h *DeviceRegistrationHandler) formatUserResponse(user *dbmodel.User) interface{} {
	return map[string]interface{}{
		"id":                user.ID,
		"email":             user.Email,
		"status":            user.Status,
		"email_verified_at": user.EmailVerifiedAt,
	}
}
