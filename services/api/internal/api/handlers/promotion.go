package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	contracts "github.com/input-output-hk/catalyst-forge/services/api/internal/contracts"
	depModel "github.com/input-output-hk/catalyst-forge/services/api/internal/models/deployment"
	depRepo "github.com/input-output-hk/catalyst-forge/services/api/internal/repository/deployment"
	depService "github.com/input-output-hk/catalyst-forge/services/api/internal/service/deployment"
)

// PromotionHandler handles promotion-related endpoints
type PromotionHandler struct {
	*BaseHandler
	service depService.PromotionService
}

// NewPromotionHandler creates a new promotion handler
func NewPromotionHandler(service depService.PromotionService, logger *slog.Logger) *PromotionHandler {
	return &PromotionHandler{BaseHandler: NewBaseHandler(logger), service: service}
}

// Create handles POST /api/v1/promotions
// @Summary Create a promotion
// @Description Create a new promotion request
// @Tags promotions
// @Accept json
// @Produce json
// @Param promotion body contracts.PromotionCreate true "Promotion creation request"
// @Success 201 {object} contracts.PromotionResponse "Created promotion"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 404 {object} contracts.ErrorResponse "Referenced project, release, or environment not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/promotions [post]
func (h *PromotionHandler) Create(c *gin.Context) {
	var req contracts.PromotionCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	svcReq := depService.CreatePromotionRequest{
		ProjectID:     uuid.MustParse(req.ProjectID),
		ReleaseID:     uuid.MustParse(req.ReleaseID),
		EnvironmentID: uuid.MustParse(req.EnvironmentID),
		ApprovalMode:  depModel.ApprovalMode(req.ApprovalMode),
		RequestedBy:   req.RequestedBy,
		Reason:        req.Reason,
		PolicyResults: req.PolicyResults,
	}

	p, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		// We don't have specialized errors from service beyond not-founds
		h.RespondWithInternalError(c, err)
		return
	}
	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(p))
}

// GetByID handles GET /api/v1/promotions/:promotion_id
// @Summary Get a promotion by ID
// @Description Retrieve a single promotion by its ID
// @Tags promotions
// @Accept json
// @Produce json
// @Param promotion_id path string true "Promotion ID (UUID)"
// @Success 200 {object} contracts.PromotionResponse "Promotion details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid promotion ID"
// @Failure 404 {object} contracts.ErrorResponse "Promotion not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/promotions/{promotion_id} [get]
func (h *PromotionHandler) GetByID(c *gin.Context) {
	idStr := c.Param("promotion_id")
	id, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	pr, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, depRepo.ErrPromotionNotFound) {
			h.RespondWithNotFound(c, "Promotion")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}
	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(pr))
}

