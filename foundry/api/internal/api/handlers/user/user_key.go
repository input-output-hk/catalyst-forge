package user

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	userservice "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
)

// UserKeyHandler handles user key endpoints
type UserKeyHandler struct {
	userKeyService userservice.UserKeyService
	logger         *slog.Logger
}

// NewUserKeyHandler creates a new user key handler
func NewUserKeyHandler(userKeyService userservice.UserKeyService, logger *slog.Logger) *UserKeyHandler {
	return &UserKeyHandler{
		userKeyService: userKeyService,
		logger:         logger,
	}
}

// CreateUserKeyRequest represents the request body for creating a user key
type CreateUserKeyRequest struct {
	UserID    uint   `json:"user_id" binding:"required"`
	Kid       string `json:"kid" binding:"required"`
	PubKeyB64 string `json:"pubkey_b64" binding:"required"`
	Status    string `json:"status,omitempty"`
}

// UpdateUserKeyRequest represents the request body for updating a user key
type UpdateUserKeyRequest struct {
	UserID    *uint   `json:"user_id,omitempty"`
	Kid       *string `json:"kid,omitempty"`
	PubKeyB64 *string `json:"pubkey_b64,omitempty"`
	Status    *string `json:"status,omitempty"`
}

// RegisterUserKeyRequest represents the request body for registering a user key
type RegisterUserKeyRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Kid       string `json:"kid" binding:"required"`
	PubKeyB64 string `json:"pubkey_b64" binding:"required"`
}

// CreateUserKey handles the POST /auth/keys endpoint
// @Summary Create a new user key
// @Description Create a new Ed25519 key for a user
// @Tags user-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateUserKeyRequest true "User key creation request"
// @Success 201 {object} user.UserKey "User key created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 409 {object} map[string]interface{} "User key already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys [post]
func (h *UserKeyHandler) CreateUserKey(c *gin.Context) {
	var req CreateUserKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Convert request to user key model
	userKey := &user.UserKey{
		UserID:    req.UserID,
		Kid:       req.Kid,
		PubKeyB64: req.PubKeyB64,
		Status:    user.UserKeyStatus(req.Status),
	}

	if err := h.userKeyService.CreateUserKey(userKey); err != nil {
		h.logger.Error("Failed to create user key", "error", err, "user_id", req.UserID, "kid", req.Kid)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, userKey)
}

// RegisterUserKey handles the POST /auth/keys/register endpoint
// @Summary Register a new user key
// @Description Register a new Ed25519 key for a user with inactive status
// @Tags user-keys
// @Accept json
// @Produce json
// @Param request body RegisterUserKeyRequest true "User key registration request"
// @Success 201 {object} user.UserKey "User key registered successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 409 {object} map[string]interface{} "User key already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/register [post]
func (h *UserKeyHandler) RegisterUserKey(c *gin.Context) {
	var req RegisterUserKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get user service from context to look up user by email
	userService := c.MustGet("userService").(userservice.UserService)

	// Look up user by email
	h.logger.Info("Looking up user by email", "email", req.Email)
	usr, err := userService.GetUserByEmail(req.Email)
	if err != nil {
		h.logger.Error("Failed to get user by email", "error", err, "email", req.Email)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	h.logger.Info("Found user", "user_id", usr.ID, "email", usr.Email)

	// Check if user key already exists
	existingUserKey, err := h.userKeyService.GetUserKeyByKid(req.Kid)
	if err == nil && existingUserKey != nil {
		h.logger.Warn("User key registration attempted for existing kid", "kid", req.Kid)
		c.JSON(http.StatusConflict, gin.H{
			"error": "User key already exists with this key ID",
		})
		return
	}

	// Convert request to user key model with inactive status
	userKey := &user.UserKey{
		UserID:    usr.ID,
		Kid:       req.Kid,
		PubKeyB64: req.PubKeyB64,
		Status:    user.UserKeyStatusInactive,
	}

	if err := h.userKeyService.CreateUserKey(userKey); err != nil {
		h.logger.Error("Failed to register user key", "error", err, "user_id", usr.ID, "kid", req.Kid)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, userKey)
}

// GetUserKey handles the GET /auth/keys/:id endpoint
// @Summary Get a user key by ID
// @Description Retrieve a user key by their ID
// @Tags user-keys
// @Produce json
// @Security BearerAuth
// @Param id path string true "User Key ID"
// @Success 200 {object} user.UserKey "User key found"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "User key not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/{id} [get]
func (h *UserKeyHandler) GetUserKey(c *gin.Context) {
	idStr := c.Param("id")

	// Convert string ID to uint
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		h.logger.Error("Invalid user key ID format", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user key ID format",
		})
		return
	}

	userKey, err := h.userKeyService.GetUserKeyByID(id)
	if err != nil {
		h.logger.Error("Failed to get user key", "error", err, "id", id)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User key not found",
		})
		return
	}

	c.JSON(http.StatusOK, userKey)
}

