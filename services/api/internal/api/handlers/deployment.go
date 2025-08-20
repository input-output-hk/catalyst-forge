package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	contracts "github.com/input-output-hk/catalyst-forge/services/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/deployment"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
	deploymentService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/deployment"
)

// DeploymentHandler handles deployment-related endpoints
type DeploymentHandler struct {
	*BaseHandler
	service deploymentService.Service
}

// NewDeploymentHandler creates a new deployment handler
func NewDeploymentHandler(service deploymentService.Service, _ interface{}, logger *slog.Logger) *DeploymentHandler {
	return &DeploymentHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// Create handles POST /api/v1/deployments
// @Summary Create a new deployment
// @Description Create a new deployment for a release to an environment
// @Tags deployments
// @Accept json
// @Produce json
// @Param deployment body contracts.DeploymentCreate true "Deployment creation request"
// @Success 201 {object} contracts.DeploymentResponse "Created deployment"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 404 {object} contracts.ErrorResponse "Release or environment not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/deployments [post]
func (h *DeploymentHandler) Create(c *gin.Context) {
	var req contracts.DeploymentCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	svcReq := deploymentService.CreateRequest{
		ReleaseID: uuid.MustParse(req.ReleaseID),
		EnvID:     uuid.MustParse(req.EnvironmentID),
		CreatedBy: req.DeployedBy,
	}

	// Store intent digest and status reason in IntentJSON if provided
	if req.IntentDigest != nil || req.StatusReason != nil {
		intents := make(map[string]interface{})
		if req.IntentDigest != nil {
			intents["digest"] = *req.IntentDigest
		}
		if req.StatusReason != nil {
			intents["status_reason"] = *req.StatusReason
		}
		svcReq.IntentJSON = intents
	}

	// Create deployment
	dep, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, deploymentService.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, deploymentService.ErrEnvironmentNotFound) {
			h.RespondWithNotFound(c, "Environment")
			return
		}
		// Note: ErrReleaseNotSealed doesn't exist in service, just use generic error
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(dep))
}

// GetByID handles GET /api/v1/deployments/:id
// @Summary Get a deployment by ID
// @Description Retrieve a single deployment by its ID
// @Tags deployments
// @Accept json
// @Produce json
// @Param id path string true "Deployment ID (UUID)"
// @Success 200 {object} contracts.DeploymentResponse "Deployment details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid deployment ID"
// @Failure 404 {object} contracts.ErrorResponse "Deployment not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/deployments/{id} [get]
func (h *DeploymentHandler) GetByID(c *gin.Context) {
	idStr := c.Param("deployment_id")
	id, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	dep, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, deploymentService.ErrDeploymentNotFound) {
			h.RespondWithNotFound(c, "Deployment")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(dep))
}

