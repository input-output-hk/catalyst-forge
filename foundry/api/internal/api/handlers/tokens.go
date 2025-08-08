package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
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

type TokenRevokeRequest struct {
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

// Refresh performs refresh token rotation and returns a new access and refresh pair
// @Summary Refresh tokens
// @Description Rotate the refresh token and return a new access token and refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body TokenRefreshRequest true "Refresh request"
// @Success 200 {object} TokenRefreshResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Invalid token"
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /tokens/refresh [post]
func (h *TokenHandler) Refresh(c *gin.Context) {
	var req TokenRefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Refresh == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Lookup existing refresh by hash
	// HMAC if configured
	secret := os.Getenv("REFRESH_HASH_SECRET")
	var hexHash string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(req.Refresh))
		hexHash = hex.EncodeToString(mac.Sum(nil))
	} else {
		hash := sha256.Sum256([]byte(req.Refresh))
		hexHash = hex.EncodeToString(hash[:])
	}
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
	var newHashHex string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(newOpaque))
		newHashHex = hex.EncodeToString(mac.Sum(nil))
	} else {
		newHash := sha256.Sum256([]byte(newOpaque))
		newHashHex = hex.EncodeToString(newHash[:])
	}
	ttl := existing.ExpiresAt.Sub(existing.CreatedAt)
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	newRefresh := &dbmodel.RefreshToken{
		UserID:    existing.UserID,
		DeviceID:  existing.DeviceID,
		TokenHash: newHashHex,
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

	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{EventType: "token.refresh", SubjectUserID: &user.ID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
		}
	}
	c.JSON(http.StatusOK, TokenRefreshResponse{Access: token, Refresh: newOpaque})
}

// Revoke invalidates the provided refresh token and any linked chain
// Revoke invalidates a refresh token and its chain
// @Summary Revoke token
// @Description Revoke a refresh token and any linked chain
// @Tags auth
// @Accept json
// @Produce json
// @Param request body TokenRevokeRequest true "Revoke request"
// @Success 200 {object} map[string]interface{} "status"
// @Router /tokens/revoke [post]
func (h *TokenHandler) Revoke(c *gin.Context) {
	var req TokenRevokeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Refresh == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	secret := os.Getenv("REFRESH_HASH_SECRET")
	var hexHash string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(req.Refresh))
		hexHash = hex.EncodeToString(mac.Sum(nil))
	} else {
		hash := sha256.Sum256([]byte(req.Refresh))
		hexHash = hex.EncodeToString(hash[:])
	}

	existing, err := h.refreshRepo.GetByHash(hexHash)
	if err != nil || existing == nil {
		// Respond 200 to avoid token probing; operation is idempotent
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	_ = h.refreshRepo.RevokeChain(existing.ID)
	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{EventType: "token.revoke", SubjectUserID: &existing.UserID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}
