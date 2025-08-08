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
    "github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
)

type TokenRefreshRequest struct {
    Refresh string `json:"refresh"`
}

type TokenRefreshResponse struct {
    Access  string `json:"access"`
    Refresh string `json:"refresh"`
}

type TokenHandler struct {
    refreshRepo userrepo.RefreshTokenRepository
    userService usersvc.UserService
    roleService usersvc.RoleService
    userRoleSvc usersvc.UserRoleService
    jwtManager  jwt.JWTManager
}

func NewTokenHandler(refreshRepo userrepo.RefreshTokenRepository, userService usersvc.UserService, roleService usersvc.RoleService, userRoleSvc usersvc.UserRoleService, jwtManager jwt.JWTManager) *TokenHandler {
    return &TokenHandler{refreshRepo: refreshRepo, userService: userService, roleService: roleService, userRoleSvc: userRoleSvc, jwtManager: jwtManager}
}

func (h *TokenHandler) Refresh(c *gin.Context) {
    var req TokenRefreshRequest
    if err := c.ShouldBindJSON(&req); err != nil || req.Refresh == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }

    // Lookup existing refresh by hash
    hash := sha256.Sum256([]byte(req.Refresh))
    hexHash := hex.EncodeToString(hash[:])
    existing, err := h.refreshRepo.GetByHash(hexHash)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
        return
    }

    // Validate expiry/revocation
    if existing.RevokedAt != nil || time.Now().After(existing.ExpiresAt) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
        return
    }
    // Reuse detection
    if existing.ReplacedBy != nil {
        _ = h.refreshRepo.RevokeChain(existing.ID)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
        return
    }

    // Load user and aggregate permissions
    user, err := h.userService.GetUserByID(existing.UserID)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
        return
    }
    permSet := map[auth.Permission]bool{}
    roles, err := h.userRoleSvc.GetUserRoles(user.ID)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
        return
    }
    for _, ur := range roles {
        r, err := h.roleService.GetRoleByID(ur.RoleID)
        if err != nil {
            continue
        }
        for _, p := range r.GetPermissions() {
            permSet[p] = true
        }
    }
    var perms []auth.Permission
    for p := range permSet {
        perms = append(perms, p)
    }

    // Create new refresh (rotate)
    raw := make([]byte, 32)
    if _, err := rand.Read(raw); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
        return
    }
    newOpaque := base64.RawURLEncoding.EncodeToString(raw)
    newHash := sha256.Sum256([]byte(newOpaque))
    ttl := existing.ExpiresAt.Sub(existing.CreatedAt)
    if ttl <= 0 {
        ttl = 30 * 24 * time.Hour
    }
    newRefresh := &dbmodel.RefreshToken{
        UserID:    existing.UserID,
        DeviceID:  existing.DeviceID,
        TokenHash: hex.EncodeToString(newHash[:]),
        ExpiresAt: time.Now().Add(ttl),
    }
    if err := h.refreshRepo.Create(newRefresh); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
        return
    }
    _ = h.refreshRepo.MarkReplaced(existing.ID, newRefresh.ID)
    _ = h.refreshRepo.TouchUsage(existing.ID, time.Now())

    // Issue new access token (30m)
    token, err := tokens.GenerateAuthToken(h.jwtManager, user.Email, perms, 30*time.Minute)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
        return
    }

    c.JSON(http.StatusOK, TokenRefreshResponse{Access: token, Refresh: newOpaque})
}

