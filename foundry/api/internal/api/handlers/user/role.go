package user

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
)

// RoleHandler handles role endpoints
type RoleHandler struct {
	roleService userservice.RoleService
	logger      *slog.Logger
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(roleService userservice.RoleService, logger *slog.Logger) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
		logger:      logger,
	}
}

// Role represents a role in the system (swagger-compatible version)
// @Description Role represents a role in the system
type Role struct {
	ID          uint      `json:"id" example:"1"`
	Name        string    `json:"name" example:"admin"`
	Permissions []string  `json:"permissions" example:"user:read,user:write"`
	CreatedAt   time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

// CreateRoleRequest represents the request body for creating a role
type CreateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest represents the request body for updating a role
type UpdateRoleRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions" binding:"required"`
}

// CreateRole handles the POST /auth/roles endpoint
// @Summary Create a new role
// @Description Create a new role with the provided information
// @Tags roles
// @Accept json
// @Produce json
// @Param request body CreateRoleRequest true "Role creation request"
// @Param admin query bool false "If true, ignore permissions and add all permissions"
// @Success 201 {object} Role "Role created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 409 {object} map[string]interface{} "Role already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	role := &user.Role{
		Name: req.Name,
	}

	adminParam := c.Query("admin")
	if adminParam == "true" {
		role.SetPermissions(auth.AllPermissions)
	} else {
		if len(req.Permissions) == 0 {
			h.logger.Warn("No permissions provided for role", "name", req.Name)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Permissions are required when not creating an admin role",
			})
			return
		}
		role.SetPermissions(convertToPermissions(req.Permissions))
	}

	if err := h.roleService.CreateRole(role); err != nil {
		h.logger.Error("Failed to create role", "error", err, "name", req.Name)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, role)
}

// GetRole handles the GET /auth/roles/:id endpoint
// @Summary Get a role by ID
// @Description Retrieve a role by their ID
// @Tags roles
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} Role "Role found"
// @Failure 404 {object} map[string]interface{} "Role not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/roles/{id} [get]
func (h *RoleHandler) GetRole(c *gin.Context) {
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role ID format",
		})
		return
	}

	role, err := h.roleService.GetRoleByID(uint(id))
	if err != nil {
		h.logger.Error("Failed to get role", "error", err, "id", id)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Role not found",
		})
		return
	}

	c.JSON(http.StatusOK, role)
}

// GetRoleByName handles the GET /auth/roles/name/:name endpoint
// @Summary Get a role by name
// @Description Retrieve a role by their name
// @Tags roles
// @Produce json
// @Param name path string true "Role name"
// @Success 200 {object} Role "Role found"
// @Failure 404 {object} map[string]interface{} "Role not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/roles/name/{name} [get]
func (h *RoleHandler) GetRoleByName(c *gin.Context) {
	name := c.Param("name")

	role, err := h.roleService.GetRoleByName(name)
	if err != nil {
		h.logger.Error("Failed to get role by name", "error", err, "name", name)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Role not found",
		})
		return
	}

	c.JSON(http.StatusOK, role)
}

// UpdateRole handles the PUT /auth/roles/:id endpoint
// @Summary Update a role
// @Description Update an existing role's information
// @Tags roles
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param request body UpdateRoleRequest true "Role update request"
// @Success 200 {object} Role "Role updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Role not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/roles/{id} [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role ID format",
		})
		return
	}

	// Get existing role
	existingRole, err := h.roleService.GetRoleByID(uint(id))
	if err != nil {
		h.logger.Error("Failed to get role", "error", err, "id", id)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Role not found",
		})
		return
	}

	// Update fields
	existingRole.Name = req.Name
	existingRole.SetPermissions(convertToPermissions(req.Permissions))

	if err := h.roleService.UpdateRole(existingRole); err != nil {
		h.logger.Error("Failed to update role", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, existingRole)
}

// DeleteRole handles the DELETE /auth/roles/:id endpoint
// @Summary Delete a role
// @Description Delete a role by their ID
// @Tags roles
// @Param id path string true "Role ID"
// @Success 204 "Role deleted successfully"
// @Failure 404 {object} map[string]interface{} "Role not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role ID format",
		})
		return
	}

	if err := h.roleService.DeleteRole(uint(id)); err != nil {
		h.logger.Error("Failed to delete role", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListRoles handles the GET /auth/roles endpoint
// @Summary List all roles
// @Description Retrieve a list of all roles
// @Tags roles
// @Produce json
// @Success 200 {array} Role "List of roles"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.roleService.ListRoles()
	if err != nil {
		h.logger.Error("Failed to list roles", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// convertToPermissions converts string slice to auth.Permission slice
func convertToPermissions(permissions []string) []auth.Permission {
	result := make([]auth.Permission, len(permissions))
	for i, p := range permissions {
		result[i] = auth.Permission(p)
	}
	return result
}
