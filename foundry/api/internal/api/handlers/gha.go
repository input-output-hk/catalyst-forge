package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	auth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth"
	ghauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/auth/github"
)

// GHAHandler handles GitHub Actions token validation endpoints
type GHAHandler struct {
	authManager *auth.AuthManager
	oidcClient  ghauth.GithubActionsOIDCClient
	logger      *slog.Logger
}

// NewGHAHandler creates a new GHA token validation handler
func NewGHAHandler(authManager *auth.AuthManager, oidcClient ghauth.GithubActionsOIDCClient, logger *slog.Logger) *GHAHandler {
	return &GHAHandler{
		authManager: authManager,
		oidcClient:  oidcClient,
		logger:      logger,
	}
}

// ValidateTokenRequest represents the request body for token validation
type ValidateTokenRequest struct {
	Token    string `json:"token" binding:"required"`
	Audience string `json:"audience,omitempty"`
}

// ValidateTokenResponse represents the response body for token validation
type ValidateTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    string    `json:"user_id"`
}

// ValidateToken handles the /gha/validate endpoint
func (h *GHAHandler) ValidateToken(c *gin.Context) {
	var req ValidateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Validate the GitHub Actions token
	tokenInfo, err := h.oidcClient.Verify(req.Token, req.Audience)
	if err != nil {
		h.logger.Warn("Failed to verify GHA token", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid GitHub Actions token",
		})
		return
	}

	// Create a user ID from the repository information
	userID := tokenInfo.Repository
	if tokenInfo.RepositoryOwner != "" {
		userID = tokenInfo.RepositoryOwner + "/" + tokenInfo.Repository
	}

	// Generate permissions based on the token information
	permissions := h.generatePermissions(tokenInfo)

	// Generate a new JWT token
	expiration := 1 * time.Hour // 1 hour expiration
	token, err := h.authManager.GenerateToken(userID, permissions, expiration)
	if err != nil {
		h.logger.Error("Failed to generate JWT token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(expiration)

	h.logger.Info("Successfully validated GHA token and generated JWT",
		"repository", tokenInfo.Repository,
		"user_id", userID,
		"permissions", permissions)

	c.JSON(http.StatusOK, ValidateTokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    userID,
	})
}

// generatePermissions generates permissions based on the GitHub Actions token information
func (h *GHAHandler) generatePermissions(tokenInfo *ghauth.TokenInfo) []auth.Permission {
	permissions := []auth.Permission{
		auth.PermReleaseRead,
		auth.PermDeploymentRead,
		auth.PermDeploymentEventRead,
	}

	// Add write permissions for main branch or specific environments
	if tokenInfo.Ref == "refs/heads/main" || tokenInfo.Ref == "refs/heads/master" {
		permissions = append(permissions,
			auth.PermReleaseWrite,
			auth.PermDeploymentWrite,
			auth.PermDeploymentEventWrite,
		)
	}

	// Add write permissions for specific environments
	if tokenInfo.Environment == "production" || tokenInfo.Environment == "staging" {
		permissions = append(permissions,
			auth.PermReleaseWrite,
			auth.PermDeploymentWrite,
			auth.PermDeploymentEventWrite,
		)
	}

	return permissions
}
