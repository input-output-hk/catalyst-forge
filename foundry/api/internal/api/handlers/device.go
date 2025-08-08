package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/middleware"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	dbmodel "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/user"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	userrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/user"
	usersvc "github.com/input-output-hk/catalyst-forge/foundry/api/internal/service/user"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/auth/jwt/tokens"
)

// DeviceInitRequest optionally carries client metadata for display
type DeviceInitRequest struct {
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Fingerprint string `json:"fingerprint"`
}

type DeviceInitResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceTokenRequest polls for completion of the device code flow
type DeviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
}

type DeviceTokenResponse struct {
	Access  string `json:"access,omitempty"`
	Refresh string `json:"refresh,omitempty"`
	Error   string `json:"error,omitempty"` // authorization_pending | slow_down | expired_token | access_denied
}

// DeviceApproveRequest approves a pending session by user_code
type DeviceApproveRequest struct {
	UserCode string `json:"user_code"`
}

// DeviceHandler implements device authorization endpoints (RFC 8628-like)
type DeviceHandler struct {
	repo        userrepo.DeviceSessionRepository
	deviceRepo  userrepo.DeviceRepository
	refreshRepo userrepo.RefreshTokenRepository
	userSvc     usersvc.UserService
	roleSvc     usersvc.RoleService
	userRoleSvc usersvc.UserRoleService
	jwtManager  jwt.JWTManager
	logger      *slog.Logger
	// configuration (defaults for now)
	defaultExpires  time.Duration
	defaultInterval int
}

func NewDeviceHandler(repo userrepo.DeviceSessionRepository, deviceRepo userrepo.DeviceRepository, refreshRepo userrepo.RefreshTokenRepository, userSvc usersvc.UserService, roleSvc usersvc.RoleService, userRoleSvc usersvc.UserRoleService, jwtManager jwt.JWTManager, logger *slog.Logger) *DeviceHandler {
	return &DeviceHandler{
		repo: repo, deviceRepo: deviceRepo, refreshRepo: refreshRepo,
		userSvc: userSvc, roleSvc: roleSvc, userRoleSvc: userRoleSvc,
		jwtManager:     jwtManager,
		logger:         logger,
		defaultExpires: 15 * time.Minute, defaultInterval: 5,
	}
}

// Init starts a new device authorization session
// @Summary Start device authorization
// @Description Initialize a device authorization session and return device_code and user_code
// @Tags device
// @Accept json
// @Produce json
// @Param request body DeviceInitRequest false "Optional device metadata"
// @Success 200 {object} DeviceInitResponse
// @Failure 500 {object} map[string]interface{} "Server error"
// @Router /device/init [post]
// POST /device/init
func (h *DeviceHandler) Init(c *gin.Context) {
	var req DeviceInitRequest
	_ = c.ShouldBindJSON(&req) // metadata is optional; ignore bind errors

	deviceCode := randomOpaque(32)
	userCode := humanUserCode(8)

	// compute response
	expiresIn := int(h.defaultExpires.Seconds())
	interval := h.defaultInterval

	// build verification URI from context/env
	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if v, ok := c.Get("public_base_url"); ok {
		if s, ok2 := v.(string); ok2 && s != "" {
			baseURL = s
		}
	}
	verificationURI := strings.TrimRight(baseURL, "/") + "/device"

	// persist in DB
	sess := &dbmodel.DeviceSession{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		ExpiresAt:       time.Now().Add(h.defaultExpires),
		IntervalSeconds: interval,
		Status:          "pending",
		Name:            req.Name,
		Platform:        req.Platform,
		Fingerprint:     req.Fingerprint,
	}
	if err := h.repo.Create(sess); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		return
	}
	if h.logger != nil {
		h.logger.Info("device session created", "user_code", userCode, "expires_in", expiresIn)
	}

	c.JSON(http.StatusOK, DeviceInitResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURI: verificationURI,
		ExpiresIn:       expiresIn,
		Interval:        interval,
	})
}

