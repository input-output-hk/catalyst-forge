package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	contracts "github.com/input-output-hk/catalyst-forge/services/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/environment"
	environmentService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/environment"
)

// EnvironmentHandler handles environment-related endpoints
type EnvironmentHandler struct {
	*BaseHandler
	service environmentService.Service
}

// NewEnvironmentHandler creates a new environment handler
func NewEnvironmentHandler(service environmentService.Service, logger *slog.Logger) *EnvironmentHandler {
	return &EnvironmentHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// Create handles POST /api/v1/environments
// @Summary Create a new environment
// @Description Create a new environment for deployments
// @Tags environments
// @Accept json
// @Produce json
// @Param environment body contracts.EnvironmentCreate true "Environment creation request"
// @Success 201 {object} contracts.EnvironmentResponse "Created environment"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 409 {object} contracts.ErrorResponse "Environment already exists"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/environments [post]
func (h *EnvironmentHandler) Create(c *gin.Context) {
	var req contracts.EnvironmentCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Map the rich DTO to the simpler service model
	// The actual environment model only has Name, Cluster, ArgoProject, IsProtected
	svcReq := environmentService.CreateRequest{
		Name:        req.Name,
		Cluster:     req.Name, // Using name as cluster identifier for now
		IsProtected: false,    // Default to not protected
	}

	// Use ClusterRef if provided, otherwise use name
	if req.ClusterRef != nil {
		svcReq.Cluster = *req.ClusterRef
	}

	// Determine protection based on environment type
	if req.EnvironmentType == "prod" {
		svcReq.IsProtected = true
	}

	// Store namespace as ArgoProject if provided
	if req.Namespace != nil {
		svcReq.ArgoProject = req.Namespace
	}

	// Create environment
	env, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, environmentService.ErrEnvironmentExists) {
			h.RespondWithConflict(c, "Environment with this name already exists")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(env, req.ProjectID, req.EnvironmentType))
}

