package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/metrics"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	usersvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
	"gorm.io/datatypes"
)

type SessionHandler struct {
	refreshRepo userrepo.RefreshTokenRepository
	userService usersvc.UserService
	roleService usersvc.RoleService
	userRoleSvc usersvc.UserRoleService
	jwtManager  jwt.JWTManager
}

func NewSessionHandler(refreshRepo userrepo.RefreshTokenRepository, userService usersvc.UserService, roleService usersvc.RoleService, userRoleSvc usersvc.UserRoleService, jwtManager jwt.JWTManager) *SessionHandler {
	return &SessionHandler{refreshRepo: refreshRepo, userService: userService, roleService: roleService, userRoleSvc: userRoleSvc, jwtManager: jwtManager}
}

// POST /sessions/refresh
func (h *SessionHandler) Refresh(c *gin.Context) {
	rt, err := c.Cookie("cforge_rt")
	if err != nil || rt == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh cookie"})
		return
	}

	secret := os.Getenv("REFRESH_HASH_SECRET")
	var hexHash string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(rt))
		hexHash = hex.EncodeToString(mac.Sum(nil))
	} else {
		sum := sha256.Sum256([]byte(rt))
		hexHash = hex.EncodeToString(sum[:])
	}

	existing, err := h.refreshRepo.GetByHash(hexHash)
	if err != nil || existing == nil {
		metrics.SessionRefreshTotal.WithLabelValues("invalid").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if existing.RevokedAt != nil || time.Now().After(existing.ExpiresAt) {
		metrics.SessionRefreshTotal.WithLabelValues("invalid").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	if existing.ReplacedBy != nil {
		_ = h.refreshRepo.RevokeChain(existing.ID)
		metrics.SessionRefreshTotal.WithLabelValues("reused").Inc()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Lazy backfill of claims snapshot
	var snapshot tokens.ClaimsSnapshot
	if existing.ClaimsJSON == "" {
		// Aggregate permissions once
		user, err := h.userService.GetUserByID(existing.UserID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		permSet := map[auth.Permission]bool{}
		roles, err := h.userRoleSvc.GetUserRoles(user.ID)
		if err == nil {
			for _, ur := range roles {
				if r, err := h.roleService.GetRoleByID(ur.RoleID); err == nil {
					for _, p := range r.GetPermissions() {
						permSet[p] = true
					}
				}
			}
		}
		var perms []auth.Permission
		for p := range permSet {
			perms = append(perms, p)
		}
		sort.Slice(perms, func(i, j int) bool { return perms[i] < perms[j] })
		// Compute authz hash over sorted perms
		ph := sha256.Sum256([]byte(strings.Join(func(ps []auth.Permission) []string {
			s := make([]string, len(ps))
			for i, v := range ps {
				s[i] = string(v)
			}
			return s
		}(perms), ",")))
		authzHash := hex.EncodeToString(ph[:])
		sid := uuid.NewString()
		snapshot = tokens.ClaimsSnapshot{Subject: user.Email, Email: user.Email, SessionID: sid, Permissions: perms, AuthzHash: authzHash}
		b, _ := json.Marshal(snapshot)
		_ = h.refreshRepo.UpdateClaims(existing.ID, string(b), authzHash)
	} else {
		if err := json.Unmarshal([]byte(existing.ClaimsJSON), &snapshot); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
	}

	// Rotate refresh token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	opaque := base64.RawURLEncoding.EncodeToString(raw)
	var newHashHex string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(opaque))
		newHashHex = hex.EncodeToString(mac.Sum(nil))
	} else {
		sum := sha256.Sum256([]byte(opaque))
		newHashHex = hex.EncodeToString(sum[:])
	}
	ttl := existing.ExpiresAt.Sub(existing.CreatedAt)
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	newRefresh := &dbmodel.RefreshToken{
		UserID:     existing.UserID,
		DeviceID:   existing.DeviceID,
		TokenHash:  newHashHex,
		ExpiresAt:  time.Now().Add(ttl),
		ClaimsJSON: existing.ClaimsJSON,
		AuthzHash:  existing.AuthzHash,
	}
	if err := h.refreshRepo.Create(newRefresh); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	_ = h.refreshRepo.MarkReplaced(existing.ID, newRefresh.ID)
	_ = h.refreshRepo.TouchUsage(existing.ID, time.Now())

	// Mint access from frozen claims (30m)
	access, err := tokens.GenerateFromFrozen(h.jwtManager, snapshot, 30*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}

	setAccessCookie(c, access, 30*time.Minute)
	setRefreshCookie(c, opaque, ttl)

	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			md, _ := json.Marshal(map[string]any{"sid": snapshot.SessionID})
			_ = ar.Create(&adm.Log{EventType: "session.refresh", SubjectUserID: &existing.UserID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent(), Metadata: datatypes.JSON(md)})
		}
	}
	metrics.SessionRefreshTotal.WithLabelValues("success").Inc()
	c.Status(http.StatusNoContent)
}

// POST /sessions/logout
func (h *SessionHandler) Logout(c *gin.Context) {
	if rt, err := c.Cookie("cforge_rt"); err == nil && rt != "" {
		secret := os.Getenv("REFRESH_HASH_SECRET")
		var hexHash string
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(rt))
			hexHash = hex.EncodeToString(mac.Sum(nil))
		} else {
			sum := sha256.Sum256([]byte(rt))
			hexHash = hex.EncodeToString(sum[:])
		}
		if existing, _ := h.refreshRepo.GetByHash(hexHash); existing != nil {
			_ = h.refreshRepo.RevokeChain(existing.ID)
			if v, ok := c.Get("auditRepo"); ok {
				if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
					var md datatypes.JSON
					if existing.ClaimsJSON != "" {
						var snap tokens.ClaimsSnapshot
						if err := json.Unmarshal([]byte(existing.ClaimsJSON), &snap); err == nil {
							b, _ := json.Marshal(map[string]any{"sid": snap.SessionID})
							md = datatypes.JSON(b)
						}
					}
					_ = ar.Create(&adm.Log{EventType: "session.logout", SubjectUserID: &existing.UserID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent(), Metadata: md})
				}
			}
		}
	}
	clearAccessCookie(c)
	clearRefreshCookie(c)
	metrics.SessionLogoutTotal.WithLabelValues("success").Inc()
	c.Status(http.StatusNoContent)
}
