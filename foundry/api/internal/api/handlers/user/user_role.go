package user

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
)

// UserRoleHandler handles user-role relationship endpoints
type UserRoleHandler struct {
	userRoleService userservice.UserRoleService
	logger          *slog.Logger
}

// NewUserRoleHandler creates a new user-role handler
func NewUserRoleHandler(userRoleService userservice.UserRoleService, logger *slog.Logger) *UserRoleHandler {
	return &UserRoleHandler{
		userRoleService: userRoleService,
		logger:          logger,
	}
}

// UserRole represents a many-to-many relationship between users and roles (swagger-compatible version)
// @Description UserRole represents a many-to-many relationship between users and roles
type UserRole struct {
	ID        uint      `json:"id" example:"1"`
	UserID    uint      `json:"user_id" example:"123"`
	RoleID    uint      `json:"role_id" example:"456"`
	User      *User     `json:"user,omitempty"`
	Role      *Role     `json:"role,omitempty"`
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

// User represents a user in the system (swagger-compatible version)
// @Description User represents a user in the system
type User struct {
	ID        uint      `json:"id" example:"123"`
	Email     string    `json:"email" example:"user@example.com"`
	Status    string    `json:"status" example:"active"`
	CreatedAt time.Time `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-01-01T00:00:00Z"`
}

// AssignUserToRoleRequest represents the request body for assigning a user to a role
type AssignUserToRoleRequest struct {
	UserID string `json:"user_id" binding:"required"`
	RoleID string `json:"role_id" binding:"required"`
}

// AssignUserToRole handles the POST /auth/user-roles endpoint
// @Summary Assign a user to a role
// @Description Assign a user to a specific role
// @Tags user-roles
// @Accept json
// @Produce json
// @Param user_id query string true "User ID"
// @Param role_id query string true "Role ID"
// @Success 201 {object} UserRole "User assigned to role successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "User or role not found"
// @Failure 409 {object} map[string]interface{} "User already has this role"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/user-roles [post]
func (h *UserRoleHandler) AssignUserToRole(c *gin.Context) {
	userIDStr := c.Query("user_id")
	roleIDStr := c.Query("role_id")

	if userIDStr == "" || roleIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id and role_id are required",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user_id format",
		})
		return
	}

	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role_id format",
		})
		return
	}

	if err := h.userRoleService.AssignUserToRole(uint(userID), uint(roleID)); err != nil {
		h.logger.Error("Failed to assign user to role", "error", err, "userID", userID, "roleID", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusCreated)
}

// RemoveUserFromRole handles the DELETE /auth/user-roles endpoint
// @Summary Remove a user from a role
// @Description Remove a user from a specific role
// @Tags user-roles
// @Param user_id query string true "User ID"
// @Param role_id query string true "Role ID"
// @Success 204 "User removed from role successfully"
// @Failure 404 {object} map[string]interface{} "User or role not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/user-roles [delete]
func (h *UserRoleHandler) RemoveUserFromRole(c *gin.Context) {
	userIDStr := c.Query("user_id")
	roleIDStr := c.Query("role_id")

	if userIDStr == "" || roleIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id and role_id are required",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user_id format",
		})
		return
	}

	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role_id format",
		})
		return
	}

	if err := h.userRoleService.RemoveUserFromRole(uint(userID), uint(roleID)); err != nil {
		h.logger.Error("Failed to remove user from role", "error", err, "userID", userID, "roleID", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetUserRoles handles the GET /auth/user-roles endpoint
// @Summary Get all roles for a user
// @Description Retrieve all roles assigned to a specific user
// @Tags user-roles
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {array} UserRole "List of user roles"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/user-roles [get]
func (h *UserRoleHandler) GetUserRoles(c *gin.Context) {
	userIDStr := c.Query("user_id")

	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "user_id is required",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user_id format",
		})
		return
	}

	userRoles, err := h.userRoleService.GetUserRoles(uint(userID))
	if err != nil {
		h.logger.Error("Failed to get user roles", "error", err, "userID", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, userRoles)
}

// GetRoleUsers handles the GET /auth/role-users endpoint
// @Summary Get all users for a role
// @Description Retrieve all users assigned to a specific role
// @Tags user-roles
// @Produce json
// @Param role_id query string true "Role ID"
// @Success 200 {array} UserRole "List of role users"
// @Failure 404 {object} map[string]interface{} "Role not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/role-users [get]
func (h *UserRoleHandler) GetRoleUsers(c *gin.Context) {
	roleIDStr := c.Query("role_id")

	if roleIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "role_id is required",
		})
		return
	}

	roleID, err := strconv.ParseUint(roleIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid role_id format",
		})
		return
	}

	userRoles, err := h.userRoleService.GetRoleUsers(uint(roleID))
	if err != nil {
		h.logger.Error("Failed to get role users", "error", err, "roleID", roleID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, userRoles)
}
