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

// UpdateDeployment handles the PUT /release/:id/deploy/:deployId endpoint
func (h *DeploymentHandler) UpdateDeployment(c *gin.Context) {
	deploymentID := c.Param("deployId")

	var deployment models.ReleaseDeployment
	if err := c.ShouldBindJSON(&deployment); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Ensure the deployment ID in the path matches the one in the request body
	if deployment.ID != deploymentID {
		h.logger.Error("Deployment ID mismatch", "pathID", deploymentID, "bodyID", deployment.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Deployment ID in path does not match ID in request body"})
		return
	}

	if err := h.deploymentService.UpdateDeployment(c.Request.Context(), &deployment); err != nil {
		h.logger.Error("Failed to update deployment", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update deployment: " + err.Error()})
		return
	}

	// Get the updated deployment to return
	updatedDeployment, err := h.deploymentService.GetDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		h.logger.Error("Failed to get updated deployment", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusOK, gin.H{"message": "Deployment updated successfully"})
		return
	}

	c.JSON(http.StatusOK, updatedDeployment)
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

// AddEventRequest represents the request body for adding a deployment event
type AddEventRequest struct {
	Name    string `json:"name" binding:"required"`
	Message string `json:"message" binding:"required"`
}

// AddDeploymentEvent handles the POST /release/:id/deploy/:deployId/events endpoint
func (h *DeploymentHandler) AddDeploymentEvent(c *gin.Context) {
	deploymentID := c.Param("deployId")

	var req AddEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if err := h.deploymentService.AddDeploymentEvent(c.Request.Context(), deploymentID, req.Name, req.Message); err != nil {
		h.logger.Error("Failed to add deployment event", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add deployment event: " + err.Error()})
		return
	}

	// Return the updated deployment with events
	deployment, err := h.deploymentService.GetDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		h.logger.Error("Failed to get updated deployment", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusOK, gin.H{"message": "Event added successfully"})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// GetDeploymentEvents handles the GET /release/:id/deploy/:deployId/events endpoint
func (h *DeploymentHandler) GetDeploymentEvents(c *gin.Context) {
	deploymentID := c.Param("deployId")

	events, err := h.deploymentService.GetDeploymentEvents(c.Request.Context(), deploymentID)
	if err != nil {
		h.logger.Error("Failed to get deployment events", "deploymentID", deploymentID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get deployment events: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}
