package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
	auth "github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth"
	ghauth "github.com/input-output-hk/catalyst-forge/foundry/api/pkg/auth/github"
)

// GHAHandler handles GitHub Actions authentication endpoints
type GHAHandler struct {
	authManager *auth.AuthManager
	oidcClient  ghauth.GithubActionsOIDCClient
	authService service.GHAAuthService
	logger      *slog.Logger
}

// NewGHAHandler creates a new GHA authentication handler
func NewGHAHandler(authManager *auth.AuthManager, oidcClient ghauth.GithubActionsOIDCClient, authService service.GHAAuthService, logger *slog.Logger) *GHAHandler {
	return &GHAHandler{
		authManager: authManager,
		oidcClient:  oidcClient,
		authService: authService,
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

// CreateAuthRequest represents the request body for creating a GHA authentication configuration
type CreateAuthRequest struct {
	Repository  string            `json:"repository" binding:"required"`
	Permissions []auth.Permission `json:"permissions" binding:"required"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description,omitempty"`
}

// UpdateAuthRequest represents the request body for updating a GHA authentication configuration
type UpdateAuthRequest struct {
	Repository  string            `json:"repository" binding:"required"`
	Permissions []auth.Permission `json:"permissions" binding:"required"`
	Enabled     bool              `json:"enabled"`
	Description string            `json:"description,omitempty"`
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

	// Get permissions from database for this repository
	permissions, err := h.authService.GetPermissionsForRepository(tokenInfo.Repository)
	if err != nil {
		h.logger.Warn("No authentication configuration found for repository",
			"repository", tokenInfo.Repository, "error", err)
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Repository not authorized for GitHub Actions authentication",
		})
		return
	}

	// Generate a new JWT token
	expiration := 1 * time.Hour // 1 hour expiration
	token, err := h.authManager.GenerateToken(tokenInfo.Repository, permissions, expiration)
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
		"user_id", tokenInfo.Repository,
		"permissions", permissions)

	c.JSON(http.StatusOK, ValidateTokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    tokenInfo.Repository,
	})
}

// CreateAuth handles the POST /gha/auth endpoint
func (h *GHAHandler) CreateAuth(c *gin.Context) {
	var req CreateAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get the authenticated user from context
	user, exists := c.Get("user")
	if !exists {
		h.logger.Warn("No authenticated user found")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	authenticatedUser := user.(*middleware.AuthenticatedUser)

	// Create the authentication configuration
	auth := &models.GHARepositoryAuth{
		Repository:  req.Repository,
		Enabled:     req.Enabled,
		Description: req.Description,
		CreatedBy:   authenticatedUser.ID,
		UpdatedBy:   authenticatedUser.ID,
	}
	auth.SetPermissions(req.Permissions)

	if err := h.authService.CreateAuth(auth); err != nil {
		h.logger.Error("Failed to create GHA authentication configuration", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create authentication configuration",
		})
		return
	}

	h.logger.Info("Successfully created GHA authentication configuration",
		"repository", req.Repository,
		"created_by", authenticatedUser.ID)

	c.JSON(http.StatusCreated, auth)
}

// GetAuth handles the GET /gha/auth/:id endpoint
func (h *GHAHandler) GetAuth(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.Warn("Invalid ID parameter", "id", idStr, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID parameter",
		})
		return
	}

	auth, err := h.authService.GetAuthByID(uint(id))
	if err != nil {
		h.logger.Warn("Failed to get GHA authentication configuration", "id", id, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Authentication configuration not found",
		})
		return
	}

	c.JSON(http.StatusOK, auth)
}

// GetAuthByRepository handles the GET /gha/auth/repository/:repository endpoint
func (h *GHAHandler) GetAuthByRepository(c *gin.Context) {
	repository := c.Param("repository")

	auth, err := h.authService.GetAuthByRepository(repository)
	if err != nil {
		h.logger.Warn("Failed to get GHA authentication configuration", "repository", repository, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Authentication configuration not found",
		})
		return
	}

	c.JSON(http.StatusOK, auth)
}

// UpdateAuth handles the PUT /gha/auth/:id endpoint
func (h *GHAHandler) UpdateAuth(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.Warn("Invalid ID parameter", "id", idStr, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID parameter",
		})
		return
	}

	var req UpdateAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get the authenticated user from context
	user, exists := c.Get("user")
	if !exists {
		h.logger.Warn("No authenticated user found")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication required",
		})
		return
	}

	authenticatedUser := user.(*middleware.AuthenticatedUser)

	// Get the existing configuration
	existing, err := h.authService.GetAuthByID(uint(id))
	if err != nil {
		h.logger.Warn("Failed to get existing GHA authentication configuration", "id", id, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Authentication configuration not found",
		})
		return
	}

	// Update the configuration
	existing.Repository = req.Repository
	existing.Enabled = req.Enabled
	existing.Description = req.Description
	existing.UpdatedBy = authenticatedUser.ID
	existing.SetPermissions(req.Permissions)

	if err := h.authService.UpdateAuth(existing); err != nil {
		h.logger.Error("Failed to update GHA authentication configuration", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update authentication configuration",
		})
		return
	}

	h.logger.Info("Successfully updated GHA authentication configuration",
		"id", id,
		"repository", req.Repository,
		"updated_by", authenticatedUser.ID)

	c.JSON(http.StatusOK, existing)
}

// DeleteAuth handles the DELETE /gha/auth/:id endpoint
func (h *GHAHandler) DeleteAuth(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		h.logger.Warn("Invalid ID parameter", "id", idStr, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid ID parameter",
		})
		return
	}

	if err := h.authService.DeleteAuth(uint(id)); err != nil {
		h.logger.Error("Failed to delete GHA authentication configuration", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete authentication configuration",
		})
		return
	}

	h.logger.Info("Successfully deleted GHA authentication configuration", "id", id)

	c.JSON(http.StatusOK, gin.H{
		"message": "Authentication configuration deleted successfully",
	})
}

// ListAuths handles the GET /gha/auth endpoint
func (h *GHAHandler) ListAuths(c *gin.Context) {
	auths, err := h.authService.ListAuths()
	if err != nil {
		h.logger.Error("Failed to list GHA authentication configurations", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list authentication configurations",
		})
		return
	}

	c.JSON(http.StatusOK, auths)
}
