package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"sort"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	akcrypto "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

type SessionsDeps struct {
	Refresh store.RefreshStore
	Devices store.DeviceStore
}

// RegisterSessions registers session listing and revocation endpoints.
func RegisterSessions(r *gin.Engine, deps SessionsDeps) {
	g := r.Group("/api/v1/auth")

	// @Summary List active sessions
	// @Tags auth
	// @Produce json
	// @Success 200 {object} apimodels.SessionsListResponse
	// @Router /api/v1/auth/sessions [get]
	g.GET("/sessions", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}

		now := time.Now().UTC()
		tokens, err := deps.Refresh.ListActiveByUser(c.Request.Context(), ctx.UserID, now)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}

		// Determine current family from refresh cookie, if present
		var currentFamily uuid.UUID
		if cookie, err := c.Request.Cookie(domain.RefreshCookieName); err == nil && cookie != nil && cookie.Value != "" {
			// Decode raw-url base64 token and hash to look up token
			if raw, err := base64.RawURLEncoding.DecodeString(cookie.Value); err == nil && len(raw) == 32 {
				// Hash same as refresh service uses (SHA-256)
				hash := akcrypto.HashSHA256(raw)
				if tok, err := deps.Refresh.GetByHash(c.Request.Context(), hash); err == nil && tok != nil {
					currentFamily = tok.FamilyID
				}
			}
		}

		// Group by family
		type famAgg struct {
			first time.Time
			last  time.Time
			exp   time.Time
			devID *uuid.UUID
			ua    string
			ip    string
		}
		agg := map[uuid.UUID]*famAgg{}
		for _, t := range tokens {
			a := agg[t.FamilyID]
			if a == nil {
				a = &famAgg{first: t.CreatedAt, last: t.CreatedAt, exp: t.ExpiresAt, devID: t.DeviceID, ua: t.UserAgent, ip: t.IPAddress}
				agg[t.FamilyID] = a
			} else {
				if t.CreatedAt.Before(a.first) {
					a.first = t.CreatedAt
				}
				if t.CreatedAt.After(a.last) {
					a.last = t.CreatedAt
				}
				if t.ExpiresAt.After(a.exp) {
					a.exp = t.ExpiresAt
				}
				if t.UserAgent != "" {
					a.ua = t.UserAgent
				}
				if t.IPAddress != "" {
					a.ip = t.IPAddress
				}
			}
		}

		// Build output, sorted by last activity desc
		out := make([]apimodels.SessionSummary, 0, len(agg))
		for fam, a := range agg {
			var deviceIDStr *string
			var deviceName *string
			amr := "webauthn"
			if a.devID != nil {
				amr = "device_link"
				idStr := a.devID.String()
				deviceIDStr = &idStr
				if deps.Devices != nil {
					if d, _ := deps.Devices.GetByID(c.Request.Context(), *a.devID); d != nil {
						name := d.DeviceName
						deviceName = &name
					}
				}
			}
			out = append(out, apimodels.SessionSummary{
				ID:             fam.String(),
				Current:        fam == currentFamily,
				AMR:            amr,
				DeviceID:       deviceIDStr,
				DeviceName:     deviceName,
				UserAgent:      a.ua,
				IPAddress:      a.ip,
				CreatedAt:      a.first,
				LastActivityAt: a.last,
				ExpiresAt:      a.exp,
			})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].LastActivityAt.After(out[j].LastActivityAt) })
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.SessionsListResponse{Sessions: out})
	})

	// @Summary Revoke a session
	// @Tags auth
	// @Param family_id path string true "Family ID"
	// @Success 204
	// @Router /api/v1/auth/sessions/{family_id} [delete]
	g.DELETE("/sessions/:family_id", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		famStr := c.Param("family_id")
		familyID, err := uuid.Parse(famStr)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid family id").Write(c.Writer)
			return
		}
		token, err := deps.Refresh.GetAnyByFamily(c.Request.Context(), familyID)
		if err != nil || token.UserID != ctx.UserID {
			_ = basehttp.NewNotFoundError("").Write(c.Writer)
			return
		}
		// Revoke entire family (use underlying store method directly)
		type revokeFamily interface {
			RevokeFamily(ctx context.Context, familyID uuid.UUID, reason string, at time.Time) error
		}
		if rf, ok := any(deps.Refresh).(revokeFamily); ok {
			_ = rf.RevokeFamily(c.Request.Context(), familyID, "user logout", time.Now().UTC())
		}
		c.Status(http.StatusNoContent)
	})
}
