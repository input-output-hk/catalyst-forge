package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	contracts "github.com/input-output-hk/catalyst-forge/foundry/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/project"
	projectService "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/project"
)

// ProjectHandler handles project-related endpoints 
type ProjectHandler struct {
	*BaseHandler
	service projectService.Service
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(service projectService.Service, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// GetByID handles GET /api/v1/projects/:id
// @Summary Get a project by ID
// @Description Retrieve a single project by its ID
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "Project ID (UUID)"
// @Success 200 {object} contracts.ProjectResponse "Project details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid project ID"
// @Failure 404 {object} contracts.ErrorResponse "Project not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/projects/{id} [get]
func (h *ProjectHandler) GetByID(c *gin.Context) {
	var param contracts.ProjectIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	proj, err := h.service.GetByID(c.Request.Context(), param.ProjectID)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) {
			h.RespondWithNotFound(c, "Project")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(proj))
}

// GetByRepoAndPath handles GET /api/v1/repositories/:repo_id/projects/by-path
// @Summary Get a project by repository and path
// @Description Retrieve a single project by repository ID and project path
// @Tags projects
// @Accept json
// @Produce json
// @Param repo_id path string true "Repository ID (UUID)"
// @Param path query string true "Project path"
// @Success 200 {object} contracts.ProjectResponse "Project details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid parameters"
// @Failure 404 {object} contracts.ErrorResponse "Project not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/repositories/{repo_id}/projects/by-path [get]
func (h *ProjectHandler) GetByRepoAndPath(c *gin.Context) {
	var param contracts.ProjectPathParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	repoID := uuid.MustParse(param.RepoID)
	proj, err := h.service.GetByRepoAndPath(c.Request.Context(), repoID, param.Path)
	if err != nil {
		if errors.Is(err, projectService.ErrProjectNotFound) {
			h.RespondWithNotFound(c, "Project")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(proj))
}

// List handles GET /api/v1/projects
// @Summary List projects
// @Description List projects with optional filtering and pagination
// @Tags projects
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param repo_id query string false "Filter by repository ID"
// @Param path query string false "Filter by path"
// @Param slug query string false "Filter by slug"
// @Param status query string false "Filter by status (active, archived)"
// @Param sort_by query string false "Sort field (created_at, updated_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.ProjectPageResult "Paginated list of projects"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/projects [get]
func (h *ProjectHandler) List(c *gin.Context) {
	var filter contracts.ProjectListFilter
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
	svcFilter := projectService.ListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
	}

	if filter.RepoID != nil {
		id := uuid.MustParse(*filter.RepoID)
		svcFilter.RepoID = &id
	}

	if filter.Path != nil {
		svcFilter.Path = filter.Path
	}

	if filter.Slug != nil {
		svcFilter.Slug = filter.Slug
	}

	if filter.Status != nil {
		status := project.ProjectStatus(*filter.Status)
		svcFilter.Status = &status
	}

	projects, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.ProjectResponse, len(projects))
	for i, proj := range projects {
		items[i] = *h.toResponse(&proj)
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// toResponse converts a project model to response DTO
func (h *ProjectHandler) toResponse(p *project.Project) *contracts.ProjectResponse {
	resp := &contracts.ProjectResponse{
		ID:        p.ID.String(),
		RepoID:    p.RepoID.String(),
		Path:      p.Path,
		Slug:      p.Slug,
		Status:    string(p.Status),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}

	if p.DisplayName != nil {
		resp.DisplayName = p.DisplayName
	}

	if p.BlueprintFingerprint != nil {
		resp.BlueprintFingerprint = p.BlueprintFingerprint
	}

	if p.FirstSeenCommit != nil {
		resp.FirstSeenCommit = p.FirstSeenCommit
	}

	if p.LastSeenCommit != nil {
		resp.LastSeenCommit = p.LastSeenCommit
	}

	return resp
}