// List handles GET /api/v1/deployments
// @Summary List deployments
// @Description List deployments with optional filtering and pagination
// @Tags deployments
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param release_id query string false "Filter by release ID"
// @Param environment_id query string false "Filter by environment ID"
// @Param status query string false "Filter by status (pending, healthy, unhealthy, failed)"
// @Param deployed_by query string false "Filter by deployer"
// @Param since query string false "Filter by deployment date (RFC3339)"
// @Param until query string false "Filter by deployment date (RFC3339)"
// @Param sort_by query string false "Sort field (created_at, deployed_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.DeploymentPageResult "Paginated list of deployments"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/deployments [get]
func (h *DeploymentHandler) List(c *gin.Context) {
	var filter contracts.DeploymentListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Set default pagination if not provided
	if filter.Page == 0 {
		filter.Page = 1
	}
	if filter.PageSize == 0 {
		filter.PageSize = 20
	}

	// Convert to service filter
	svcFilter := deploymentService.ListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
	}

	if filter.ReleaseID != nil {
		id := uuid.MustParse(*filter.ReleaseID)
		svcFilter.ReleaseID = &id
	}
	if filter.EnvironmentID != nil {
		id := uuid.MustParse(*filter.EnvironmentID)
		svcFilter.EnvID = &id
	}
	if filter.Status != nil {
		status := enums.DeploymentStatus(*filter.Status)
		svcFilter.Status = &status
	}
	if filter.DeployedBy != nil {
		svcFilter.CreatedBy = filter.DeployedBy
	}
	if filter.Since != nil {
		svcFilter.Since = filter.Since
	}
	if filter.Until != nil {
		svcFilter.Until = filter.Until
	}

	deployments, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.DeploymentResponse, len(deployments))
	for i, dep := range deployments {
		items[i] = *h.toResponse(&dep)
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// Update handles PATCH /api/v1/deployments/:id
// @Summary Update a deployment
// @Description Update a deployment's status and status reason
// @Tags deployments
// @Accept json
// @Produce json
// @Param id path string true "Deployment ID (UUID)"
// @Param deployment body contracts.DeploymentUpdate true "Deployment update request"
// @Success 200 {object} contracts.DeploymentResponse "Updated deployment"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Deployment not found"
// @Failure 422 {object} contracts.ErrorResponse "Invalid status transition"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/deployments/{id} [patch]
func (h *DeploymentHandler) Update(c *gin.Context) {
	type pathParam struct {
		DeploymentID string `uri:"deployment_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.DeploymentUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	svcReq := deploymentService.UpdateRequest{}

	if req.Status != nil {
		status := enums.DeploymentStatus(*req.Status)
		svcReq.Status = &status
	}

	// Store status reason in IntentJSON if provided
	if req.StatusReason != nil {
		if svcReq.IntentJSON == nil {
			svcReq.IntentJSON = make(map[string]interface{})
		}
		svcReq.IntentJSON["status_reason"] = *req.StatusReason
		svcReq.LastError = req.StatusReason
	}

	id, err := h.ParseUUID(p.DeploymentID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	dep, err := h.service.Update(c.Request.Context(), id, svcReq)
	if err != nil {
		if errors.Is(err, deploymentService.ErrDeploymentNotFound) {
			h.RespondWithNotFound(c, "Deployment")
			return
		}
		if errors.Is(err, deploymentService.ErrInvalidStatus) {
			h.RespondWithUnprocessableEntity(c, "Invalid status transition", nil)
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(dep))
}

// Delete handles DELETE /api/v1/deployments/:id
// @Summary Delete a deployment
// @Description Delete a deployment
// @Tags deployments
// @Accept json
// @Produce json
// @Param id path string true "Deployment ID (UUID)"
// @Success 204 "Deployment deleted successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid deployment ID"
// @Failure 404 {object} contracts.ErrorResponse "Deployment not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/deployments/{id} [delete]
func (h *DeploymentHandler) Delete(c *gin.Context) {
	type pathParam struct {
		DeploymentID string `uri:"deployment_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	id, err := h.ParseUUID(p.DeploymentID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, deploymentService.ErrDeploymentNotFound) {
			h.RespondWithNotFound(c, "Deployment")
			return
		}
		// Note: These specific errors don't exist in service
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// toResponse converts a deployment model to response DTO
func (h *DeploymentHandler) toResponse(d *deployment.Deployment) *contracts.DeploymentResponse {
	resp := &contracts.DeploymentResponse{
		ID:            d.ID.String(),
		ReleaseID:     d.ReleaseID.String(),
		EnvironmentID: d.EnvID.String(),
		Status:        string(d.Status),
		IntentDigest:  d.IntentDigest,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.CreatedAt, // Deployment model doesn't have UpdatedAt
	}

	// Extract StatusReason and DeployedBy from IntentJSON if available
	if d.IntentJSON != nil {
		if statusReason, ok := d.IntentJSON["status_reason"].(string); ok {
			resp.StatusReason = &statusReason
		}
	}

	if d.CreatedBy != nil {
		resp.DeployedBy = d.CreatedBy
	}

	// DeployedAt would be derived from status changes or stored separately
	// For now, using created time when status is healthy
	if d.Status == enums.DeploymentStatusHealthy {
		resp.DeployedAt = &d.CreatedAt
	}

	return resp
}