// GetUserKeyByKid handles the GET /auth/keys/kid/:kid endpoint
// @Summary Get a user key by kid
// @Description Retrieve a user key by their kid (key ID)
// @Tags user-keys
// @Produce json
// @Security BearerAuth
// @Param kid path string true "Key ID"
// @Success 200 {object} user.UserKey "User key found"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "User key not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/kid/{kid} [get]
func (h *UserKeyHandler) GetUserKeyByKid(c *gin.Context) {
	kid := c.Param("kid")

	userKey, err := h.userKeyService.GetUserKeyByKid(kid)
	if err != nil {
		h.logger.Error("Failed to get user key by kid", "error", err, "kid", kid)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User key not found",
		})
		return
	}

	c.JSON(http.StatusOK, userKey)
}

// GetUserKeysByUserID handles the GET /auth/keys/user/:user_id endpoint
// @Summary Get user keys by user ID
// @Description Retrieve all keys for a specific user
// @Tags user-keys
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {array} user.UserKey "List of user keys"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/user/{user_id} [get]
func (h *UserKeyHandler) GetUserKeysByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")

	// Convert string user_id to uint
	var userID uint
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		h.logger.Error("Invalid user ID format", "error", err, "user_id", userIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	userKeys, err := h.userKeyService.GetUserKeysByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to get user keys by user ID", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, userKeys)
}

// GetActiveUserKeysByUserID handles the GET /auth/keys/user/:user_id/active endpoint
// @Summary Get active user keys by user ID
// @Description Get all active user keys for a specific user
// @Tags user-keys
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {array} user.UserKey "List of active user keys"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/user/{user_id}/active [get]
func (h *UserKeyHandler) GetActiveUserKeysByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")
	var userID uint
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		h.logger.Warn("Invalid user ID format", "user_id", userIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	userKeys, err := h.userKeyService.GetActiveUserKeysByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to get active user keys", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get active user keys",
		})
		return
	}

	c.JSON(http.StatusOK, userKeys)
}

// GetInactiveUserKeysByUserID handles the GET /auth/keys/user/:user_id/inactive endpoint
// @Summary Get inactive user keys by user ID
// @Description Get all inactive user keys for a specific user
// @Tags user-keys
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID"
// @Success 200 {array} user.UserKey "List of inactive user keys"
// @Failure 400 {object} map[string]interface{} "Invalid user ID"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/user/{user_id}/inactive [get]
func (h *UserKeyHandler) GetInactiveUserKeysByUserID(c *gin.Context) {
	userIDStr := c.Param("user_id")
	var userID uint
	if _, err := fmt.Sscanf(userIDStr, "%d", &userID); err != nil {
		h.logger.Warn("Invalid user ID format", "user_id", userIDStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	userKeys, err := h.userKeyService.GetInactiveUserKeysByUserID(userID)
	if err != nil {
		h.logger.Error("Failed to get inactive user keys", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get inactive user keys",
		})
		return
	}

	c.JSON(http.StatusOK, userKeys)
}

// GetInactiveUserKeys handles the GET /auth/pending/keys endpoint
// @Summary Get all inactive user keys
// @Description Get all user keys with inactive status
// @Tags user-keys
// @Produce json
// @Security BearerAuth
// @Success 200 {array} user.UserKey "List of inactive user keys"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/pending/keys [get]
func (h *UserKeyHandler) GetInactiveUserKeys(c *gin.Context) {
	userKeys, err := h.userKeyService.GetInactiveUserKeys()
	if err != nil {
		h.logger.Error("Failed to get inactive user keys", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get inactive user keys",
		})
		return
	}

	c.JSON(http.StatusOK, userKeys)
}