// Token polls the device session; if approved, issues tokens
// @Summary Poll device token
// @Description Poll the device authorization session for completion and receive tokens when approved
// @Tags device
// @Accept json
// @Produce json
// @Param request body DeviceTokenRequest true "Device token request"
// @Success 200 {object} DeviceTokenResponse
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} DeviceTokenResponse "authorization_pending | expired_token | access_denied"
// @Failure 429 {object} DeviceTokenResponse "slow_down"
// @Router /device/token [post]
// POST /device/token
func (h *DeviceHandler) Token(c *gin.Context) {
	var req DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.DeviceCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	sess, err := h.repo.GetByDeviceCode(req.DeviceCode)
	if err != nil || sess == nil {
		c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "expired_token"})
		return
	}

	now := time.Now()
	if now.After(sess.ExpiresAt) {
		_ = h.repo.UpdateStatus(sess.ID, "denied")
		c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "expired_token"})
		return
	}

	// enforce polling interval
	if sess.LastPolledAt != nil {
		minNext := sess.LastPolledAt.Add(time.Duration(sess.IntervalSeconds) * time.Second)
		if now.Before(minNext) {
			_ = h.repo.TouchPoll(sess.ID, now)
			_ = h.repo.IncrementPollCount(sess.ID)
			// Exponential backoff up to 15s
			newInterval := sess.IntervalSeconds * 2
			if newInterval < 1 {
				newInterval = 1
			}
			if newInterval > 15 {
				newInterval = 15
			}
			if newInterval != sess.IntervalSeconds {
				_ = h.repo.UpdateInterval(sess.ID, newInterval)
			}
			if h.logger != nil {
				h.logger.Warn("device poll too fast", "interval", sess.IntervalSeconds, "new_interval", newInterval)
			}
			c.JSON(http.StatusTooManyRequests, DeviceTokenResponse{Error: "slow_down"})
			return
		}
	}
	_ = h.repo.TouchPoll(sess.ID, now)
	_ = h.repo.IncrementPollCount(sess.ID)
	// Cap total polls to prevent abuse (e.g., > 600 polls ~ 50min at 5s)
	if sess.PollCount+1 > 600 {
		_ = h.repo.UpdateStatus(sess.ID, "denied")
		if h.logger != nil {
			h.logger.Warn("device poll cap exceeded; denying session", "poll_count", sess.PollCount+1)
		}
		c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "access_denied"})
		return
	}

	switch sess.Status {
	case "pending":
		c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "authorization_pending"})
		return
	case "denied":
		c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "access_denied"})
		return
	case "approved":
		if sess.ApprovedUserID == nil {
			c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "authorization_pending"})
			return
		}
		// Aggregate permissions
		permSet := map[auth.Permission]bool{}
		userRoles, err := h.userRoleSvc.GetUserRoles(*sess.ApprovedUserID)
		if err == nil {
			for _, ur := range userRoles {
				r, err := h.roleSvc.GetRoleByID(ur.RoleID)
				if err != nil {
					continue
				}
				for _, p := range r.GetPermissions() {
					permSet[p] = true
				}
			}
		}
		var perms []auth.Permission
		for p := range permSet {
			perms = append(perms, p)
		}
		// Lookup user for subject
		u, err := h.userSvc.GetUserByID(*sess.ApprovedUserID)
		if err != nil || u == nil {
			c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "access_denied"})
			return
		}
		// Issue access token (30m)
		access, err := tokens.GenerateAuthToken(h.jwtManager, u.Email, perms, 30*time.Minute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		// Ensure device row exists
		var deviceID *uint
		if sess.Fingerprint != "" {
			if d, err := h.deviceRepo.GetByUserAndFingerprint(*sess.ApprovedUserID, sess.Fingerprint); err == nil && d != nil {
				deviceID = &d.ID
			} else {
				d := &dbmodel.Device{UserID: *sess.ApprovedUserID, Name: sess.Name, Platform: sess.Platform, Fingerprint: sess.Fingerprint}
				if err := h.deviceRepo.Create(d); err == nil {
					deviceID = &d.ID
				}
			}
		}
		// Create refresh token (opaque, 30d default)
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		opaque := base64.RawURLEncoding.EncodeToString(raw)
		var hashHex string
		if secret := os.Getenv("REFRESH_HASH_SECRET"); secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write([]byte(opaque))
			hashHex = hex.EncodeToString(mac.Sum(nil))
		} else {
			sum := sha256.Sum256([]byte(opaque))
			hashHex = hex.EncodeToString(sum[:])
		}
		ttl := 30 * 24 * time.Hour
		rt := &dbmodel.RefreshToken{UserID: *sess.ApprovedUserID, DeviceID: deviceID, TokenHash: hashHex, ExpiresAt: time.Now().Add(ttl)}
		if err := h.refreshRepo.Create(rt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		// Optionally mark session completed
		_ = h.repo.UpdateStatus(sess.ID, "completed")
		if h.logger != nil {
			h.logger.Info("device session completed", "user_id", *sess.ApprovedUserID)
		}
		if v, ok := c.Get("auditRepo"); ok {
			if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
				_ = ar.Create(&adm.Log{EventType: "device.tokens_issued", SubjectUserID: sess.ApprovedUserID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
			}
		}
		c.JSON(http.StatusOK, DeviceTokenResponse{Access: access, Refresh: opaque})
		return
	default:
		c.JSON(http.StatusUnauthorized, DeviceTokenResponse{Error: "authorization_pending"})
		return
	}
}

// Approve sets a pending session to approved for the current authenticated user
// @Summary Approve device session
// @Description Approve a pending device session identified by user_code
// @Tags device
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body DeviceApproveRequest true "Approval request"
// @Success 200 {object} map[string]interface{} "approved"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 404 {object} map[string]interface{} "Not found"
// @Router /device/approve [post]
// POST /device/approve
func (h *DeviceHandler) Approve(c *gin.Context) {
	var req DeviceApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.UserCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	// Get authed user from context
	uval, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	au, ok := uval.(*middleware.AuthenticatedUser)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Resolve user ID
	user, err := h.userSvc.GetUserByEmail(au.ID)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := h.repo.ApproveByUserCode(req.UserCode, user.ID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if v, ok := c.Get("auditRepo"); ok {
		if ar, ok2 := v.(auditrepo.LogRepository); ok2 {
			_ = ar.Create(&adm.Log{EventType: "device.approved", ActorUserID: &user.ID, RequestIP: c.ClientIP(), UserAgent: c.Request.UserAgent()})
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "approved"})
}

func randomOpaque(numBytes int) string {
	b := make([]byte, numBytes)
	if _, err := rand.Read(b); err != nil {
		// fallback to time if rng fails
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func humanUserCode(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // avoid ambiguous chars
	runes := make([]rune, 0, length+1)
	for i := 0; i < length; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		runes = append(runes, rune(alphabet[idx.Int64()]))
		if i == (length/2)-1 { // insert hyphen in the middle, e.g., ABCD-EFGH
			runes = append(runes, '-')
		}
	}
	return string(runes)
}
