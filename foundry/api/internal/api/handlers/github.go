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
	auth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	ghauth "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/github"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
)

// GithubRepositoryAuthResponse represents the response structure for GHA authentication
// This is used to avoid the pq.StringArray issue in Swagger generation
type GithubRepositoryAuthResponse struct {
	ID          uint      `json:"id"`
	Repository  string    `json:"repository"`
	Permissions []string  `json:"permissions"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description,omitempty"`
	CreatedBy   string    `json:"created_by"`
	UpdatedBy   string    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GithubHandler handles GitHub Actions authentication endpoints
type GithubHandler struct {
	jwtManager  *jwt.JWTManager
	oidcClient  ghauth.GithubActionsOIDCClient
	authService service.GithubAuthService
	logger      *slog.Logger
}

// NewGithubHandler creates a new GitHub authentication handler
func NewGithubHandler(jwtManager *jwt.JWTManager, oidcClient ghauth.GithubActionsOIDCClient, authService service.GithubAuthService, logger *slog.Logger) *GithubHandler {
	return &GithubHandler{
		jwtManager:  jwtManager,
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

// ValidateToken handles the /auth/github/login endpoint
// @Summary Validate GitHub Actions token
// @Description Validate a GitHub Actions OIDC token and return a JWT token
// @Tags gha
// @Accept json
// @Produce json
// @Param request body ValidateTokenRequest true "Token validation request"
// @Success 200 {object} ValidateTokenResponse "Token validated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid token"
// @Failure 403 {object} map[string]interface{} "Repository not authorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/github/login [post]
func (h *GithubHandler) ValidateToken(c *gin.Context) {
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
	token, err := h.jwtManager.GenerateToken(tokenInfo.Repository, permissions, expiration)
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

// CreateAuth handles the POST /auth/github endpoint
// @Summary Create GHA authentication configuration
// @Description Create a new GitHub Actions authentication configuration for a repository
// @Tags gha
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateAuthRequest true "GHA authentication configuration"
// @Success 201 {object} GithubRepositoryAuthResponse "Authentication configuration created"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/github [post]
func (h *GithubHandler) CreateAuth(c *gin.Context) {
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
	auth := &models.GithubRepositoryAuth{
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

// GetAuth handles the GET /auth/github/:id endpoint
// @Summary Get GHA authentication configuration by ID
// @Description Get a specific GitHub Actions authentication configuration by its ID
// @Tags gha
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Authentication configuration ID"
// @Success 200 {object} GithubRepositoryAuthResponse "Authentication configuration"
// @Failure 400 {object} map[string]interface{} "Invalid ID parameter"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Authentication configuration not found"
// @Router /auth/github/{id} [get]
func (h *GithubHandler) GetAuth(c *gin.Context) {
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

// GetAuthByRepository handles the GET /auth/github/repository/:repository endpoint
// @Summary Get GHA authentication configuration by repository
// @Description Get a GitHub Actions authentication configuration by repository name
// @Tags gha
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param repository path string true "Repository name"
// @Success 200 {object} GithubRepositoryAuthResponse "Authentication configuration"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Authentication configuration not found"
// @Router /auth/github/repository/{repository} [get]
func (h *GithubHandler) GetAuthByRepository(c *gin.Context) {
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

// UpdateAuth handles the PUT /auth/github/:id endpoint
// @Summary Update GHA authentication configuration
// @Description Update an existing GitHub Actions authentication configuration
// @Tags gha
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Authentication configuration ID"
// @Param request body UpdateAuthRequest true "Updated GHA authentication configuration"
// @Success 200 {object} GithubRepositoryAuthResponse "Authentication configuration updated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Authentication configuration not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/github/{id} [put]
func (h *GithubHandler) UpdateAuth(c *gin.Context) {
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

// DeleteAuth handles the DELETE /auth/github/:id endpoint
// @Summary Delete GHA authentication configuration
// @Description Delete a GitHub Actions authentication configuration
// @Tags gha
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Authentication configuration ID"
// @Success 200 {object} map[string]interface{} "Authentication configuration deleted"
// @Failure 400 {object} map[string]interface{} "Invalid ID parameter"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/github/{id} [delete]
func (h *GithubHandler) DeleteAuth(c *gin.Context) {
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

// ListAuths handles the GET /auth/github endpoint
// @Summary List GHA authentication configurations
// @Description Get all GitHub Actions authentication configurations
// @Tags gha
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} GithubRepositoryAuthResponse "List of authentication configurations"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/github [get]
func (h *GithubHandler) ListAuths(c *gin.Context) {
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
