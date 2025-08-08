package handlers

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
    userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
    usersvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
    "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
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
}

func NewInviteHandler(invRepo userrepo.InviteRepository, userSvc usersvc.UserService, roleSvc usersvc.RoleService, userRole usersvc.UserRoleService) *InviteHandler {
    return &InviteHandler{invites: invRepo, userSvc: userSvc, roleSvc: roleSvc, userRole: userRole}
}

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
    ttl := 72 * time.Hour
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
    sum := sha256.Sum256([]byte(token))
    hexHash := hex.EncodeToString(sum[:])

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

    c.JSON(http.StatusCreated, CreateInviteResponse{ID: inv.ID, Token: token})
}

func (h *InviteHandler) Verify(c *gin.Context) {
    token := c.Query("token")
    if token == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "missing token"})
        return
    }
    sum := sha256.Sum256([]byte(token))
    hexHash := hex.EncodeToString(sum[:])
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

    c.JSON(http.StatusOK, gin.H{"status": "verified"})
}

func timePtr(t time.Time) *time.Time { return &t }

