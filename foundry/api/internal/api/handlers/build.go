package handlers

import (
	"net/http"
	"time"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	metrics "github.com/input-output-hk/catalyst-forge/foundry/api/internal/metrics"
	adm "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/audit"
	build "github.com/input-output-hk/catalyst-forge/foundry/api/internal/models/build"
	auditrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/audit"
	buildrepo "github.com/input-output-hk/catalyst-forge/foundry/api/internal/repository/build"
	"gorm.io/datatypes"
)

type BuildHandler struct {
	sessions         buildrepo.BuildSessionRepository
	sessionMaxActive int
	audits           auditrepo.LogRepository
}

func NewBuildHandler(repo buildrepo.BuildSessionRepository, sessionMaxActive int, audits auditrepo.LogRepository) *BuildHandler {
	return &BuildHandler{sessions: repo, sessionMaxActive: sessionMaxActive, audits: audits}
}

type CreateBuildSessionRequest struct {
	OwnerType string         `json:"owner_type" binding:"required"` // "user" or "repo"
	OwnerID   string         `json:"owner_id" binding:"required"`
	TTL       string         `json:"ttl" binding:"required"` // eg. "90m"
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type CreateBuildSessionResponse struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TODO: metrics hooks can be wired here (eg. prom counter) when metrics package is introduced

// CreateBuildSession creates a new build session enforcing per-owner concurrency cap
func (h *BuildHandler) CreateBuildSession(c *gin.Context) {
	var req CreateBuildSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// enforce cap
	count, err := h.sessions.CountActive(req.OwnerType, req.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check active sessions"})
		return
	}
	if int(count) >= h.sessionMaxActive {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many active sessions"})
		return
	}

	ttl, err := time.ParseDuration(req.TTL)
	if err != nil || ttl <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ttl"})
		return
	}

	bs := &build.BuildSession{
		ID:        uuid.NewString(),
		OwnerType: req.OwnerType,
		OwnerID:   req.OwnerID,
		Source:    "api",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := h.sessions.Create(bs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	// audit event
	metaMap := map[string]any{
		"session_id": bs.ID,
		"owner_type": bs.OwnerType,
		"owner_id":   bs.OwnerID,
		"ttl":        ttl.String(),
	}
	metaJSON, _ := json.Marshal(metaMap)
	_ = h.audits.Create(&adm.Log{
		EventType: "build.session.created",
		RequestIP: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Metadata:  datatypes.JSON(metaJSON),
	})
	if metrics.BuildSessionCreated != nil {
		metrics.BuildSessionCreated.WithLabelValues(bs.OwnerType).Inc()
	}
	c.JSON(http.StatusCreated, CreateBuildSessionResponse{ID: bs.ID, ExpiresAt: bs.ExpiresAt})
}
