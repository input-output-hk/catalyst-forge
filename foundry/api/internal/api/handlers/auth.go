package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
    "github.com/google/uuid"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	userKeyService  user.UserKeyService
	userService     user.UserService
	userRoleService user.UserRoleService
	roleService     user.RoleService
	authManager     *auth.AuthManager
	jwtManager      jwt.JWTManager
	logger          *slog.Logger
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userKeyService user.UserKeyService, userService user.UserService, userRoleService user.UserRoleService, roleService user.RoleService, authManager *auth.AuthManager, jwtManager jwt.JWTManager, logger *slog.Logger) *AuthHandler {
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

// ChallengeResponse represents the response body for a challenge request
type ChallengeResponse struct {
	Token string `json:"token"`
}

// CreateChallenge handles the POST /auth/challenge endpoint
// @Summary Create a new authentication challenge
// @Description Create a new challenge for user authentication using Ed25519 keys
// @Tags auth
// @Accept json
// @Produce json
// @Param request body ChallengeRequest true "Challenge creation request"
// @Success 200 {object} ChallengeResponse "Challenge created successfully"
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

	// Generate challenge JWT with 60 second duration
	challenge, _, err := tokens.GenerateChallengeJWT(h.jwtManager, req.Email, req.Kid, 60*time.Second)
	if err != nil {
		h.logger.Error("Failed to generate challenge", "error", err, "kid", req.Kid)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate challenge",
		})
		return
	}

	h.logger.Info("Challenge created successfully",
		"kid", req.Kid,
		"email", req.Email)

	// Return the challenge token to the user
	c.JSON(http.StatusOK, ChallengeResponse{
		Token: challenge,
	})
}

// LoginResponse represents the response body for authentication
type LoginResponse struct {
	Token string `json:"token"`
}

// LoginRequest represents the request body for authentication
type LoginRequest struct {
	Token     string `json:"token"`
	Signature string `json:"signature"`
}

// Login handles the POST /auth/login endpoint
// @Summary Authenticate user with challenge response
// @Description Authenticate a user using their signed challenge response
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} LoginResponse "Authentication successful"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication failed"
// @Failure 404 {object} map[string]interface{} "Challenge or user not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Validate the challenge token
	claims, err := tokens.VerifyChallengeJWT(h.jwtManager, req.Token)
	if err != nil {
		h.logger.Error("Failed to validate challenge token", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Challenge validation failed",
		})
		return
	}

	// Lookup the user key using the kid
	userKey, err := h.userKeyService.GetUserKeyByKid(claims.Kid)
	if err != nil {
		h.logger.Error("Failed to get user key by kid", "error", err, "kid", claims.Kid)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User key not found",
		})
		return
	}

	// Convert user key to KeyPair
	keyPair, err := userKey.ToKeyPair()
	if err != nil {
		h.logger.Error("Failed to convert user key to key pair", "error", err, "kid", claims.Kid)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process user key",
		})
		return
	}

	// Verify the challenge response
	if err := keyPair.VerifySignature(claims.Nonce, req.Signature); err != nil {
		h.logger.Error("Challenge verification failed", "error", err, "kid", claims.Kid)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Challenge verification failed",
		})
		return
	}

	// Lookup the user from the challenge response
	user, err := h.userService.GetUserByEmail(claims.Email)
	if err != nil {
		h.logger.Error("Failed to get user by email", "error", err, "email", claims.Email)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	// Verify that the user key belongs to the user
	if userKey.UserID != user.ID {
		h.logger.Warn("kid does not belong to user", "kid", claims.Kid, "user_id", user.ID)
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

    // Generate JWT with 30 minute expiration and new claims (jti, akid, user_ver)
    token, err := tokens.GenerateAuthToken(
        h.jwtManager,
        user.Email,
        permissions,
        30*time.Minute,
        jwt.WithTokenID(uuid.NewString()),
        jwt.WithAdditionalClaims(map[string]interface{}{
            "akid":     userKey.Kid,
            "user_ver": user.UserVer,
        }),
    )
	if err != nil {
		h.logger.Error("Failed to generate JWT token", "error", err, "user_id", user.ID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
		})
		return
	}

	h.logger.Info("User authenticated successfully",
		"user_id", user.ID,
		"email", user.Email)

	// Return the JWT token
	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
	})
}
