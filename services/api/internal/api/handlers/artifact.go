package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	contracts "github.com/input-output-hk/catalyst-forge/services/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/artifact"
	artifactService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/artifact"
)

// ArtifactHandler handles artifact-related endpoints
type ArtifactHandler struct {
	*BaseHandler
	service artifactService.Service
}

// NewArtifactHandler creates a new artifact handler
func NewArtifactHandler(service artifactService.Service, logger *slog.Logger) *ArtifactHandler {
	return &ArtifactHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// Create handles POST /api/v1/artifacts
// @Summary Create a new artifact
// @Description Create a new container artifact associated with a build
// @Tags artifacts
// @Accept json
// @Produce json
// @Param artifact body contracts.ArtifactCreate true "Artifact creation request"
// @Success 201 {object} contracts.ArtifactResponse "Created artifact"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 404 {object} contracts.ErrorResponse "Build not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/artifacts [post]
func (h *ArtifactHandler) Create(c *gin.Context) {
	var req contracts.ArtifactCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request based on actual artifact model
	// The v2 artifact is generic - can be OCI image, GitHub asset, S3 object, SBOM, etc.
	svcReq := artifactService.CreateRequest{
		BuildID: uuid.MustParse(req.BuildID),
		Kind:    "oci-image", // Default to oci-image for container artifacts
	}

	// Map the container-specific fields to the generic artifact model
	if req.ImageName != "" {
		svcReq.Name = &req.ImageName
	}

	if req.ImageDigest != "" {
		svcReq.Digest = &req.ImageDigest
	}

	// Store container-specific metadata in the Labels field
	labels := make(map[string]interface{})
	if req.Tag != nil {
		labels["tag"] = *req.Tag
	}
	if req.Repo != nil {
		labels["repo"] = *req.Repo
	}
	if req.Provider != nil {
		labels["provider"] = *req.Provider
	}
	if len(labels) > 0 {
		svcReq.Labels = labels
	}

	// Store build and scan info in Metadata
	metadata := make(map[string]interface{})
	if req.BuildArgs != nil {
		metadata["build_args"] = req.BuildArgs
	}
	if req.BuildMeta != nil {
		metadata["build_meta"] = req.BuildMeta
	}
	if req.ScanStatus != nil {
		metadata["scan_status"] = *req.ScanStatus
	}
	if req.ScanResults != nil {
		metadata["scan_results"] = req.ScanResults
	}
	if req.SignedBy != nil {
		metadata["signed_by"] = *req.SignedBy
	}
	if len(metadata) > 0 {
		svcReq.Metadata = metadata
	}

	// Create artifact
	art, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, artifactService.ErrBuildNotFound) {
			h.RespondWithNotFound(c, "Build")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(art))
}

