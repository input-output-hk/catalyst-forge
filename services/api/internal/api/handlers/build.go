package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	contracts "github.com/input-output-hk/catalyst-forge/services/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/build"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/enums"
	buildService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/build"
)

// BuildHandler handles build-related endpoints
type BuildHandler struct {
	*BaseHandler
	service buildService.Service
}

// NewBuildHandler creates a new build handler
func NewBuildHandler(service buildService.Service, logger *slog.Logger) *BuildHandler {
	return &BuildHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// Create handles POST /api/v1/builds
// @Summary Create a new build
// @Description Create a new build record for a project
// @Tags builds
// @Accept json
// @Produce json
// @Param build body contracts.BuildCreate true "Build creation request"
// @Success 201 {object} contracts.BuildResponse "Created build"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 404 {object} contracts.ErrorResponse "Repository or project not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/builds [post]
func (h *BuildHandler) Create(c *gin.Context) {
	var req contracts.BuildCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	svcReq := buildService.CreateRequest{
		RepoID:        uuid.MustParse(req.RepoID),
		ProjectID:     uuid.MustParse(req.ProjectID),
		CommitSHA:     req.CommitSHA,
		Branch:        req.Branch,
		WorkflowRunID: req.WorkflowRunID,
		Status:        enums.BuildStatus(req.Status),
		RunnerEnv:     req.RunnerEnv,
	}

	if req.TraceID != nil {
		id := uuid.MustParse(*req.TraceID)
		svcReq.TraceID = &id
	}

	// Create build
	b, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, buildService.ErrRepositoryNotFound) {
			h.RespondWithNotFound(c, "Repository")
			return
		}
		if errors.Is(err, buildService.ErrProjectNotFound) {
			h.RespondWithNotFound(c, "Project")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(b))
}

// GetByID handles GET /api/v1/builds/:id
// @Summary Get a build by ID
// @Description Retrieve a single build by its ID
// @Tags builds
// @Accept json
// @Produce json
// @Param id path string true "Build ID (UUID)"
// @Success 200 {object} contracts.BuildResponse "Build details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid build ID"
// @Failure 404 {object} contracts.ErrorResponse "Build not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/builds/{id} [get]
func (h *BuildHandler) GetByID(c *gin.Context) {
	type pathParam struct {
		BuildID string `uri:"build_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	id, err := h.ParseUUID(p.BuildID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	b, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, buildService.ErrBuildNotFound) {
			h.RespondWithNotFound(c, "Build")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(b))
}

// List handles GET /api/v1/builds
// @Summary List builds
// @Description List builds with optional filtering and pagination
// @Tags builds
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param trace_id query string false "Filter by trace ID"
// @Param repo_id query string false "Filter by repository ID"
// @Param project_id query string false "Filter by project ID"
// @Param commit_sha query string false "Filter by commit SHA"
// @Param branch query string false "Filter by branch"
// @Param workflow_run_id query string false "Filter by workflow run ID"
// @Param status query string false "Filter by status (pending, running, succeeded, failed)"
// @Param since query string false "Filter by creation date (RFC3339)"
// @Param until query string false "Filter by creation date (RFC3339)"
// @Param sort_by query string false "Sort field (created_at, updated_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.BuildPageResult "Paginated list of builds"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/builds [get]
func (h *BuildHandler) List(c *gin.Context) {
	var filter contracts.BuildListFilter
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
	svcFilter := buildService.ListFilter{
		Pagination:    h.GetPagination(c),
		Sort:          h.GetSort(c),
		CommitSHA:     filter.CommitSHA,
		Branch:        filter.Branch,
		WorkflowRunID: filter.WorkflowRunID,
		Since:         filter.Since,
		Until:         filter.Until,
	}

	if filter.TraceID != nil {
		id := uuid.MustParse(*filter.TraceID)
		svcFilter.TraceID = &id
	}

	if filter.RepoID != nil {
		id := uuid.MustParse(*filter.RepoID)
		svcFilter.RepoID = &id
	}

	if filter.ProjectID != nil {
		id := uuid.MustParse(*filter.ProjectID)
		svcFilter.ProjectID = &id
	}

	if filter.Status != nil {
		status := enums.BuildStatus(*filter.Status)
		svcFilter.Status = &status
	}

	builds, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.BuildResponse, len(builds))
	for i, b := range builds {
		items[i] = *h.toResponse(&b)
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// Update handles PATCH /api/v1/builds/:id
// @Summary Update a build
// @Description Update a build's status and metadata
// @Tags builds
// @Accept json
// @Produce json
// @Param id path string true "Build ID (UUID)"
// @Param build body contracts.BuildUpdate true "Build update request"
// @Success 200 {object} contracts.BuildResponse "Updated build"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Build not found"
// @Failure 422 {object} contracts.ErrorResponse "Invalid status transition"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/builds/{id} [patch]
func (h *BuildHandler) Update(c *gin.Context) {
	type pathParam struct {
		BuildID string `uri:"build_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.BuildUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	svcReq := buildService.UpdateRequest{
		WorkflowRunID: req.WorkflowRunID,
		RunnerEnv:     req.RunnerEnv,
		FinishedAt:    req.FinishedAt,
	}

	if req.Status != nil {
		status := enums.BuildStatus(*req.Status)
		svcReq.Status = &status
	}

	id, err := h.ParseUUID(p.BuildID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	b, err := h.service.Update(c.Request.Context(), id, svcReq)
	if err != nil {
		if errors.Is(err, buildService.ErrBuildNotFound) {
			h.RespondWithNotFound(c, "Build")
			return
		}
		if errors.Is(err, buildService.ErrInvalidStatus) {
			h.RespondWithUnprocessableEntity(c, "Invalid status transition", nil)
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(b))
}

// UpdateStatus handles PATCH /api/v1/builds/:id/status
// @Summary Update build status
// @Description Update only the status of a build
// @Tags builds
// @Accept json
// @Produce json
// @Param id path string true "Build ID (UUID)"
// @Param status body contracts.BuildStatusUpdate true "Status update request"
// @Success 204 "Status updated successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Build not found"
// @Failure 422 {object} contracts.ErrorResponse "Invalid status transition"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/builds/{id}/status [patch]
func (h *BuildHandler) UpdateStatus(c *gin.Context) {
	type pathParam struct {
		BuildID string `uri:"build_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.BuildStatusUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	status := enums.BuildStatus(req.Status)

	id, err := h.ParseUUID(p.BuildID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.UpdateStatus(c.Request.Context(), id, status); err != nil {
		if errors.Is(err, buildService.ErrBuildNotFound) {
			h.RespondWithNotFound(c, "Build")
			return
		}
		if errors.Is(err, buildService.ErrInvalidStatus) {
			h.RespondWithUnprocessableEntity(c, "Invalid status transition", nil)
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// toResponse converts a build model to response DTO
func (h *BuildHandler) toResponse(b *build.Build) *contracts.BuildResponse {
	resp := &contracts.BuildResponse{
		ID:            b.ID.String(),
		RepoID:        b.RepoID.String(),
		ProjectID:     b.ProjectID.String(),
		CommitSHA:     b.CommitSHA,
		Branch:        b.Branch,
		WorkflowRunID: b.WorkflowRunID,
		Status:        string(b.Status),
		FinishedAt:    b.FinishedAt,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
	}

	if b.TraceID != nil {
		traceStr := b.TraceID.String()
		resp.TraceID = &traceStr
	}

	if b.RunnerEnv != nil {
		resp.RunnerEnv = map[string]interface{}(b.RunnerEnv)
	}

	return resp
}
