package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth/jwt"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userKeyService  user.UserKeyService
	userService     user.UserService
	userRoleService user.UserRoleService
	roleService     user.RoleService
	authManager     *auth.AuthManager
	jwtManager      *jwt.JWTManager
	logger          *slog.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userKeyService user.UserKeyService, userService user.UserService, userRoleService user.UserRoleService, roleService user.RoleService, authManager *auth.AuthManager, jwtManager *jwt.JWTManager, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		userKeyService:  userKeyService,
		userService:     userService,
		userRoleService: userRoleService,
		roleService:     roleService,
		authManager:     authManager,
		jwtManager:      jwtManager,
		logger:          logger,
	}
}

// ChallengeRequest represents the request body for creating a challenge
type ChallengeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Kid   string `json:"kid" binding:"required"`
}

// CreateChallenge handles the POST /auth/challenge endpoint
// @Summary Create a new authentication challenge
// @Description Create a new challenge for user authentication using Ed25519 keys
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ChallengeRequest true "Challenge creation request"
// @Success 200 {object} auth.KeyPairChallenge "Challenge created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "User key not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/challenge [post]
func (h *AuthHandler) CreateChallenge(c *gin.Context) {
	var req ChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Lookup the user key by kid
	userKey, err := h.userKeyService.GetUserKeyByKid(req.Kid)
	if err != nil {
		h.logger.Error("Failed to get user key by kid", "error", err, "kid", req.Kid)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User key not found",
		})
		return
	}

	// Verify that the user key belongs to the user
	user, err := h.userService.GetUserByID(userKey.UserID)
	if err != nil || user.Email != req.Email {
		h.logger.Warn("kid/email mismatch", "kid", req.Kid, "email", req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Convert user key to KeyPair
	keyPair, err := userKey.ToKeyPair()
	if err != nil {
		h.logger.Error("Failed to convert user key to key pair", "error", err, "kid", req.Kid)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process user key",
		})
		return
	}

	// Generate challenge with 60 second duration
	challenge, err := keyPair.GenerateChallenge(req.Email, 60*time.Second)
	if err != nil {
		h.logger.Error("Failed to generate challenge", "error", err, "kid", req.Kid)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate challenge",
		})
		return
	}

	// Store challenge in Redis cache
	challengeID, err := h.authManager.SaveChallenge(challenge)
	if err != nil {
		h.logger.Error("Failed to save challenge", "error", err, "kid", req.Kid)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save challenge",
		})
		return
	}

	h.logger.Info("Challenge created successfully",
		"challenge_id", challengeID,
		"kid", req.Kid,
		"email", req.Email)

	// Return the challenge to the user
	c.JSON(http.StatusOK, challenge)
}

// LoginResponse represents the response body for authentication
type LoginResponse struct {
	Token string `json:"token"`
}

