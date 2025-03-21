package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/service"
)

// DeploymentHandler handles HTTP requests related to deployments
type DeploymentHandler struct {
	deploymentService service.DeploymentService
	logger            *slog.Logger
}

// NewDeploymentHandler creates a new instance of DeploymentHandler
func NewDeploymentHandler(deploymentService service.DeploymentService, logger *slog.Logger) *DeploymentHandler {
	return &DeploymentHandler{
		deploymentService: deploymentService,
		logger:            logger,
	}
}

// CreateDeployment handles the POST /release/{id}/deploy endpoint
func (h *DeploymentHandler) CreateDeployment(c *gin.Context) {
	releaseID := c.Param("id")

	deployment, err := h.deploymentService.CreateDeployment(c.Request.Context(), releaseID)
	if err != nil {
		h.logger.Error("Failed to create deployment", "releaseID", releaseID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deployment: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, deployment)
}

// GetDeployment handles the GET /release/:id/deploy/:deployId endpoint
func (h *DeploymentHandler) GetDeployment(c *gin.Context) {
	deploymentID := c.Param("deployId")

	deployment, err := h.deploymentService.GetDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		h.logger.Error("Failed to get deployment", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// UpdateDeploymentStatusRequest represents the request body for updating a deployment status
type UpdateDeploymentStatusRequest struct {
	Status models.DeploymentStatus `json:"status" binding:"required"`
	Reason string                  `json:"reason"`
}

// UpdateDeploymentStatus handles the PUT /release/:id/deploy/:deployId/status endpoint
func (h *DeploymentHandler) UpdateDeploymentStatus(c *gin.Context) {
	deploymentID := c.Param("deployId")

	var req UpdateDeploymentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if err := h.deploymentService.UpdateDeploymentStatus(c.Request.Context(), deploymentID, req.Status, req.Reason); err != nil {
		h.logger.Error("Failed to update deployment status", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update deployment status: " + err.Error()})
		return
	}

	deployment, err := h.deploymentService.GetDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		h.logger.Error("Failed to get updated deployment", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// ListDeployments handles the GET /release/{id}/deployments endpoint
func (h *DeploymentHandler) ListDeployments(c *gin.Context) {
	releaseID := c.Param("id")

	deployments, err := h.deploymentService.ListDeployments(c.Request.Context(), releaseID)
	if err != nil {
		h.logger.Error("Failed to list deployments", "releaseID", releaseID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list deployments: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployments)
}

// GetLatestDeployment handles the GET /release/{id}/deploy/latest endpoint
func (h *DeploymentHandler) GetLatestDeployment(c *gin.Context) {
	releaseID := c.Param("id")

	deployment, err := h.deploymentService.GetLatestDeployment(c.Request.Context(), releaseID)
	if err != nil {
		h.logger.Error("Failed to get latest deployment", "releaseID", releaseID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "No deployments found: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}
