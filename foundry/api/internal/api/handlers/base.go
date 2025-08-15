package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	
	contracts "github.com/input-output-hk/catalyst-forge/foundry/api/internal/contracts"
	base "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository"
)

// BaseHandler provides common utilities for all handlers
type BaseHandler struct {
	logger *slog.Logger
}

// NewBaseHandler creates a new base handler
func NewBaseHandler(logger *slog.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

// ParseUUID parses and validates a UUID from a string
func (h *BaseHandler) ParseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}

// GetPagination extracts pagination parameters from gin context
func (h *BaseHandler) GetPagination(c *gin.Context) *base.Pagination {
	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 && val <= 100 {
			pageSize = val
		}
	}

	return &base.Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// GetSort extracts sort parameters from gin context
func (h *BaseHandler) GetSort(c *gin.Context) *base.Sort {
	field := c.Query("sort_field")
	order := c.Query("sort_order")

	if field == "" {
		field = "created_at"
	}

	var sortOrder base.SortOrder
	if order == "asc" {
		sortOrder = base.SortAsc
	} else {
		sortOrder = base.SortDesc
	}

	return &base.Sort{
		Field: field,
		Order: sortOrder,
	}
}

// RespondWithError sends an error response
func (h *BaseHandler) RespondWithError(c *gin.Context, statusCode int, code, message string, details interface{}) {
	c.JSON(statusCode, contracts.NewErrorResponse(code, message, details))
}

// RespondWithValidationError sends a validation error response
func (h *BaseHandler) RespondWithValidationError(c *gin.Context, err error) {
	h.RespondWithError(c, http.StatusBadRequest, contracts.ErrCodeBadRequest, "Validation failed", err.Error())
}

// RespondWithNotFound sends a not found error response
func (h *BaseHandler) RespondWithNotFound(c *gin.Context, resource string) {
	h.RespondWithError(c, http.StatusNotFound, contracts.ErrCodeNotFound, resource+" not found", nil)
}

// RespondWithInternalError sends an internal server error response
func (h *BaseHandler) RespondWithInternalError(c *gin.Context, err error) {
	h.logger.Error("Internal server error", "error", err)
	h.RespondWithError(c, http.StatusInternalServerError, contracts.ErrCodeInternalError, "An internal error occurred", nil)
}

// RespondWithConflict sends a conflict error response
func (h *BaseHandler) RespondWithConflict(c *gin.Context, message string) {
	h.RespondWithError(c, http.StatusConflict, contracts.ErrCodeConflict, message, nil)
}

// RespondWithUnprocessableEntity sends an unprocessable entity error response
func (h *BaseHandler) RespondWithUnprocessableEntity(c *gin.Context, message string, details interface{}) {
	h.RespondWithError(c, http.StatusUnprocessableEntity, contracts.ErrCodeUnprocessableEntity, message, details)
}

// RespondWithSuccess sends a success response
func (h *BaseHandler) RespondWithSuccess(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, data)
}

// RespondWithPagination sends a paginated response
func (h *BaseHandler) RespondWithPagination(c *gin.Context, items interface{}, page, pageSize int, total int64) {
	c.JSON(http.StatusOK, gin.H{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}