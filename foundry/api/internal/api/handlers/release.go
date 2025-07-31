package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
)

// ReleaseHandler handles HTTP requests related to releases
type ReleaseHandler struct {
	releaseService service.ReleaseService
	logger         *slog.Logger
}

// NewReleaseHandler creates a new instance of ReleaseHandler
func NewReleaseHandler(releaseService service.ReleaseService, logger *slog.Logger) *ReleaseHandler {
	return &ReleaseHandler{
		releaseService: releaseService,
		logger:         logger,
	}
}

// CreateReleaseRequest represents the request body for creating a release
type CreateReleaseRequest struct {
	SourceRepo   string `json:"source_repo" binding:"required"`
	SourceCommit string `json:"source_commit" binding:"required"`
	SourceBranch string `json:"source_branch"`
	Project      string `json:"project" binding:"required"`
	ProjectPath  string `json:"project_path" binding:"required"`
	Bundle       string `json:"bundle" binding:"required"`
}

// CreateRelease handles the POST /release endpoint
// @Summary Create a new release
// @Description Create a new release with the specified source repository and project details
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateReleaseRequest true "Release creation request"
// @Param deploy query string false "Deploy the release immediately" Enums(true, false, 1, 0)
// @Success 201 {object} models.Release "Release created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release [post]
func (h *ReleaseHandler) CreateRelease(c *gin.Context) {
	var req CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	release := &models.Release{
		SourceRepo:   req.SourceRepo,
		SourceCommit: req.SourceCommit,
		SourceBranch: req.SourceBranch,
		Project:      req.Project,
		ProjectPath:  req.ProjectPath,
		Bundle:       req.Bundle,
	}

	if err := h.releaseService.CreateRelease(c.Request.Context(), release); err != nil {
		h.logger.Error("Failed to create release", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create release: " + err.Error()})
		return
	}

	deployParam := c.Query("deploy")
	shouldDeploy := deployParam == "true" || deployParam == "1"

	if shouldDeploy {
		deploymentService := c.MustGet("deploymentService").(service.DeploymentService)
		deployment, err := deploymentService.CreateDeployment(c.Request.Context(), release.ID)
		if err != nil {
			h.logger.Error("Failed to create deployment", "error", err)
		} else {
			release.Deployments = []models.ReleaseDeployment{*deployment}
		}
	}

	c.JSON(http.StatusCreated, release)
}

// GetRelease handles the GET /release/{id} endpoint
// @Summary Get a release by ID
// @Description Retrieve a specific release by its ID
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Success 200 {object} models.Release "Release details"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Release not found"
// @Router /release/{id} [get]
func (h *ReleaseHandler) GetRelease(c *gin.Context) {
	id := c.Param("id")

	release, err := h.releaseService.GetRelease(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get release", "id", id, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Release not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, release)
}

// UpdateReleaseRequest represents the request body for updating a release
type UpdateReleaseRequest struct {
	SourceRepo   string `json:"source_repo"`
	SourceCommit string `json:"source_commit"`
	SourceBranch string `json:"source_branch"`
	ProjectPath  string `json:"project_path"`
	Bundle       string `json:"bundle"`
}

// UpdateRelease handles the PUT /release/{id} endpoint
// @Summary Update a release
// @Description Update an existing release with new information
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Param request body UpdateReleaseRequest true "Release update request"
// @Success 200 {object} models.Release "Release updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Release not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/{id} [put]
func (h *ReleaseHandler) UpdateRelease(c *gin.Context) {
	id := c.Param("id")

	// Get the existing release
	release, err := h.releaseService.GetRelease(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get release for update", "id", id, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Release not found: " + err.Error()})
		return
	}

	var req UpdateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Update the fields (only if provided in the request)
	if req.SourceRepo != "" {
		release.SourceRepo = req.SourceRepo
	}
	if req.SourceCommit != "" {
		release.SourceCommit = req.SourceCommit
	}
	if req.SourceBranch != "" || req.SourceBranch == "" && c.Request.Method == http.MethodPut {
		// Allow explicitly setting to empty string
		release.SourceBranch = req.SourceBranch
	}
	if req.ProjectPath != "" {
		release.ProjectPath = req.ProjectPath
	}
	if req.Bundle != "" {
		release.Bundle = req.Bundle
	}

	// Update the release
	if err := h.releaseService.UpdateRelease(c.Request.Context(), release); err != nil {
		h.logger.Error("Failed to update release", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update release: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, release)
}

// ListReleases handles the GET /releases endpoint
// @Summary List releases
// @Description Get all releases, optionally filtered by project
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project query string false "Filter releases by project name"
// @Success 200 {array} models.Release "List of releases"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /releases [get]
func (h *ReleaseHandler) ListReleases(c *gin.Context) {
	projectName := c.Query("project")

	var releases []models.Release
	var err error

	if projectName != "" {
		releases, err = h.releaseService.ListReleases(c.Request.Context(), projectName)
	} else {
		releases, err = h.releaseService.ListAllReleases(c.Request.Context())
	}

	if err != nil {
		h.logger.Error("Failed to list releases", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list releases: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, releases)
}

// GetReleaseByAlias handles GET /release/alias/{name} endpoint
// @Summary Get release by alias
// @Description Retrieve a release by its alias name
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name path string true "Release alias name"
// @Success 200 {object} models.Release "Release details"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Release alias not found"
// @Router /release/alias/{name} [get]
func (h *ReleaseHandler) GetReleaseByAlias(c *gin.Context) {
	name := c.Param("name")

	release, err := h.releaseService.GetReleaseByAlias(c.Request.Context(), name)
	if err != nil {
		h.logger.Error("Failed to get release by alias", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Release alias not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, release)
}

// CreateAliasRequest represents the request body for creating an alias
type CreateAliasRequest struct {
	ReleaseID string `json:"release_id" binding:"required"`
}

// CreateAlias handles POST /release/alias/{name} endpoint
// @Summary Create a release alias
// @Description Create an alias for a release
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name path string true "Alias name"
// @Param request body CreateAliasRequest true "Alias creation request"
// @Success 201 {object} map[string]interface{} "Alias created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/alias/{name} [post]
func (h *ReleaseHandler) CreateAlias(c *gin.Context) {
	name := c.Param("name")

	var req CreateAliasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if err := h.releaseService.CreateReleaseAlias(c.Request.Context(), name, req.ReleaseID); err != nil {
		h.logger.Error("Failed to create alias", "name", name, "releaseID", req.ReleaseID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alias: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"name": name, "release_id": req.ReleaseID})
}

// DeleteAlias handles DELETE /release/alias/{name} endpoint
// @Summary Delete a release alias
// @Description Delete an alias for a release
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param name path string true "Alias name"
// @Success 200 {object} map[string]interface{} "Alias deleted successfully"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/alias/{name} [delete]
func (h *ReleaseHandler) DeleteAlias(c *gin.Context) {
	name := c.Param("name")

	if err := h.releaseService.DeleteReleaseAlias(c.Request.Context(), name); err != nil {
		h.logger.Error("Failed to delete alias", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete alias: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alias deleted successfully"})
}

// ListAliases handles GET /release/{id}/aliases endpoint
// @Summary List release aliases
// @Description Get all aliases for a specific release
// @Tags releases
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Success 200 {array} models.ReleaseAlias "List of aliases"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/{id}/aliases [get]
func (h *ReleaseHandler) ListAliases(c *gin.Context) {
	releaseID := c.Param("id")

	aliases, err := h.releaseService.ListReleaseAliases(c.Request.Context(), releaseID)
	if err != nil {
		h.logger.Error("Failed to list aliases", "releaseID", releaseID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list aliases: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, aliases)
}
