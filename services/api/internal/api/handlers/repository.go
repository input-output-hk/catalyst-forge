package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	contracts "github.com/input-output-hk/catalyst-forge/services/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/repository"
	repositoryService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/repository"
)

// RepositoryHandler handles repository-related endpoints
type RepositoryHandler struct {
	*BaseHandler
	service repositoryService.Service
}

// NewRepositoryHandler creates a new repository handler
func NewRepositoryHandler(service repositoryService.Service, logger *slog.Logger) *RepositoryHandler {
	return &RepositoryHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// GetByID handles GET /api/v1/repositories/:repo_id
// @Summary Get a repository by ID
// @Description Retrieve a single repository by its ID
// @Tags repositories
// @Accept json
// @Produce json
// @Param repo_id path string true "Repository ID (UUID)"
// @Success 200 {object} contracts.RepositoryResponse "Repository details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid repository ID"
// @Failure 404 {object} contracts.ErrorResponse "Repository not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/repositories/{repo_id} [get]
func (h *RepositoryHandler) GetByID(c *gin.Context) {
	type pathParam struct {
		RepoID string `uri:"repo_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	id, err := h.ParseUUID(p.RepoID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	repo, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repositoryService.ErrRepositoryNotFound) {
			h.RespondWithNotFound(c, "Repository")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(repo))
}

// GetByPath handles GET /api/v1/repositories/by-path/:host/:org/:name
// @Summary Get a repository by path
// @Description Retrieve a single repository by its host, organization, and name
// @Tags repositories
// @Accept json
// @Produce json
// @Param host path string true "Repository host (e.g., github.com)"
// @Param org path string true "Organization name"
// @Param name path string true "Repository name"
// @Success 200 {object} contracts.RepositoryResponse "Repository details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid parameters"
// @Failure 404 {object} contracts.ErrorResponse "Repository not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/repositories/by-path/{host}/{org}/{name} [get]
func (h *RepositoryHandler) GetByPath(c *gin.Context) {
	var param contracts.RepositoryPathParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	repo, err := h.service.GetByHostOrgName(c.Request.Context(), param.Host, param.Org, param.Name)
	if err != nil {
		if errors.Is(err, repositoryService.ErrRepositoryNotFound) {
			h.RespondWithNotFound(c, "Repository")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(repo))
}

// List handles GET /api/v1/repositories
// @Summary List repositories
// @Description List repositories with optional filtering and pagination
// @Tags repositories
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param host query string false "Filter by host"
// @Param org query string false "Filter by organization"
// @Param name query string false "Filter by name"
// @Param sort_by query string false "Sort field (created_at, updated_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.RepositoryPageResult "Paginated list of repositories"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/repositories [get]
func (h *RepositoryHandler) List(c *gin.Context) {
	var filter contracts.RepositoryListFilter
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
	svcFilter := repositoryService.ListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
		Host:       filter.Host,
		Org:        filter.Org,
		Name:       filter.Name,
	}

	repositories, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.RepositoryResponse, len(repositories))
	for i, repo := range repositories {
		items[i] = *h.toResponse(&repo)
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// toResponse converts a repository model to response DTO
func (h *RepositoryHandler) toResponse(r *repository.Repository) *contracts.RepositoryResponse {
	return &contracts.RepositoryResponse{
		ID:        r.ID.String(),
		Host:      r.Host,
		Org:       r.Org,
		Name:      r.Name,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
