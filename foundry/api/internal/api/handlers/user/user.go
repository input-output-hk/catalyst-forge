package user

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
)

// UserHandler handles user endpoints
type UserHandler struct {
	userService userservice.UserService
	logger      *slog.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService userservice.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Email  string `json:"email" binding:"required,email"`
	Status string `json:"status,omitempty"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email  string `json:"email" binding:"required,email"`
	Status string `json:"status,omitempty"`
}

// RegisterUserRequest represents the request body for registering a user
type RegisterUserRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// CreateUser handles the POST /auth/users endpoint
// @Summary Create a new user
// @Description Create a new user with the provided information
// @Tags users
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User creation request"
// @Success 201 {object} user.User "User created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 409 {object} map[string]interface{} "User already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Convert request to user model
	user := &user.User{
		Email:  req.Email,
		Status: user.UserStatus(req.Status),
	}

	if err := h.userService.CreateUser(user); err != nil {
		h.logger.Error("Failed to create user", "error", err, "email", req.Email)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// RegisterUser handles the POST /auth/users/register endpoint
// @Summary Register a new user
// @Description Register a new user with pending status
// @Tags users
// @Accept json
// @Produce json
// @Param request body RegisterUserRequest true "User registration request"
// @Success 201 {object} user.User "User registered successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 409 {object} map[string]interface{} "User already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users/register [post]
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Check if user already exists
	existingUser, err := h.userService.GetUserByEmail(req.Email)
	if err == nil && existingUser != nil {
		h.logger.Warn("User registration attempted for existing email", "email", req.Email)
		c.JSON(http.StatusConflict, gin.H{
			"error": "User already exists with this email address",
		})
		return
	}

	// Convert request to user model with pending status
	user := &user.User{
		Email:  req.Email,
		Status: user.UserStatusPending,
	}

	if err := h.userService.CreateUser(user); err != nil {
		h.logger.Error("Failed to register user", "error", err, "email", req.Email)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUser handles the GET /auth/users/:id endpoint
// @Summary Get a user by ID
// @Description Retrieve a user by their ID
// @Tags users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} user.User "User found"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")

	// Convert string ID to uint
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		h.logger.Error("Invalid user ID format", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	user, err := h.userService.GetUserByID(id)
	if err != nil {
		h.logger.Error("Failed to get user", "error", err, "id", id)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUserByEmail handles the GET /auth/users/email/:email endpoint
// @Summary Get a user by email
// @Description Retrieve a user by their email address
// @Tags users
// @Produce json
// @Param email path string true "User email"
// @Success 200 {object} user.User "User found"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users/email/{email} [get]
func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")

	user, err := h.userService.GetUserByEmail(email)
	if err != nil {
		h.logger.Error("Failed to get user by email", "error", err, "email", email)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateUser handles the PUT /auth/users/:id endpoint
// @Summary Update a user
// @Description Update an existing user's information
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body UpdateUserRequest true "User update request"
// @Success 200 {object} user.User "User updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")

	// Convert string ID to uint
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		h.logger.Error("Invalid user ID format", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get existing user
	existingUser, err := h.userService.GetUserByID(id)
	if err != nil {
		h.logger.Error("Failed to get user", "error", err, "id", id)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	// Update fields
	existingUser.Email = req.Email
	if req.Status != "" {
		existingUser.Status = user.UserStatus(req.Status)
	}

	if err := h.userService.UpdateUser(existingUser); err != nil {
		h.logger.Error("Failed to update user", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, existingUser)
}

// DeleteUser handles the DELETE /auth/users/:id endpoint
// @Summary Delete a user
// @Description Delete a user by their ID
// @Tags users
// @Param id path string true "User ID"
// @Success 204 "User deleted successfully"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// Convert string ID to uint
	var idUint uint
	if _, err := fmt.Sscanf(id, "%d", &idUint); err != nil {
		h.logger.Error("Invalid user ID format for deletion", "error", err, "id", id)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	if err := h.userService.DeleteUser(idUint); err != nil {
		h.logger.Error("Failed to delete user", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListUsers handles the GET /auth/users endpoint
// @Summary List all users
// @Description Get a list of all users in the system
// @Tags users
// @Produce json
// @Success 200 {array} user.User "List of users"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers()
	if err != nil {
		h.logger.Error("Failed to list users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list users",
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

// GetPendingUsers handles the GET /auth/pending/users endpoint
// @Summary List pending users
// @Description Get a list of all users with pending status
// @Tags users
// @Produce json
// @Success 200 {array} user.User "List of pending users"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/pending/users [get]
func (h *UserHandler) GetPendingUsers(c *gin.Context) {
	users, err := h.userService.GetPendingUsers()
	if err != nil {
		h.logger.Error("Failed to get pending users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get pending users",
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

// ActivateUser handles the POST /auth/users/:id/activate endpoint
// @Summary Activate a user
// @Description Activate a user by setting their status to active
// @Tags users
// @Param id path string true "User ID"
// @Success 200 {object} user.User "User activated successfully"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users/{id}/activate [post]
func (h *UserHandler) ActivateUser(c *gin.Context) {
	id := c.Param("id")

	// Convert string ID to uint
	var idUint uint
	if _, err := fmt.Sscanf(id, "%d", &idUint); err != nil {
		h.logger.Error("Invalid user ID format for activation", "error", err, "id", id)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	if err := h.userService.ActivateUser(idUint); err != nil {
		h.logger.Error("Failed to activate user", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.userService.GetUserByID(idUint)
	if err != nil {
		h.logger.Error("Failed to get user after activation", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve updated user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeactivateUser handles the POST /auth/users/:id/deactivate endpoint
// @Summary Deactivate a user
// @Description Deactivate a user by setting their status to inactive
// @Tags users
// @Param id path string true "User ID"
// @Success 200 {object} user.User "User deactivated successfully"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/users/{id}/deactivate [post]
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	id := c.Param("id")

	// Convert string ID to uint
	var idUint uint
	if _, err := fmt.Sscanf(id, "%d", &idUint); err != nil {
		h.logger.Error("Invalid user ID format for deactivation", "error", err, "id", id)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	if err := h.userService.DeactivateUser(idUint); err != nil {
		h.logger.Error("Failed to deactivate user", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	user, err := h.userService.GetUserByID(idUint)
	if err != nil {
		h.logger.Error("Failed to get user after deactivation", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve updated user",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}