// Login handles the POST /auth/login endpoint
// @Summary Authenticate user with challenge response
// @Description Authenticate a user using their signed challenge response
// @Tags auth
// @Accept json
// @Produce json
// @Param request body auth.KeyPairChallengeResponse true "Login request"
// @Success 200 {object} LoginResponse "Authentication successful"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication failed"
// @Failure 404 {object} map[string]interface{} "Challenge or user not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req auth.KeyPairChallengeResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get challenge ID and lookup the original challenge
	challengeID := req.ID()
	originalChallenge, err := h.authManager.LookupChallenge(challengeID)
	if err != nil {
		h.logger.Error("Failed to lookup challenge", "error", err, "challenge_id", challengeID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Challenge not found or expired",
		})
		return
	}

	// Validate that the challenge response fields match the original challenge
	if req.Challenge != originalChallenge.Challenge ||
		req.Email != originalChallenge.Email ||
		req.KeyID != originalChallenge.KeyID {
		h.logger.Error("Challenge response fields do not match original challenge",
			"challenge_id", challengeID,
			"original_challenge", originalChallenge.Challenge,
			"response_challenge", req.Challenge)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Challenge response validation failed",
		})
		// Clean up challenge regardless of failure
		h.authManager.RemoveChallenge(challengeID)
		return
	}

	// Lookup the user key using the kid
	userKey, err := h.userKeyService.GetUserKeyByKid(req.KeyID)
	if err != nil {
		h.logger.Error("Failed to get user key by kid", "error", err, "kid", req.KeyID)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User key not found",
		})
		// Clean up challenge regardless of failure
		h.authManager.RemoveChallenge(challengeID)
		return
	}

	// Convert user key to KeyPair
	keyPair, err := userKey.ToKeyPair()
	if err != nil {
		h.logger.Error("Failed to convert user key to key pair", "error", err, "kid", req.KeyID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process user key",
		})
		// Clean up challenge regardless of failure
		h.authManager.RemoveChallenge(challengeID)
		return
	}

	// Verify the challenge response
	if err := keyPair.VerifyChallenge(&req); err != nil {
		h.logger.Error("Challenge verification failed", "error", err, "kid", req.KeyID)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Challenge verification failed",
		})
		// Clean up challenge regardless of failure
		h.authManager.RemoveChallenge(challengeID)
		return
	}

	// Lookup the user from the challenge response
	user, err := h.userService.GetUserByEmail(req.Email)
	if err != nil {
		h.logger.Error("Failed to get user by email", "error", err, "email", req.Email)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		// Clean up challenge regardless of failure
		h.authManager.RemoveChallenge(challengeID)
		return
	}

	// Verify that the user key belongs to the user
	if userKey.UserID != user.ID {
		h.logger.Warn("kid does not belong to user", "kid", req.KeyID, "user_id", user.ID)
		h.authManager.RemoveChallenge(challengeID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Get user roles and their permissions
	permissions := make([]auth.Permission, 0)
	permissionSet := make(map[auth.Permission]bool) // Use map to ensure uniqueness

	// Get all roles assigned to the user
	userRoles, err := h.userRoleService.GetUserRoles(user.ID)
	if err != nil {
		h.logger.Error("Failed to get user roles", "error", err, "user_id", user.ID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user permissions",
		})
		// Clean up challenge regardless of failure
		h.authManager.RemoveChallenge(challengeID)
		return
	}

	// For each user role, get the role details and its permissions
	for _, userRole := range userRoles {
		role, err := h.roleService.GetRoleByID(userRole.RoleID)
		if err != nil {
			h.logger.Error("Failed to get role details", "error", err, "role_id", userRole.RoleID)
			continue // Skip this role but continue with others
		}

		// Add all permissions from this role to the set
		rolePermissions := role.GetPermissions()
		for _, permission := range rolePermissions {
			permissionSet[permission] = true
		}
	}

	// Convert the permission set back to a slice
	for permission := range permissionSet {
		permissions = append(permissions, permission)
	}

	// If user has no roles or no permissions, add basic permissions for active users
	// if len(permissions) == 0 && user.Status == "active" {
	// 	permissions = append(permissions, auth.PermUserRead)
	// }

	h.logger.Info("User permissions determined",
		"user_id", user.ID,
		"email", user.Email,
		"permissions", permissions,
		"roles_count", len(userRoles))

	// Generate JWT with 8 hour expiration
	token, err := h.jwtManager.GenerateToken(user.Email, permissions, 8*time.Hour)
	if err != nil {
		h.logger.Error("Failed to generate JWT token", "error", err, "user_id", user.ID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
		})
		// Clean up challenge regardless of failure
		h.authManager.RemoveChallenge(challengeID)
		return
	}

	h.logger.Info("User authenticated successfully",
		"user_id", user.ID,
		"email", user.Email,
		"challenge_id", challengeID)

	// Clean up challenge after successful authentication
	h.authManager.RemoveChallenge(challengeID)

	// Return the JWT token
	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
	})
}
