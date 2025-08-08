package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	emailsvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/email"
	usersvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
)

type CreateInviteRequest struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	TTL   string   `json:"ttl,omitempty"` // e.g., "72h"
}

type CreateInviteResponse struct {
	ID    uint   `json:"id"`
	Token string `json:"token"`
}

type InviteHandler struct {
	invites    userrepo.InviteRepository
	userSvc    usersvc.UserService
	roleSvc    usersvc.RoleService
	userRole   usersvc.UserRoleService
	defaultTTL time.Duration
	email      emailsvc.Service
}

func NewInviteHandler(invRepo userrepo.InviteRepository, userSvc usersvc.UserService, roleSvc usersvc.RoleService, userRole usersvc.UserRoleService, defaultTTL time.Duration, email emailsvc.Service) *InviteHandler {
	return &InviteHandler{invites: invRepo, userSvc: userSvc, roleSvc: roleSvc, userRole: userRole, defaultTTL: defaultTTL, email: email}
}

// CreateInvite issues an invite and optionally emails the recipient
// @Summary Create invite
// @Description Create an invite for a user with one or more roles; optionally emails a verification link
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateInviteRequest true "Invite creation request"
// @Success 201 {object} CreateInviteResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /auth/invites [post]
func (h *InviteHandler) CreateInvite(c *gin.Context) {
	// caller must be authenticated; get user context to set created_by
	userData, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	authUser, ok := userData.(*middleware.AuthenticatedUser)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Email == "" || len(req.Roles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	ttl := h.defaultTTL
	if v, ok := c.Get("invite_default_ttl"); ok {
		if d, ok2 := v.(time.Duration); ok2 && d > 0 {
			ttl = d
		}
	}
	if req.TTL != "" {
		if d, err := time.ParseDuration(req.TTL); err == nil && d > 0 {
			ttl = d
		}
	}

	// resolve creator id
	creator, err := h.userSvc.GetUserByEmail(authUser.ID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// generate invite token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	// Use optional HMAC secret for hashing if configured
	secret := os.Getenv("INVITE_HASH_SECRET")
	var hexHash string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(token))
		hexHash = hex.EncodeToString(mac.Sum(nil))
	} else {
		sum := sha256.Sum256([]byte(token))
		hexHash = hex.EncodeToString(sum[:])
	}

	inv := &dbmodel.Invite{
		Email:     req.Email,
		Roles:     req.Roles,
		TokenHash: hexHash,
		ExpiresAt: time.Now().Add(ttl),
		CreatedBy: creator.ID,
	}
	if err := h.invites.Create(inv); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create invite"})
		return
	}

	// Optionally send email if service is configured
	if h.email != nil {
		baseURL := os.Getenv("PUBLIC_BASE_URL")
		if v, ok := c.Get("public_base_url"); ok {
			if s, ok2 := v.(string); ok2 && s != "" {
				baseURL = s
			}
		}
		link := baseURL + "/verify?token=" + token + "&invite_id=" + fmt.Sprintf("%d", inv.ID)
		_ = h.email.SendInvite(c.Request.Context(), inv.Email, link)
	}

	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{EventType: "invite.created", ActorUserID: &creator.ID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
		}
	}
	c.JSON(http.StatusCreated, CreateInviteResponse{ID: inv.ID, Token: token})
}

// Verify validates an invite token and activates the user, assigning roles
// @Summary Verify invite
// @Description Verify an invite token and activate the user; assigns roles from the invite
// @Tags auth
// @Accept json
// @Produce json
// @Param token query string true "Invite token"
// @Success 200 {object} map[string]interface{} "verified"
// @Failure 400 {object} map[string]interface{} "Missing token"
// @Failure 401 {object} map[string]interface{} "Invalid or expired"
// @Router /verify [get]
func (h *InviteHandler) Verify(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
		return
	}
	secret := os.Getenv("INVITE_HASH_SECRET")
	var hexHash string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(token))
		hexHash = hex.EncodeToString(mac.Sum(nil))
	} else {
		sum := sha256.Sum256([]byte(token))
		hexHash = hex.EncodeToString(sum[:])
	}
	inv, err := h.invites.GetByTokenHash(hexHash)
	if err != nil || inv == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired"})
		return
	}
	if inv.RedeemedAt != nil || time.Now().After(inv.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired"})
		return
	}
	// upsert user
	u, err := h.userSvc.GetUserByEmail(inv.Email)
	if err != nil || u == nil {
		u = &dbmodel.User{Email: inv.Email, Status: dbmodel.UserStatusActive}
		if err := h.userSvc.CreateUser(u); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
			return
		}
	}
	now := timePtr(time.Now())
	u.EmailVerifiedAt = now
	u.Status = dbmodel.UserStatusActive
	_ = h.userSvc.UpdateUser(u)

	// assign roles
	for _, name := range inv.Roles {
		r, err := h.roleSvc.GetRoleByName(name)
		if err != nil || r == nil {
			continue
		}
		_ = h.userRole.AssignUserToRole(u.ID, r.ID)
	}
	_ = h.invites.MarkRedeemed(inv.ID)
	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{EventType: "invite.verified", SubjectUserID: &u.ID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

func timePtr(t time.Time) *time.Time { return &t }
