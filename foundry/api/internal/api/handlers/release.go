package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	contracts "github.com/input-output-hk/catalyst-forge/foundry/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/enums"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/release"
	releaseRepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/release"
	releaseService "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/release"
)

// ReleaseHandler handles release-related endpoints
type ReleaseHandler struct {
	*BaseHandler
	service releaseService.Service
}

// NewReleaseHandler creates a new release handler
func NewReleaseHandler(service releaseService.Service, logger *slog.Logger) *ReleaseHandler {
	return &ReleaseHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// Create handles POST /api/v1/releases
// @Summary Create a new release
// @Description Create a new release with modules, injections, and artifacts
// @Tags releases
// @Accept json
// @Produce json
// @Param release body contracts.ReleaseCreate true "Release creation request"
// @Success 201 {object} contracts.ReleaseResponse "Created release"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 404 {object} contracts.ErrorResponse "Project or artifact not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases [post]
func (h *ReleaseHandler) Create(c *gin.Context) {
	var req contracts.ReleaseCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	svcReq := releaseService.CreateRequest{
		ProjectID:      uuid.MustParse(req.ProjectID),
		ReleaseKey:     req.ReleaseKey,
		SourceCommit:   req.SourceCommit,
		SourceBranch:   req.SourceBranch,
		Tag:            req.Tag,
		OCIRef:         req.OCIRef,
		OCIDigest:      req.OCIDigest,
		ValuesHash:     req.ValuesHash,
		ValuesSnapshot: req.ValuesSnapshot,
		ContentHash:    req.ContentHash,
		CreatedBy:      req.CreatedBy,
	}

	if req.TraceID != nil {
		id := uuid.MustParse(*req.TraceID)
		svcReq.TraceID = &id
	}

	if req.Status != nil {
		status := enums.ReleaseStatus(*req.Status)
		svcReq.Status = &status
	}

	// Handle modules
	for _, m := range req.Modules {
		module := releaseService.ModuleRequest{
			ModuleKey:  m.ModuleKey,
			Name:       m.Name,
			ModuleType: m.ModuleType,
			Version:    m.Version,
			Registry:   m.Registry,
			OCIRef:     m.OCIRef,
			OCIDigest:  m.OCIDigest,
			GitURL:     m.GitURL,
			GitRef:     m.GitRef,
			Path:       m.Path,
		}
		svcReq.Modules = append(svcReq.Modules, module)
	}

	// Handle injections
	for _, i := range req.Injections {
		injection := releaseService.InjectionRequest{
			JSONPointer:   i.JSONPointer,
			ArtifactKey:   i.ArtifactKey,
			ArtifactField: i.ArtifactField,
			ModuleKey:     i.ModuleKey,
			ModuleName:    i.ModuleName,
		}
		svcReq.Injections = append(svcReq.Injections, injection)
	}

	// Handle artifacts
	for _, a := range req.Artifacts {
		artifactID := uuid.MustParse(a.ArtifactID)
		artifact := releaseService.ArtifactLinkRequest{
			ArtifactID:  artifactID,
			Role:        a.Role,
			ArtifactKey: a.ArtifactKey,
		}
		svcReq.Artifacts = append(svcReq.Artifacts, artifact)
	}

	// Create release
	rel, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		if errors.Is(err, releaseService.ErrProjectNotFound) {
			h.RespondWithNotFound(c, "Project")
			return
		}
		if errors.Is(err, releaseService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(rel))
}

// GetByID handles GET /api/v1/releases/:id
// @Summary Get a release by ID
// @Description Retrieve a single release by its ID
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Success 200 {object} contracts.ReleaseResponse "Release details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid release ID"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id} [get]
func (h *ReleaseHandler) GetByID(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	rel, err := h.service.GetByID(c.Request.Context(), param.ReleaseID)
	if err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(rel))
}