// GetByID handles GET /api/v1/artifacts/:id
// @Summary Get an artifact by ID
// @Description Retrieve a single artifact by its ID
// @Tags artifacts
// @Accept json
// @Produce json
// @Param id path string true "Artifact ID (UUID)"
// @Success 200 {object} contracts.ArtifactResponse "Artifact details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid artifact ID"
// @Failure 404 {object} contracts.ErrorResponse "Artifact not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/artifacts/{id} [get]
func (h *ArtifactHandler) GetByID(c *gin.Context) {
	type pathParam struct {
		ArtifactID string `uri:"artifact_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	id, err := h.ParseUUID(p.ArtifactID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	art, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, artifactService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(art))
}

// GetByDigest handles GET /api/v1/artifacts/digest/:digest
// @Summary Get an artifact by digest
// @Description Retrieve a single artifact by its image digest
// @Tags artifacts
// @Accept json
// @Produce json
// @Param digest path string true "Image digest (sha256:...)"
// @Success 200 {object} contracts.ArtifactResponse "Artifact details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid digest format"
// @Failure 404 {object} contracts.ErrorResponse "Artifact not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/artifacts/digest/{digest} [get]
func (h *ArtifactHandler) GetByDigest(c *gin.Context) {
	var param contracts.ArtifactDigestParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	art, err := h.service.GetByDigest(c.Request.Context(), param.Digest)
	if err != nil {
		if errors.Is(err, artifactService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(art))
}

// List handles GET /api/v1/artifacts
// @Summary List artifacts
// @Description List artifacts with optional filtering and pagination
// @Tags artifacts
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param build_id query string false "Filter by build ID"
// @Param image_name query string false "Filter by image name"
// @Param image_digest query string false "Filter by image digest"
// @Param tag query string false "Filter by tag"
// @Param repo query string false "Filter by repository"
// @Param provider query string false "Filter by provider"
// @Param signed_by query string false "Filter by signer"
// @Param scan_status query string false "Filter by scan status"
// @Param since query string false "Filter by creation date (RFC3339)"
// @Param until query string false "Filter by creation date (RFC3339)"
// @Param sort_by query string false "Sort field (created_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.ArtifactPageResult "Paginated list of artifacts"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/artifacts [get]
func (h *ArtifactHandler) List(c *gin.Context) {
	var filter contracts.ArtifactListFilter
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

	// Convert to service filter based on actual artifact model
	svcFilter := artifactService.ListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
	}

	if filter.BuildID != nil {
		id := uuid.MustParse(*filter.BuildID)
		svcFilter.BuildID = &id
	}

	// Map container-specific filters to generic artifact filters
	if filter.ImageName != nil {
		svcFilter.Name = filter.ImageName
	}
	if filter.ImageDigest != nil {
		svcFilter.Digest = filter.ImageDigest
	}

	// For container artifacts, always filter by kind=oci-image
	kind := "oci-image"
	svcFilter.Kind = &kind

	artifacts, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.ArtifactResponse, len(artifacts))
	for i, art := range artifacts {
		items[i] = *h.toResponse(&art)
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// Update handles PATCH /api/v1/artifacts/:id
// @Summary Update an artifact
// @Description Update an artifact's metadata, scan results, and signature information
// @Tags artifacts
// @Accept json
// @Produce json
// @Param id path string true "Artifact ID (UUID)"
// @Param artifact body contracts.ArtifactUpdate true "Artifact update request"
// @Success 200 {object} contracts.ArtifactResponse "Updated artifact"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Artifact not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/artifacts/{id} [patch]
func (h *ArtifactHandler) Update(c *gin.Context) {
	type pathParam struct {
		ArtifactID string `uri:"artifact_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.ArtifactUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Get existing artifact to preserve existing metadata
	id, err := h.ParseUUID(p.ArtifactID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	existing, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, artifactService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	// Build update request - only Labels and Metadata can be updated
	svcReq := artifactService.UpdateRequest{}

	// Update labels if needed
	labels := map[string]interface{}(existing.Labels)
	if labels == nil {
		labels = make(map[string]interface{})
	}
	if req.Tag != nil {
		labels["tag"] = *req.Tag
	}
	svcReq.Labels = labels

	// Update metadata if needed
	metadata := map[string]interface{}(existing.Metadata)
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if req.ScanStatus != nil {
		metadata["scan_status"] = *req.ScanStatus
	}
	if req.ScanResults != nil {
		metadata["scan_results"] = req.ScanResults
	}
	if req.SignedBy != nil {
		metadata["signed_by"] = *req.SignedBy
	}
	if req.SignedAt != nil {
		metadata["signed_at"] = req.SignedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	svcReq.Metadata = metadata

	art, err := h.service.Update(c.Request.Context(), id, svcReq)
	if err != nil {
		if errors.Is(err, artifactService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(art))
}

// Delete handles DELETE /api/v1/artifacts/:id
// @Summary Delete an artifact
// @Description Delete an artifact if it is not referenced by any releases
// @Tags artifacts
// @Accept json
// @Produce json
// @Param id path string true "Artifact ID (UUID)"
// @Success 204 "Artifact deleted successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid artifact ID"
// @Failure 404 {object} contracts.ErrorResponse "Artifact not found"
// @Failure 409 {object} contracts.ErrorResponse "Artifact is referenced by releases"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/artifacts/{id} [delete]
func (h *ArtifactHandler) Delete(c *gin.Context) {
	type pathParam struct {
		ArtifactID string `uri:"artifact_id" binding:"required,uuid4"`
	}
	var p pathParam
	if err := c.ShouldBindUri(&p); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	id, err := h.ParseUUID(p.ArtifactID)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, artifactService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		if errors.Is(err, artifactService.ErrArtifactInUse) {
			h.RespondWithConflict(c, "Artifact is referenced by releases and cannot be deleted")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// toResponse converts a generic artifact model to a container-specific response DTO
func (h *ArtifactHandler) toResponse(a *artifact.Artifact) *contracts.ArtifactResponse {
	resp := &contracts.ArtifactResponse{
		ID:        a.ID.String(),
		BuildID:   a.BuildID.String(),
		CreatedAt: a.CreatedAt,
	}

	// Extract container-specific fields from the generic model
	// ProjectID would need to be fetched from the build relationship
	// For now, using placeholder since the actual artifact doesn't have ProjectID
	resp.ProjectID = a.BuildID.String() // This should be resolved via Build->Project

	// Map generic fields to container-specific fields
	if a.Name != nil {
		resp.ImageName = *a.Name
	}
	if a.Digest != nil {
		resp.ImageDigest = *a.Digest
	}

	// Extract container fields from Labels
	if a.Labels != nil {
		if tag, ok := a.Labels["tag"].(string); ok {
			resp.Tag = &tag
		}
		if repo, ok := a.Labels["repo"].(string); ok {
			resp.Repo = &repo
		}
		if provider, ok := a.Labels["provider"].(string); ok {
			resp.Provider = &provider
		}
	}

	// Extract build and scan info from Metadata
	if a.Metadata != nil {
		if buildArgs, ok := a.Metadata["build_args"].(map[string]interface{}); ok {
			resp.BuildArgs = buildArgs
		}
		if buildMeta, ok := a.Metadata["build_meta"].(map[string]interface{}); ok {
			resp.BuildMeta = buildMeta
		}
		if scanStatus, ok := a.Metadata["scan_status"].(string); ok {
			resp.ScanStatus = &scanStatus
		}
		if scanResults, ok := a.Metadata["scan_results"].(map[string]interface{}); ok {
			resp.ScanResults = scanResults
		}
		if signedBy, ok := a.Metadata["signed_by"].(string); ok {
			resp.SignedBy = &signedBy
		}
		// Note: SignedAt would need to be parsed from string if stored
	}

	// UpdatedAt doesn't exist in the actual artifact model
	resp.UpdatedAt = a.CreatedAt // Using CreatedAt as fallback

	return resp
}