// UpdateUserKey handles the PUT /auth/keys/:id endpoint
// @Summary Update a user key
// @Description Update an existing user key's information
// @Tags user-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User Key ID"
// @Param request body UpdateUserKeyRequest true "User key update request"
// @Success 200 {object} user.UserKey "User key updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "User key not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/{id} [put]
func (h *UserKeyHandler) UpdateUserKey(c *gin.Context) {
	idStr := c.Param("id")

	// Convert string ID to uint
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		h.logger.Error("Invalid user key ID format", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user key ID format",
		})
		return
	}

	var req UpdateUserKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Get existing user key
	existingUserKey, err := h.userKeyService.GetUserKeyByID(id)
	if err != nil {
		h.logger.Error("Failed to get user key", "error", err, "id", id)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User key not found",
		})
		return
	}

	// Update fields only if provided
	if req.UserID != nil {
		existingUserKey.UserID = *req.UserID
	}
	if req.Kid != nil {
		existingUserKey.Kid = *req.Kid
	}
	if req.PubKeyB64 != nil {
		existingUserKey.PubKeyB64 = *req.PubKeyB64
	}
	if req.Status != nil {
		existingUserKey.Status = user.UserKeyStatus(*req.Status)
	}

	if err := h.userKeyService.UpdateUserKey(existingUserKey); err != nil {
		h.logger.Error("Failed to update user key", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, existingUserKey)
}

// DeleteUserKey handles the DELETE /auth/keys/:id endpoint
// @Summary Delete a user key
// @Description Delete a user key by their ID
// @Tags user-keys
// @Security BearerAuth
// @Param id path string true "User Key ID"
// @Success 204 "User key deleted successfully"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "User key not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/{id} [delete]
func (h *UserKeyHandler) DeleteUserKey(c *gin.Context) {
	idStr := c.Param("id")

	// Convert string ID to uint
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		h.logger.Error("Invalid user key ID format", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user key ID format",
		})
		return
	}

	if err := h.userKeyService.DeleteUserKey(id); err != nil {
		h.logger.Error("Failed to delete user key", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// RevokeUserKey handles the POST /auth/keys/:id/revoke endpoint
// @Summary Revoke a user key
// @Description Revoke a user key by setting its status to revoked
// @Tags user-keys
// @Security BearerAuth
// @Param id path string true "User Key ID"
// @Success 200 {object} user.UserKey "User key revoked successfully"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 404 {object} map[string]interface{} "User key not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys/{id}/revoke [post]
func (h *UserKeyHandler) RevokeUserKey(c *gin.Context) {
	idStr := c.Param("id")

	// Convert string ID to uint
	var id uint
	if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
		h.logger.Error("Invalid user key ID format", "error", err, "id", idStr)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user key ID format",
		})
		return
	}

	if err := h.userKeyService.RevokeUserKey(id); err != nil {
		h.logger.Error("Failed to revoke user key", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	userKey, err := h.userKeyService.GetUserKeyByID(id)
	if err != nil {
		h.logger.Error("Failed to get user key after revocation", "error", err, "id", id)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve updated user key",
		})
		return
	}

	c.JSON(http.StatusOK, userKey)
}

// ListUserKeys handles the GET /auth/keys endpoint
// @Summary List all user keys
// @Description Retrieve a list of all user keys
// @Tags user-keys
// @Produce json
// @Security BearerAuth
// @Success 200 {array} user.UserKey "List of user keys"
// @Failure 401 {object} map[string]interface{} "Authentication required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/keys [get]
func (h *UserKeyHandler) ListUserKeys(c *gin.Context) {
	userKeys, err := h.userKeyService.ListUserKeys()
	if err != nil {
		h.logger.Error("Failed to list user keys", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, userKeys)
}
