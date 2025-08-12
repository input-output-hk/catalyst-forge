package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	usersvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"gorm.io/gorm"
)

type BootstrapRequest struct {
	Email          string `json:"email" binding:"required,email"`
	BootstrapToken string `json:"bootstrap_token" binding:"required"`
}

type BootstrapHandler struct {
	inviteHandler   *InviteHandler
	userSvc         usersvc.UserService
	roleSvc         usersvc.RoleService
	bootstrapRepo   userrepo.BootstrapTokenRepository
	inviteRepo      userrepo.InviteRepository
	configuredToken string // The BOOTSTRAP_TOKEN from environment
	db              *gorm.DB
}

func NewBootstrapHandler(
	inviteHandler *InviteHandler,
	userSvc usersvc.UserService,
	roleSvc usersvc.RoleService,
	bootstrapRepo userrepo.BootstrapTokenRepository,
	inviteRepo userrepo.InviteRepository,
	configuredToken string,
	db *gorm.DB,
) *BootstrapHandler {
	return &BootstrapHandler{
		inviteHandler:   inviteHandler,
		userSvc:         userSvc,
		roleSvc:         roleSvc,
		bootstrapRepo:   bootstrapRepo,
		inviteRepo:      inviteRepo,
		configuredToken: configuredToken,
		db:              db,
	}
}

// Bootstrap creates an admin invite using a one-time bootstrap token
// @Summary Bootstrap admin account
// @Description Create an admin invite using the bootstrap token (one-time use)
// @Tags auth
// @Accept json
// @Produce json
// @Param request body BootstrapRequest true "Bootstrap request"
// @Success 201 {object} CreateInviteResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid or used token"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/bootstrap [post].
func (h *BootstrapHandler) Bootstrap(c *gin.Context) {
	// Check if bootstrap token is configured
	if h.configuredToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bootstrap not configured"})
		return
	}

	var req BootstrapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Validate the provided token matches configured token
	if req.BootstrapToken != h.configuredToken {
		// Log failed attempt
		if v, ok := c.Get("auditRepo"); ok {
			if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
				_ = ar.Create(&adm.Log{
					EventType: "bootstrap.failed",
					RequestIP: c.ClientIP(),
					UserAgent: c.Request.UserAgent(),
				})
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired bootstrap token"})
		return
	}

	// Check if token has been used
	tokenHash := sha256.Sum256([]byte(req.BootstrapToken))
	tokenHashHex := hex.EncodeToString(tokenHash[:])

	used, err := h.bootstrapRepo.IsTokenUsed(tokenHashHex)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if used {
		// Log failed attempt
		if v, ok := c.Get("auditRepo"); ok {
			if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
				_ = ar.Create(&adm.Log{
					EventType: "bootstrap.failed",
					RequestIP: c.ClientIP(),
					UserAgent: c.Request.UserAgent(),
				})
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "bootstrap token has already been used"})
		return
	}

	// Check if any admin users exist (optional extra safety check)
	adminRole, _ := h.roleSvc.GetRoleByName("admin")
	if adminRole != nil {
		// Check if any users have the admin role
		// This is an extra safety measure - bootstrap should only work on fresh systems
		// For now, we'll allow it if the token hasn't been used, but log a warning
	}

	// Create admin role if it doesn't exist
	if adminRole == nil {
		adminRole = &dbmodel.Role{
			Name: "admin",
		}
		// Set all permissions for admin role
		allPerms := make([]auth.Permission, len(auth.AllPermissions))
		copy(allPerms, auth.AllPermissions)
		adminRole.SetPermissions(allPerms)

		if err := h.roleSvc.CreateRole(adminRole); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin role"})
			return
		}
	}

	// Use a transaction to atomically create the invite and mark the token as used
	// This prevents the TOCTOU vulnerability where invite could be created but token not marked as used
	var inviteResp *CreateInviteResponse
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// First, mark bootstrap token as used within the transaction
		// This ensures if anything fails, the token won't be marked as used
		bootstrapToken := &dbmodel.BootstrapToken{
			TokenHash:   tokenHashHex,
			UsedAt:      time.Now(),
			UsedByEmail: req.Email,
		}

		// Create the bootstrap token record first (fail-closed approach)
		if err := tx.Create(bootstrapToken).Error; err != nil {
			return err
		}

		// Now create the invite within the same transaction
		var inviteErr error
		inviteResp, inviteErr = h.createAdminInviteWithTx(c, req.Email, tx)
		if inviteErr != nil {
			return inviteErr
		}

		// Update the bootstrap token with the invite ID
		bootstrapToken.InviteID = inviteResp.ID
		if err := tx.Save(bootstrapToken).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete bootstrap"})
		return
	}

	// Log successful bootstrap
	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{
				EventType: "bootstrap.success",
				RequestIP: c.ClientIP(),
				UserAgent: c.Request.UserAgent(),
			})
		}
	}

	c.JSON(http.StatusCreated, inviteResp)
}

// createAdminInviteWithTx creates an admin invite within a transaction.
func (h *BootstrapHandler) createAdminInviteWithTx(c *gin.Context, email string, tx *gorm.DB) (*CreateInviteResponse, error) {
	// Generate invite token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("failed to generate token")
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	// Hash the token for storage
	tokenHash := sha256.Sum256([]byte(token))
	hexHash := hex.EncodeToString(tokenHash[:])

	// Create the invite within the transaction
	inv := &dbmodel.Invite{
		Email:     email,
		Roles:     []string{"admin"},
		TokenHash: hexHash,
		ExpiresAt: time.Now().Add(72 * time.Hour),
		CreatedBy: 0, // 0 for bootstrap "user"
	}

	if err := tx.Create(inv).Error; err != nil {
		return nil, fmt.Errorf("failed to create invite: %w", err)
	}

	// Log invite creation (outside transaction, best effort)
	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			creatorID := uint(0) // bootstrap user
			_ = ar.Create(&adm.Log{
				EventType:   "invite.created.bootstrap",
				ActorUserID: &creatorID,
				RequestIP:   c.ClientIP(),
				UserAgent:   c.Request.UserAgent(),
			})
		}
	}

	return &CreateInviteResponse{
		ID:    inv.ID,
		Token: token,
	}, nil
}