// List handles GET /api/v1/promotions
// @Summary List promotions
// @Description List promotions with optional filtering and pagination
// @Tags promotions
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param project_id query string false "Filter by project ID"
// @Param environment_id query string false "Filter by environment ID"
// @Param release_id query string false "Filter by release ID"
// @Param status query string false "Filter by status"
// @Param since query string false "Filter by creation date (RFC3339)"
// @Param until query string false "Filter by creation date (RFC3339)"
// @Param sort_by query string false "Sort field (created_at, updated_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.PromotionPageResult "Paginated list of promotions"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/promotions [get]
func (h *PromotionHandler) List(c *gin.Context) {
	var filter contracts.PromotionListFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	// Defaults handled by GetPagination/GetSort
	svcFilter := depService.PromotionListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
	}
	if filter.ProjectID != nil {
		id := uuid.MustParse(*filter.ProjectID)
		svcFilter.ProjectID = &id
	}
	if filter.EnvironmentID != nil {
		id := uuid.MustParse(*filter.EnvironmentID)
		svcFilter.EnvID = &id
	}
	if filter.ReleaseID != nil {
		id := uuid.MustParse(*filter.ReleaseID)
		svcFilter.ReleaseID = &id
	}
	if filter.Status != nil {
		st := depModel.PromotionStatus(*filter.Status)
		svcFilter.Status = &st
	}

	items, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	respItems := make([]contracts.PromotionResponse, len(items))
	for i := range items {
		respItems[i] = *h.toResponse(&items[i])
	}
	result := contracts.NewPageResult(respItems, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// Update handles PATCH /api/v1/promotions/:promotion_id
// @Summary Update a promotion
// @Description Update a promotion's status and metadata
// @Tags promotions
// @Accept json
// @Produce json
// @Param promotion_id path string true "Promotion ID (UUID)"
// @Param promotion body contracts.PromotionUpdate true "Promotion update request"
// @Success 200 {object} contracts.PromotionResponse "Updated promotion"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request"
// @Failure 404 {object} contracts.ErrorResponse "Promotion not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/promotions/{promotion_id} [patch]
func (h *PromotionHandler) Update(c *gin.Context) {
	idStr := c.Param("promotion_id")
	id, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	var req contracts.PromotionUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	svcReq := depService.UpdatePromotionRequest{}
	if req.Status != nil {
		st := depModel.PromotionStatus(*req.Status)
		svcReq.Status = &st
	}
	svcReq.Reason = req.Reason
	svcReq.ApproverID = req.ApproverID
	svcReq.ApprovedAt = req.ApprovedAt
	svcReq.StepUpVerifiedAt = req.StepUpVerifiedAt
	svcReq.PolicyResults = req.PolicyResults
	if req.DeploymentID != nil {
		pid := uuid.MustParse(*req.DeploymentID)
		svcReq.DeploymentID = &pid
	}
	if req.TraceID != nil {
		tid := uuid.MustParse(*req.TraceID)
		svcReq.TraceID = &tid
	}

	pr, err := h.service.Update(c.Request.Context(), id, svcReq)
	if err != nil {
		if errors.Is(err, depRepo.ErrPromotionNotFound) {
			h.RespondWithNotFound(c, "Promotion")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}
	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(pr))
}

// Delete handles DELETE /api/v1/promotions/:promotion_id
// @Summary Delete a promotion
// @Description Delete a promotion by ID
// @Tags promotions
// @Accept json
// @Produce json
// @Param promotion_id path string true "Promotion ID (UUID)"
// @Success 204 "Promotion deleted successfully"
// @Failure 400 {object} contracts.ErrorResponse "Invalid promotion ID"
// @Failure 404 {object} contracts.ErrorResponse "Promotion not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/promotions/{promotion_id} [delete]
func (h *PromotionHandler) Delete(c *gin.Context) {
	idStr := c.Param("promotion_id")
	id, err := h.ParseUUID(idStr)
	if err != nil {
		h.RespondWithValidationError(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		if errors.Is(err, depRepo.ErrPromotionNotFound) {
			h.RespondWithNotFound(c, "Promotion")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *PromotionHandler) toResponse(p *depModel.Promotion) *contracts.PromotionResponse {
	resp := &contracts.PromotionResponse{
		ID:               p.ID.String(),
		ProjectID:        p.ProjectID.String(),
		ReleaseID:        p.ReleaseID.String(),
		EnvironmentID:    p.EnvID.String(),
		Status:           string(p.Status),
		ApprovalMode:     string(p.ApprovalMode),
		RequestedBy:      p.RequestedBy,
		RequestedAt:      p.RequestedAt,
		Reason:           p.Reason,
		ApproverID:       p.ApproverID,
		ApprovedAt:       p.ApprovedAt,
		StepUpVerifiedAt: p.StepUpVerifiedAt,
		DeploymentID:     nil,
		TraceID:          nil,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
	if p.DeploymentID != nil {
		id := p.DeploymentID.String()
		resp.DeploymentID = &id
	}
	if p.TraceID != nil {
		id := p.TraceID.String()
		resp.TraceID = &id
	}
	if p.PolicyResults != nil {
		resp.PolicyResults = map[string]interface{}(p.PolicyResults)
	}
	return resp
}
