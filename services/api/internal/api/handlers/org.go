package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/input-output-hk/catalyst-forge/services/api/internal/models/org"
	orgsvc "github.com/input-output-hk/catalyst-forge/services/api/internal/service/org"
)

type OrgHandler struct{ S orgsvc.Service }

func NewOrgHandler(s orgsvc.Service) *OrgHandler { return &OrgHandler{S: s} }

// List handles GET /api/v1/admin/orgs
// @Summary List organizations
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/admin/orgs [get]
func (h *OrgHandler) List(c *gin.Context) {
	rows, total, err := h.S.List(c.Request.Context(), nil, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": rows, "total": total})
}

// Get handles GET /api/v1/admin/orgs/:id
// @Summary Get organization
// @Tags admin
// @Produce json
// @Param id path string true "Organization ID"
// @Success 200 {object} org.Organization
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/admin/orgs/{id} [get]
func (h *OrgHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	row, err := h.S.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.JSON(http.StatusOK, row)
}

type orgCreateRequest struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

// Create handles POST /api/v1/admin/orgs
// @Summary Create organization
// @Tags admin
// @Accept json
// @Produce json
// @Success 201 {object} org.Organization
// @Router /api/v1/admin/orgs [post]
func (h *OrgHandler) Create(c *gin.Context) {
	var req orgCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if _, err := h.S.Create(c.Request.Context(), req.Name, req.IsDefault); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	c.JSON(http.StatusCreated, &org.Organization{Name: req.Name, IsDefault: req.IsDefault})
}

type orgUpdateNameRequest struct {
	Name string `json:"name"`
}

// UpdateName handles PATCH /api/v1/admin/orgs/:id/name
// @Summary Rename organization
// @Tags admin
// @Accept json
// @Success 204
// @Router /api/v1/admin/orgs/{id}/name [patch]
func (h *OrgHandler) UpdateName(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	var req orgUpdateNameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := h.S.UpdateName(c.Request.Context(), id, req.Name); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusNoContent)
}

// SetDefault handles PATCH /api/v1/admin/orgs/:id/default
// @Summary Set default organization
// @Tags admin
// @Success 204
// @Router /api/v1/admin/orgs/{id}/default [patch]
func (h *OrgHandler) SetDefault(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if err := h.S.SetDefault(c.Request.Context(), id); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	c.Status(http.StatusNoContent)
}