// List handles GET /api/v1/releases
// @Summary List releases
// @Description List releases with optional filtering and pagination
// @Tags releases
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param project_id query string false "Filter by project ID"
// @Param release_key query string false "Filter by release key"
// @Param status query string false "Filter by status (pending, building, sealed, failed)"
// @Param oci_digest query string false "Filter by OCI digest"
// @Param tag query string false "Filter by tag"
// @Param created_by query string false "Filter by creator"
// @Param since query string false "Filter by creation date (RFC3339)"
// @Param until query string false "Filter by creation date (RFC3339)"
// @Param sort_by query string false "Sort field (created_at, updated_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.ReleasePageResult "Paginated list of releases"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases [get]
func (h *ReleaseHandler) List(c *gin.Context) {
	var filter contracts.ReleaseListFilter
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
	svcFilter := releaseService.ListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
	}

	if filter.ProjectID != nil {
		id := uuid.MustParse(*filter.ProjectID)
		svcFilter.ProjectID = &id
	}
	if filter.ReleaseKey != nil {
		svcFilter.ReleaseKey = filter.ReleaseKey
	}
	if filter.Status != nil {
		status := enums.ReleaseStatus(*filter.Status)
		svcFilter.Status = &status
	}
	if filter.OCIDigest != nil {
		svcFilter.OCIDigest = filter.OCIDigest
	}
	if filter.Tag != nil {
		svcFilter.Tag = filter.Tag
	}
	if filter.CreatedBy != nil {
		svcFilter.CreatedBy = filter.CreatedBy
	}
	if filter.Since != nil {
		svcFilter.Since = filter.Since
	}
	if filter.Until != nil {
		svcFilter.Until = filter.Until
	}

	releases, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.ReleaseResponse, len(releases))
	for i, rel := range releases {
		items[i] = *h.toResponse(&rel)
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// Update handles PATCH /api/v1/releases/:id
// @Summary Update a release
// @Description Update a release's status and signature information
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Param release body contracts.ReleaseUpdate true "Release update request"
// @Success 200 {object} contracts.ReleaseResponse "Updated release"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed and cannot be modified"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id} [patch]
func (h *ReleaseHandler) Update(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.ReleaseUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	svcReq := releaseService.UpdateRequest{
		Status:              (*enums.ReleaseStatus)(req.Status),
		OCIRef:              req.OCIRef,
		OCIDigest:           req.OCIDigest,
		Signed:              req.Signed,
		SigIssuer:           req.SigIssuer,
		SigSubject:          req.SigSubject,
		SignatureVerifiedAt: req.SignatureVerifiedAt,
	}

	rel, err := h.service.Update(c.Request.Context(), param.ReleaseID, svcReq)
	if err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(rel))
}

// Delete handles DELETE /api/v1/releases/:id
// @Summary Delete a release
// @Description Delete a release if it has no deployments
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Success 204 "Release deleted successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid release ID"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed or has deployments"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id} [delete]
func (h *ReleaseHandler) Delete(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), param.ReleaseID); err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be deleted")
			return
		}
		// TODO: Check if release has deployments
		if false {
			h.RespondWithConflict(c, "Release has deployments and cannot be deleted")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetModules handles GET /api/v1/releases/:id/modules
// @Summary Get release modules
// @Description List all modules associated with a release
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Success 200 {array} contracts.ReleaseModule "List of release modules"
// @Failure 400 {object} contracts.ErrorResponse "Invalid release ID"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id}/modules [get]
func (h *ReleaseHandler) GetModules(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	modules, err := h.service.ListModules(c.Request.Context(), param.ReleaseID)
	if err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.ReleaseModule, len(modules))
	for i, m := range modules {
		items[i] = h.toModuleResponse(&m)
	}

	h.RespondWithSuccess(c, http.StatusOK, items)
}

