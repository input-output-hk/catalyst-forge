package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	contracts "github.com/input-output-hk/catalyst-forge/services/api/internal/contracts"
	model "github.com/input-output-hk/catalyst-forge/services/api/internal/models/release"
	renderedRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/release"
	renderedService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/release"
)

// RenderedReleaseHandler handles rendered release endpoints
type RenderedReleaseHandler struct {
	*BaseHandler
	service renderedService.RenderedService
}

// NewRenderedReleaseHandler creates a new rendered release handler
func NewRenderedReleaseHandler(service renderedService.RenderedService, logger *slog.Logger) *RenderedReleaseHandler {
	return &RenderedReleaseHandler{BaseHandler: NewBaseHandler(logger), service: service}
}

// Create handles POST /api/v1/rendered-releases
// @Summary Create a rendered release record
// @Description Create a rendered release associated with a specific deployment, release, and environment
// @Tags rendered-releases
// @Accept json
// @Produce json
// @Param rendered_release body contracts.RenderedReleaseCreate true "Rendered release creation request"
// @Success 201 {object} contracts.RenderedReleaseResponse "Created rendered release"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 409 {object} contracts.ErrorResponse "Rendered release already exists"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/rendered-releases [post]
func (h *RenderedReleaseHandler) Create(c *gin.Context) {
	var req contracts.RenderedReleaseCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	depID, err := h.ParseUUID(req.DeploymentID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	relID, err := h.ParseUUID(req.ReleaseID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	envID, err := h.ParseUUID(req.EnvironmentID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	svcReq := renderedService.RenderedCreateRequest{
		DeploymentID:        depID,
		ReleaseID:           relID,
		EnvironmentID:       envID,
		RendererVersion:     req.RendererVersion,
		ModuleVersions:      req.ModuleVersions,
		BundleHash:          req.BundleHash,
		OutputHash:          req.OutputHash,
		OCIRef:              req.OCIRef,
		OCIDigest:           req.OCIDigest,
		StorageURI:          req.StorageURI,
		Signed:              req.Signed,
		SignatureVerifiedAt: req.SignatureVerifiedAt,
	}

	rr, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, renderedRepo.ErrRenderedReleaseExists) {
			h.RespondWithConflict(c, "Rendered release already exists")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(rr))
}

// GetByID handles GET /api/v1/rendered-releases/:rendered_release_id
// @Summary Get a rendered release by ID
// @Description Retrieve a single rendered release by its ID
// @Tags rendered-releases
// @Accept json
// @Produce json
// @Param rendered_release_id path string true "Rendered Release ID (UUID)"
// @Success 200 {object} contracts.RenderedReleaseResponse "Rendered release details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid rendered release ID"
// @Failure 404 {object} contracts.ErrorResponse "Rendered release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/rendered-releases/{rendered_release_id} [get]
func (h *RenderedReleaseHandler) GetByID(c *gin.Context) {
	idStr := c.Param("rendered_release_id")
	id, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	rr, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, renderedRepo.ErrRenderedReleaseNotFound) {
			h.RespondWithNotFound(c, "RenderedRelease")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}
	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(rr))
}