// GetByID handles GET /api/v1/environments/:id
// @Summary Get an environment by ID
// @Description Retrieve a single environment by its ID
// @Tags environments
// @Accept json
// @Produce json
// @Param id path string true "Environment ID (UUID)"
// @Success 200 {object} contracts.EnvironmentResponse "Environment details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid environment ID"
// @Failure 404 {object} contracts.ErrorResponse "Environment not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/environments/{id} [get]
func (h *EnvironmentHandler) GetByID(c *gin.Context) {
	idStr := c.Param("environment_id")
	envID, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	env, err := h.service.GetByID(c.Request.Context(), envID)
	if err != nil {
		if errors.Is(err, environmentService.ErrEnvironmentNotFound) {
			h.RespondWithNotFound(c, "Environment")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	// We don't have ProjectID or EnvironmentType stored, so using placeholders
	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(env, "", ""))
}

// GetByProjectAndName handles GET /api/v1/projects/:project_id/environments/:name
// @Summary Get an environment by project and name
// @Description Retrieve a single environment by project ID and environment name
// @Tags environments
// @Accept json
// @Produce json
// @Param project_id path string true "Project ID (UUID)"
// @Param name path string true "Environment name"
// @Success 200 {object} contracts.EnvironmentResponse "Environment details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid parameters"
// @Failure 404 {object} contracts.ErrorResponse "Environment not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/projects/{project_id}/environments/{name} [get]
func (h *EnvironmentHandler) GetByProjectAndName(c *gin.Context) {
	// Bind path params as strings to avoid tight coupling to uuid.UUID in transport
	type envPathParam struct {
		ProjectID string `uri:"project_id" binding:"required"`
		Name      string `uri:"name" binding:"required"`
	}
	var param envPathParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Current service supports lookup by name only.
	env, err := h.service.GetByName(c.Request.Context(), param.Name)
	if err != nil {
		if errors.Is(err, environmentService.ErrEnvironmentNotFound) {
			h.RespondWithNotFound(c, "Environment")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(env, param.ProjectID, ""))
}

// List handles GET /api/v1/environments
// @Summary List environments
// @Description List environments with optional filtering and pagination
// @Tags environments
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param project_id query string false "Filter by project ID"
// @Param name query string false "Filter by name"
// @Param environment_type query string false "Filter by type (dev, staging, prod)"
// @Param cluster_ref query string false "Filter by cluster reference"
// @Param namespace query string false "Filter by namespace"
// @Param active query bool false "Filter by active status"
// @Param sort_by query string false "Sort field (created_at, updated_at, name)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.EnvironmentPageResult "Paginated list of environments"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/environments [get]
func (h *EnvironmentHandler) List(c *gin.Context) {
	var filter contracts.EnvironmentListFilter
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

	// Convert to service filter - only Name, Cluster, ArgoProject, IsProtected are supported
	svcFilter := environmentService.ListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
	}

	if filter.Name != nil {
		svcFilter.Name = filter.Name
	}

	if filter.ClusterRef != nil {
		svcFilter.Cluster = filter.ClusterRef
	}

	if filter.Namespace != nil {
		svcFilter.ArgoProject = filter.Namespace
	}

	// Map environment type to protection status
	if filter.EnvironmentType != nil && *filter.EnvironmentType == "prod" {
		isProtected := true
		svcFilter.IsProtected = &isProtected
	}

	environments, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.EnvironmentResponse, len(environments))
	for i, env := range environments {
		// We don't have ProjectID or EnvironmentType stored, so using defaults
		items[i] = *h.toResponse(&env, "", h.inferEnvironmentType(&env))
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// Update handles PATCH /api/v1/environments/:id
// @Summary Update an environment
// @Description Update an environment's configuration
// @Tags environments
// @Accept json
// @Produce json
// @Param id path string true "Environment ID (UUID)"
// @Param environment body contracts.EnvironmentUpdate true "Environment update request"
// @Success 200 {object} contracts.EnvironmentResponse "Updated environment"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Environment not found"
// @Failure 409 {object} contracts.ErrorResponse "Environment is protected"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/environments/{id} [patch]
func (h *EnvironmentHandler) Update(c *gin.Context) {
	idStr := c.Param("environment_id")

	var req contracts.EnvironmentUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request - only Name, Cluster, ArgoProject, IsProtected can be updated
	svcReq := environmentService.UpdateRequest{
		Name: req.Name,
	}

	if req.ClusterRef != nil {
		svcReq.Cluster = req.ClusterRef
	}

	if req.Namespace != nil {
		svcReq.ArgoProject = req.Namespace
	}

	// Map environment type to protection status
	if req.EnvironmentType != nil {
		isProtected := *req.EnvironmentType == "prod"
		svcReq.IsProtected = &isProtected
	}

	envID, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	env, err := h.service.Update(c.Request.Context(), envID, svcReq)
	if err != nil {
		if errors.Is(err, environmentService.ErrEnvironmentNotFound) {
			h.RespondWithNotFound(c, "Environment")
			return
		}
		if errors.Is(err, environmentService.ErrProtectedEnvironment) {
			h.RespondWithConflict(c, "Environment is protected and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(env, "", h.inferEnvironmentType(env)))
}

// Delete handles DELETE /api/v1/environments/:id
// @Summary Delete an environment
// @Description Delete an environment if it has no deployments
// @Tags environments
// @Accept json
// @Produce json
// @Param id path string true "Environment ID (UUID)"
// @Success 204 "Environment deleted successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid environment ID"
// @Failure 404 {object} contracts.ErrorResponse "Environment not found"
// @Failure 409 {object} contracts.ErrorResponse "Environment has deployments or is protected"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/environments/{id} [delete]
func (h *EnvironmentHandler) Delete(c *gin.Context) {
	idStr := c.Param("environment_id")

	envID, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), envID); err != nil {
		if errors.Is(err, environmentService.ErrEnvironmentNotFound) {
			h.RespondWithNotFound(c, "Environment")
			return
		}
		if errors.Is(err, environmentService.ErrEnvironmentInUse) {
			h.RespondWithConflict(c, "Environment has deployments and cannot be deleted")
			return
		}
		if errors.Is(err, environmentService.ErrProtectedEnvironment) {
			h.RespondWithConflict(c, "Environment is protected and cannot be deleted")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// toResponse converts an environment model to response DTO
// Since the actual model is simpler than the DTO, we need to provide defaults or derive values
func (h *EnvironmentHandler) toResponse(e *environment.Environment, projectID string, envType string) *contracts.EnvironmentResponse {
	resp := &contracts.EnvironmentResponse{
		ID:              e.ID.String(),
		ProjectID:       projectID, // Not stored in model, passed from context
		Name:            e.Name,
		EnvironmentType: envType, // Not stored in model, inferred or passed
		ClusterRef:      &e.Cluster,
		Active:          true, // Default to active since we don't track this
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}

	// Use ArgoProject as Namespace if available
	if e.ArgoProject != nil {
		resp.Namespace = e.ArgoProject
	}

	// Store protection rules based on IsProtected flag
	if e.IsProtected {
		resp.ProtectionRules = map[string]interface{}{
			"protected": true,
			"reason":    "Production environment",
		}
	}

	// If no environment type was provided, try to infer it
	if resp.EnvironmentType == "" {
		resp.EnvironmentType = h.inferEnvironmentType(e)
	}

	// If no project ID was provided, use a placeholder
	if resp.ProjectID == "" {
		// In a real implementation, this would need to be fetched from a relationship
		resp.ProjectID = "00000000-0000-0000-0000-000000000000"
	}

	return resp
}

// inferEnvironmentType tries to determine the environment type from the model
func (h *EnvironmentHandler) inferEnvironmentType(e *environment.Environment) string {
	if e.IsProtected {
		return "prod"
	}
	// Could also check name patterns
	if e.Name == "staging" || e.Name == "stage" {
		return "staging"
	}
	return "dev" // Default to dev
}