// AddModules handles POST /api/v1/releases/:id/modules
// @Summary Add modules to a release
// @Description Add one or more modules to an existing release
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Param modules body contracts.ReleaseModuleCreate true "Modules to add"
// @Success 201 "Modules added successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed and cannot be modified"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id}/modules [post]
func (h *ReleaseHandler) AddModules(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.ReleaseModuleCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	var modules []releaseService.ModuleRequest
	for _, m := range req.Modules {
		module := releaseService.ModuleRequest{
			ModuleKey:  m.ModuleKey,
			Name:       m.Name,
			ModuleType: m.ModuleType,
			Version:    m.Version,
			Registry:   m.Registry,
			OCIRef:     m.OCIRef,
			OCIDigest:  m.OCIDigest,
			GitURL:     m.GitURL,
			GitRef:     m.GitRef,
			Path:       m.Path,
		}
		modules = append(modules, module)
	}

	if err := h.service.CreateModules(c.Request.Context(), param.ReleaseID, modules); err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

// RemoveModule handles DELETE /api/v1/releases/:release_id/modules/:module_key
// @Summary Remove a module from a release
// @Description Remove a specific module from a release by module key
// @Tags releases
// @Accept json
// @Produce json
// @Param release_id path string true "Release ID (UUID)"
// @Param module_key path string true "Module key"
// @Success 204 "Module removed successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid parameters"
// @Failure 404 {object} contracts.ErrorResponse "Release or module not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed and cannot be modified"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{release_id}/modules/{module_key} [delete]
func (h *ReleaseHandler) RemoveModule(c *gin.Context) {
	var param contracts.ReleaseModuleKeyParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.DeleteModule(c.Request.Context(), param.ReleaseID, param.ModuleKey); err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseRepo.ErrModuleNotFound) {
			h.RespondWithNotFound(c, "Module")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetInjections handles GET /api/v1/releases/:id/injections
// @Summary Get release injections
// @Description List all injections associated with a release
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Success 200 {array} contracts.ReleaseInjection "List of release injections"
// @Failure 400 {object} contracts.ErrorResponse "Invalid release ID"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id}/injections [get]
func (h *ReleaseHandler) GetInjections(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	injections, err := h.service.ListInjections(c.Request.Context(), param.ReleaseID)
	if err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.ReleaseInjection, len(injections))
	for i, inj := range injections {
		items[i] = h.toInjectionResponse(&inj)
	}

	h.RespondWithSuccess(c, http.StatusOK, items)
}

// AddInjections handles POST /api/v1/releases/:id/injections
// @Summary Add injections to a release
// @Description Add one or more injections to an existing release
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Param injections body contracts.ReleaseInjectionCreate true "Injections to add"
// @Success 201 "Injections added successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed and cannot be modified"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id}/injections [post]
func (h *ReleaseHandler) AddInjections(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.ReleaseInjectionCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	var injections []releaseService.InjectionRequest
	for _, i := range req.Injections {
		injection := releaseService.InjectionRequest{
			JSONPointer:   i.JSONPointer,
			ArtifactKey:   i.ArtifactKey,
			ArtifactField: i.ArtifactField,
			ModuleKey:     i.ModuleKey,
			ModuleName:    i.ModuleName,
		}
		injections = append(injections, injection)
	}

	if err := h.service.CreateInjections(c.Request.Context(), param.ReleaseID, injections); err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

// RemoveInjection handles DELETE /api/v1/releases/:release_id/injections/:injection_id
// @Summary Remove an injection from a release
// @Description Remove a specific injection from a release
// @Tags releases
// @Accept json
// @Produce json
// @Param release_id path string true "Release ID (UUID)"
// @Param injection_id path string true "Injection ID (UUID)"
// @Success 204 "Injection removed successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid parameters"
// @Failure 404 {object} contracts.ErrorResponse "Release or injection not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed and cannot be modified"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{release_id}/injections/{injection_id} [delete]
func (h *ReleaseHandler) RemoveInjection(c *gin.Context) {
	var param contracts.ReleaseInjectionIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.DeleteInjection(c.Request.Context(), param.ReleaseID, param.InjectionID); err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseRepo.ErrInjectionNotFound) {
			h.RespondWithNotFound(c, "Injection")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetArtifacts handles GET /api/v1/releases/:id/artifacts
// @Summary Get release artifacts
// @Description List all artifacts associated with a release
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Success 200 {array} contracts.ReleaseArtifactResponse "List of release artifacts"
// @Failure 400 {object} contracts.ErrorResponse "Invalid release ID"
// @Failure 404 {object} contracts.ErrorResponse "Release not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id}/artifacts [get]
func (h *ReleaseHandler) GetArtifacts(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	artifacts, err := h.service.ListArtifacts(c.Request.Context(), param.ReleaseID)
	if err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.ReleaseArtifactResponse, len(artifacts))
	for i, a := range artifacts {
		items[i] = h.toArtifactResponse(&a)
	}

	h.RespondWithSuccess(c, http.StatusOK, items)
}

// AttachArtifact handles POST /api/v1/releases/:id/artifacts
// @Summary Attach an artifact to a release
// @Description Attach an existing artifact to a release with a specific role
// @Tags releases
// @Accept json
// @Produce json
// @Param id path string true "Release ID (UUID)"
// @Param artifact body contracts.ReleaseArtifactCreate true "Artifact attachment request"
// @Success 201 "Artifact attached successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Release or artifact not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed and cannot be modified"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{id}/artifacts [post]
func (h *ReleaseHandler) AttachArtifact(c *gin.Context) {
	var param contracts.ReleaseIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	var req contracts.ReleaseArtifactCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	artifactID := uuid.MustParse(req.ArtifactID)
	linkReq := releaseService.ArtifactLinkRequest{
		ArtifactID:  artifactID,
		Role:        req.Role,
		ArtifactKey: req.ArtifactKey,
	}
	if err := h.service.AttachArtifact(c.Request.Context(), param.ReleaseID, linkReq); err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusCreated)
}

// DetachArtifact handles DELETE /api/v1/releases/:release_id/artifacts/:artifact_id
// @Summary Detach an artifact from a release
// @Description Detach a specific artifact from a release
// @Tags releases
// @Accept json
// @Produce json
// @Param release_id path string true "Release ID (UUID)"
// @Param artifact_id path string true "Artifact ID (UUID)"
// @Param role query string false "Artifact role (optional)"
// @Success 204 "Artifact detached successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid parameters"
// @Failure 404 {object} contracts.ErrorResponse "Release or artifact not found"
// @Failure 409 {object} contracts.ErrorResponse "Release is sealed and cannot be modified"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/releases/{release_id}/artifacts/{artifact_id} [delete]
func (h *ReleaseHandler) DetachArtifact(c *gin.Context) {
	var param contracts.ReleaseArtifactIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	if err := h.service.DetachArtifact(c.Request.Context(), param.ReleaseID, param.ArtifactID, param.Role); err != nil {
		if errors.Is(err, releaseRepo.ErrReleaseNotFound) {
			h.RespondWithNotFound(c, "Release")
			return
		}
		if errors.Is(err, releaseService.ErrArtifactNotFound) {
			h.RespondWithNotFound(c, "Artifact")
			return
		}
		if errors.Is(err, releaseService.ErrReleaseSealed) {
			h.RespondWithConflict(c, "Release is sealed and cannot be modified")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// toResponse converts a release model to response DTO
func (h *ReleaseHandler) toResponse(r *release.Release) *contracts.ReleaseResponse {
	resp := &contracts.ReleaseResponse{
		ID:                  r.ID.String(),
		ProjectID:           r.ProjectID.String(),
		ReleaseKey:          r.ReleaseKey,
		SourceCommit:        r.SourceCommit,
		SourceBranch:        r.SourceBranch,
		Tag:                 r.Tag,
		Status:              string(r.Status),
		OCIRef:              r.OCIRef,
		OCIDigest:           r.OCIDigest,
		Signed:              r.Signed,
		SigIssuer:           r.SigIssuer,
		SigSubject:          r.SigSubject,
		SignatureVerifiedAt: r.SignatureVerifiedAt,
		ValuesHash:          r.ValuesHash,
		ContentHash:         r.ContentHash,
		CreatedBy:           r.CreatedBy,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
	}

	if r.TraceID != nil {
		traceStr := r.TraceID.String()
		resp.TraceID = &traceStr
	}

	if r.ValuesSnapshot != nil {
		resp.ValuesSnapshot = map[string]interface{}(r.ValuesSnapshot)
	}

	return resp
}

// toModuleResponse converts a release module model to response DTO
func (h *ReleaseHandler) toModuleResponse(m *release.ReleaseModule) contracts.ReleaseModule {
	resp := contracts.ReleaseModule{
		ID:         m.ID.String(),
		ReleaseID:  m.ReleaseID.String(),
		ModuleKey:  m.ModuleKey,
		Name:       m.Name,
		ModuleType: string(m.ModuleType),
		Version:    m.Version,
		Registry:   m.Registry,
		OCIRef:     m.OCIRef,
		OCIDigest:  m.OCIDigest,
		GitURL:     m.GitURL,
		GitRef:     m.GitRef,
		Path:       m.Path,
		CreatedAt:  &m.CreatedAt,
	}

	return resp
}

// toInjectionResponse converts a release injection model to response DTO
func (h *ReleaseHandler) toInjectionResponse(i *release.ReleaseInjection) contracts.ReleaseInjection {
	resp := contracts.ReleaseInjection{
		ID:            i.ID.String(),
		ReleaseID:     i.ReleaseID.String(),
		JSONPointer:   i.JSONPointer,
		ArtifactKey:   i.ArtifactKey,
		ArtifactField: string(i.ArtifactField),
		ModuleKey:     i.ModuleKey,
		ModuleName:    i.ModuleName,
		CreatedAt:     &i.CreatedAt,
	}

	return resp
}

// toArtifactResponse converts a release artifact model to response DTO
func (h *ReleaseHandler) toArtifactResponse(a *release.ReleaseArtifact) contracts.ReleaseArtifactResponse {
	resp := contracts.ReleaseArtifactResponse{
		ReleaseID:   a.ReleaseID.String(),
		ArtifactID:  a.ArtifactID.String(),
		Role:        a.Role,
		ArtifactKey: a.ArtifactKey,
		CreatedAt:   a.CreatedAt,
	}

	return resp
}