package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	contracts "github.com/input-output-hk/catalyst-forge/foundry/api/internal/contracts"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/enums"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/trace"
	traceService "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/trace"
)

// TraceHandler handles trace-related endpoints
type TraceHandler struct {
	*BaseHandler
	service traceService.Service
}

// NewTraceHandler creates a new trace handler
func NewTraceHandler(service traceService.Service, logger *slog.Logger) *TraceHandler {
	return &TraceHandler{
		BaseHandler: NewBaseHandler(logger),
		service:     service,
	}
}

// Create handles POST /api/v1/traces
// @Summary Create a new trace
// @Description Create a new trace for tracking build operations
// @Tags traces
// @Accept json
// @Produce json
// @Param trace body contracts.TraceCreate true "Trace creation request"
// @Success 201 {object} contracts.TraceResponse "Created trace"
// @Failure 400 {object} contracts.ErrorResponse "Invalid request body"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/traces [post]
func (h *TraceHandler) Create(c *gin.Context) {
	var req contracts.TraceCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	// Convert to service request
	svcReq := traceService.CreateRequest{
		Purpose:        enums.TracePurpose(req.Purpose),
		RetentionClass: enums.RetentionClass(req.RetentionClass),
		Branch:         req.Branch,
		CreatedBy:      req.CreatedBy,
	}

	if req.RepoID != nil {
		id := uuid.MustParse(*req.RepoID)
		svcReq.RepoID = &id
	}

	// Create trace
	tr, err := h.service.Create(c.Request.Context(), svcReq)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusCreated, h.toResponse(tr))
}

// GetByID handles GET /api/v1/traces/:id
// @Summary Get a trace by ID
// @Description Retrieve a single trace by its ID
// @Tags traces
// @Accept json
// @Produce json
// @Param id path string true "Trace ID (UUID)"
// @Success 200 {object} contracts.TraceResponse "Trace details"
// @Failure 400 {object} contracts.ErrorResponse "Invalid trace ID"
// @Failure 404 {object} contracts.ErrorResponse "Trace not found"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/traces/{id} [get]
func (h *TraceHandler) GetByID(c *gin.Context) {
	var param contracts.TraceIDParam
	if err := c.ShouldBindUri(&param); err != nil {
		h.RespondWithValidationError(c, err)
		return
	}

	tr, err := h.service.GetByID(c.Request.Context(), param.TraceID)
	if err != nil {
		if errors.Is(err, traceService.ErrTraceNotFound) {
			h.RespondWithNotFound(c, "Trace")
			return
		}
		h.RespondWithInternalError(c, err)
		return
	}

	h.RespondWithSuccess(c, http.StatusOK, h.toResponse(tr))
}

// List handles GET /api/v1/traces
// @Summary List traces
// @Description List traces with optional filtering and pagination
// @Tags traces
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param repo_id query string false "Filter by repository ID"
// @Param purpose query string false "Filter by purpose (build, test, deploy)"
// @Param retention_class query string false "Filter by retention class (temp, short, long)"
// @Param branch query string false "Filter by branch"
// @Param created_by query string false "Filter by creator"
// @Param since query string false "Filter by creation date (RFC3339)"
// @Param until query string false "Filter by creation date (RFC3339)"
// @Param sort_by query string false "Sort field (created_at)"
// @Param sort_order query string false "Sort order (asc, desc)"
// @Success 200 {object} contracts.TracePageResult "Paginated list of traces"
// @Failure 400 {object} contracts.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} contracts.ErrorResponse "Internal server error"
// @Router /api/v1/traces [get]
func (h *TraceHandler) List(c *gin.Context) {
	var filter contracts.TraceListFilter
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
	svcFilter := traceService.ListFilter{
		Pagination: h.GetPagination(c),
		Sort:       h.GetSort(c),
		Branch:     filter.Branch,
		CreatedBy:  filter.CreatedBy,
		Since:      filter.Since,
		Until:      filter.Until,
	}

	if filter.RepoID != nil {
		id := uuid.MustParse(*filter.RepoID)
		svcFilter.RepoID = &id
	}

	if filter.Purpose != nil {
		purpose := enums.TracePurpose(*filter.Purpose)
		svcFilter.Purpose = &purpose
	}

	if filter.RetentionClass != nil {
		retClass := enums.RetentionClass(*filter.RetentionClass)
		svcFilter.RetentionClass = &retClass
	}

	traces, total, err := h.service.List(c.Request.Context(), svcFilter)
	if err != nil {
		h.RespondWithInternalError(c, err)
		return
	}

	// Convert to response
	items := make([]contracts.TraceResponse, len(traces))
	for i, tr := range traces {
		items[i] = *h.toResponse(&tr)
	}

	result := contracts.NewPageResult(items, filter.Page, filter.PageSize, total)
	h.RespondWithSuccess(c, http.StatusOK, result)
}

// toResponse converts a trace model to response DTO
func (h *TraceHandler) toResponse(t *trace.Trace) *contracts.TraceResponse {
	resp := &contracts.TraceResponse{
		ID:             t.ID.String(),
		Purpose:        string(t.Purpose),
		RetentionClass: string(t.RetentionClass),
		Branch:         t.Branch,
		CreatedBy:      t.CreatedBy,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.CreatedAt, // Trace model doesn't have UpdatedAt
	}

	if t.RepoID != nil {
		repoStr := t.RepoID.String()
		resp.RepoID = &repoStr
	}

	return resp
}