// GetByDeployment handles GET /api/v1/deployments/:deployment_id/rendered-release
// @Summary Get a rendered release by deployment ID
// @Description Retrieve the rendered release associated with a deployment
// @Tags rendered-releases
// @Accept json
// @Produce json
// @Param deployment_id path string true "Deployment ID (UUID)"
// @Success 200 {object} contracts.RenderedReleaseResponse "Rendered release for deployment"
// @Failure 400 {object} contracts.ErrorResponse "Invalid deployment ID"
// @Failure 404 {object} contracts.ErrorResponse "Rendered release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/deployments/{deployment_id}/rendered-release [get]
func (h *RenderedReleaseHandler) GetByDeployment(c *gin.Context) {
	depStr := c.Param("deployment_id")
	depID, err := h.ParseUUID(depStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	rr, err := h.service.GetByDeployment(c.Request.Context(), depID)
	if err != nil {
		if errors.Is(err, renderedRepo.ErrRenderedReleaseNotFound) {
			h.RespondWithNotFound(c, "RenderedRelease")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}
	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(rr))
}

// List handles GET /api/v1/rendered-releases
// @Summary List rendered releases
// @Description List rendered releases with optional filtering and pagination
// @Tags rendered-releases
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param release_id query string false "Filter by release ID"
// @Param environment_id query string false "Filter by environment ID"
// @Param deployment_id query string false "Filter by deployment ID"
// @Param oci_digest query string false "Filter by OCI digest"
// @Param output_hash query string false "Filter by output hash"
// @Param since query string false "Filter by creation date (RFC3339)"
// @Param until query string false "Filter by creation date (RFC3339)"
// @Param sort_by query string false "Sort field (created_at, updated_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.RenderedReleasePageResult "Paginated list of rendered releases"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/rendered-releases [get]
func (h *RenderedReleaseHandler) List(c *gin.Context) {
	var filter contracts.RenderedReleaseListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	svcFilter := renderedService.RenderedListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
	}
	if filter.ReleaseID != nil {
		id, err := h.ParseUUID(*filter.ReleaseID)
		if err != nil {
			h.RespondWithValidationError(c, err)
			return
		}
		svcFilter.ReleaseID = &id
	}
	if filter.EnvironmentID != nil {
		id, err := h.ParseUUID(*filter.EnvironmentID)
		if err != nil {
			h.RespondWithValidationError(c, err)
			return
		}
		svcFilter.EnvironmentID = &id
	}
	if filter.DeploymentID != nil {
		id, err := h.ParseUUID(*filter.DeploymentID)
		if err != nil {
			h.RespondWithValidationError(c, err)
			return
		}
		svcFilter.DeploymentID = &id
	}
	svcFilter.OCIDigest = filter.OCIDigest
	svcFilter.OutputHash = filter.OutputHash

	items, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	respItems := make([]contracts.RenderedReleaseResponse, len(items))
	for i := range items {
		respItems[i] = *h.toResponse(&items[i])
	}
	result := contracts.NewPageResult(respItems, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// Update handles PATCH /api/v1/rendered-releases/:rendered_release_id
// @Summary Update a rendered release
// @Description Update a rendered release's metadata (OCI fields, signature, storage URI)
// @Tags rendered-releases
// @Accept json
// @Produce json
// @Param rendered_release_id path string true "Rendered Release ID (UUID)"
// @Param rendered_release body contracts.RenderedReleaseUpdate true "Rendered release update request"
// @Success 200 {object} contracts.RenderedReleaseResponse "Updated rendered release"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Rendered release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/rendered-releases/{rendered_release_id} [patch]
func (h *RenderedReleaseHandler) Update(c *gin.Context) {
	idStr := c.Param("rendered_release_id")
	id, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.RenderedReleaseUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	svcReq := renderedService.RenderedUpdateRequest{
		OCIRef:              req.OCIRef,
		OCIDigest:           req.OCIDigest,
		StorageURI:          req.StorageURI,
		Signed:              req.Signed,
		SignatureVerifiedAt: req.SignatureVerifiedAt,
	}
	rr, err := h.service.Update(c.Request.Context(), id, svcReq)
	if err != nil {
		if errors.Is(err, renderedRepo.ErrRenderedReleaseNotFound) {
			h.RespondWithNotFound(c, "RenderedRelease")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}
	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(rr))
}

// Delete handles DELETE /api/v1/rendered-releases/:rendered_release_id
// @Summary Delete a rendered release
// @Description Delete a rendered release by ID
// @Tags rendered-releases
// @Accept json
// @Produce json
// @Param rendered_release_id path string true "Rendered Release ID (UUID)"
// @Success 204 "Rendered release deleted successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid rendered release ID"
// @Failure 404 {object} contracts.ErrorResponse "Rendered release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/rendered-releases/{rendered_release_id} [delete]
func (h *RenderedReleaseHandler) Delete(c *gin.Context) {
	idStr := c.Param("rendered_release_id")
	id, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, renderedRepo.ErrRenderedReleaseNotFound) {
			h.RespondWithNotFound(c, "RenderedRelease")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RenderedReleaseHandler) toResponse(rr *model.RenderedRelease) *contracts.RenderedReleaseResponse {
	resp := &contracts.RenderedReleaseResponse{
		ID:                  rr.ID.String(),
		DeploymentID:        rr.DeploymentID.String(),
		ReleaseID:           rr.ReleaseID.String(),
		EnvironmentID:       rr.EnvironmentID.String(),
		RendererVersion:     rr.RendererVersion,
		BundleHash:          rr.BundleHash,
		OutputHash:          rr.OutputHash,
		OCIRef:              rr.OCIRef,
		OCIDigest:           rr.OCIDigest,
		StorageURI:          rr.StorageURI,
		Signed:              rr.Signed,
		SignatureVerifiedAt: rr.SignatureVerifiedAt,
		CreatedAt:           rr.CreatedAt,
		UpdatedAt:           rr.UpdatedAt,
	}
	// ModuleVersions: best-effort mapping to []map[string]interface{}
	// The model uses datatypes.JSON; if needed, decode in service layer. Here we omit for brevity.
	return resp
}
