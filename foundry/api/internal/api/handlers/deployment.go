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
// @Summary Create a deployment
// @Description Create a new deployment for a release
// @Tags deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Success 201 {object} models.ReleaseDeployment "Deployment created successfully"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/{id}/deploy [post]
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
// @Summary Get a deployment
// @Description Get a specific deployment by its ID
// @Tags deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Param deployId path string true "Deployment ID"
// @Success 200 {object} models.ReleaseDeployment "Deployment details"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "Deployment not found"
// @Router /release/{id}/deploy/{deployId} [get]
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
// @Summary Update a deployment
// @Description Update an existing deployment
// @Tags deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Param deployId path string true "Deployment ID"
// @Param request body models.ReleaseDeployment true "Deployment update request"
// @Success 200 {object} models.ReleaseDeployment "Deployment updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/{id}/deploy/{deployId} [put]
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
// @Summary List deployments
// @Description Get all deployments for a release
// @Tags deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Success 200 {array} models.ReleaseDeployment "List of deployments"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/{id}/deployments [get]
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
// @Summary Get latest deployment
// @Description Get the most recent deployment for a release
// @Tags deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Success 200 {object} models.ReleaseDeployment "Latest deployment"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "No deployments found"
// @Router /release/{id}/deploy/latest [get]
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
// @Summary Add deployment event
// @Description Add an event to a deployment
// @Tags deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Param deployId path string true "Deployment ID"
// @Param request body AddEventRequest true "Event details"
// @Success 200 {object} models.ReleaseDeployment "Deployment with updated events"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/{id}/deploy/{deployId}/events [post]
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
// @Summary Get deployment events
// @Description Get all events for a deployment
// @Tags deployments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release ID"
// @Param deployId path string true "Deployment ID"
// @Success 200 {array} models.DeploymentEvent "List of deployment events"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /release/{id}/deploy/{deployId}/events [get]
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
