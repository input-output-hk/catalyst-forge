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
