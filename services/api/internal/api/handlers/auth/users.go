package auth

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	apiauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

type AdminUsersDeps struct {
	Users      store.UserStore
	Refresh    store.RefreshStore
	RefreshSvc akservice.RefreshService
	Creds      store.CredentialStore
	Recovery   store.RecoveryCodeStore
	Rand       crypto.Rand
	Audit      store.AuditStore
}

// RegisterAdminUsers registers admin endpoints for user management.
func RegisterAdminUsers(r *gin.Engine, deps AdminUsersDeps) {
	g := r.Group("/api/v1/admin")

	// Public: preview invite metadata (email/expiry) without revealing sensitive data
	// @Summary Preview invite (public)
	// @Tags admin
	// @Produce json
	// @Success 200 {object} apimodels.InvitePreviewResponse
	// @Router /api/v1/admin/invites/preview [get]
	g.GET("/invites/preview", func(c *gin.Context) {
		token := c.Query("token")
		idStr := c.Query("id")
		var inviteID *uuid.UUID
		if idStr != "" {
			if id, err := uuid.Parse(idStr); err == nil {
				inviteID = &id
			}
		}
		depsAll := apiauth.BuildDepsCached()
		if depsAll.Stores.Invites == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		var emailPtr *string
		var expPtr *time.Time
		if inviteID != nil {
			if inv, err := depsAll.Stores.Invites.Get(c.Request.Context(), *inviteID); err == nil && inv != nil {
				email := inv.Email
				emailPtr = &email
				exp := inv.ExpiresAt
				expPtr = &exp
			}
		}
		valid := token != ""
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.InvitePreviewResponse{Valid: valid, Email: emailPtr, ExpiresAt: expPtr})
	})

	// @Summary Create invite (admin)
	// @Tags admin
	// @Accept json
	// @Produce json
	// @Param request body apimodels.AdminInviteCreateRequest true "Invite create request"
	// @Success 201 {object} apimodels.AdminInviteCreateResponse
	// @Router /api/v1/admin/invites [post]
	g.POST("/invites", func(c *gin.Context) {
		ctx, ok := apiauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in apimodels.AdminInviteCreateRequest
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.Email == "" || in.DaysToExpire <= 0 || in.DaysToExpire > 60 {
			_ = basehttp.NewBadRequestError("invalid input").Write(c.Writer)
			return
		}
		// Create raw token and hash
		raw, err := deps.Rand.Bytes(32)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		hash := crypto.HashSHA256(raw)
		expires := time.Now().UTC().Add(time.Duration(in.DaysToExpire) * 24 * time.Hour)
		inv := &domain.Invite{
			ID:        uuid.New(),
			Email:     in.Email,
			Roles:     append([]string(nil), in.Roles...),
			TokenHash: hash,
			ExpiresAt: expires,
			CreatedBy: ctx.UserID,
		}
		// Persist
		depsAll := apiauth.BuildDepsCached()
		if depsAll.Stores.Invites == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		if err := depsAll.Stores.Invites.Create(c.Request.Context(), inv); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		// Build invite link with absolute origin
		tokenStr := base64.RawURLEncoding.EncodeToString(raw)
		origin := c.GetString("public_base_url")
		if origin == "" {
			origin = c.Request.Header.Get("X-Forwarded-Proto") + "://" + c.Request.Host
		}
		link := origin + "/invite/" + tokenStr + "?id=" + inv.ID.String()
		// Optionally email the user
		if in.EmailUser && depsAll.Mailer != nil {
			_ = depsAll.Mailer.SendInviteEmail(c.Request.Context(), in.Email, link)
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusCreated, apimodels.AdminInviteCreateResponse{InviteID: inv.ID.String(), InviteLink: link, ExpiresAt: expires})
	})

	// @Summary List users (admin)
	// @Tags admin
	// @Produce json
	// @Success 200 {object} apimodels.AdminUsersListResponse
	// @Router /api/v1/admin/users [get]
	g.GET("/users", func(c *gin.Context) {
		ctx, ok := apiauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}

		// Parse query params
		q := c.Query("q")
		role := c.Query("role")
		limit := 50
		offset := 0
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if v := c.Query("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}

		users, err := deps.Users.ListFiltered(c.Request.Context(), q, role, limit, offset)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}

		// Count active refresh tokens per user
		now := time.Now().UTC()
		resp := apimodels.AdminUsersListResponse{Users: make([]apimodels.AdminUser, 0, len(users))}
		for _, u := range users {
			// Fetch active refresh tokens to count sessions
			var activeCount int
			var lastActivity *time.Time
			if deps.Refresh != nil {
				if toks, err := deps.Refresh.ListActiveByUser(c.Request.Context(), u.ID, now); err == nil {
					// Count unique families and compute last activity
					fams := map[string]struct{}{}
					for _, t := range toks {
						fams[t.FamilyID.String()] = struct{}{}
						if lastActivity == nil || t.CreatedAt.After(*lastActivity) {
							ts := t.CreatedAt
							lastActivity = &ts
						}
					}
					activeCount = len(fams)
				}
			}
			resp.Users = append(resp.Users, apimodels.AdminUser{
				ID:             u.ID.String(),
				Email:          u.Email,
				Roles:          append([]string(nil), u.Roles...),
				ActiveSessions: activeCount,
				LastActivityAt: lastActivity,
				CreatedAt:      u.CreatedAt,
				UpdatedAt:      u.UpdatedAt,
			})
		}

		if total, err := deps.Users.CountFiltered(c.Request.Context(), q, role); err == nil {
			c.Writer.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, resp)
	})

	// @Summary List user credentials (admin)
	// @Tags admin
	// @Produce json
	// @Success 200 {object} apimodels.CredentialsListResponse
	// @Router /api/v1/admin/users/{id}/credentials [get]
	g.GET("/users/:id/credentials", func(c *gin.Context) {
		ctx, ok := apiauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		uid, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid user id").Write(c.Writer)
			return
		}
		if deps.Creds == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		creds, err := deps.Creds.GetByUser(c.Request.Context(), uid)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := make([]apimodels.CredentialSummary, 0, len(creds))
		for _, cd := range creds {
			out = append(out, apimodels.CredentialSummary{
				ID:         base64.RawURLEncoding.EncodeToString(cd.ID),
				AAGUID:     cd.AAGUID,
				DeviceName: cd.DeviceName,
				SignCount:  cd.SignCount,
				LastUsedAt: cd.LastUsedAt,
			})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.CredentialsListResponse{Credentials: out})
	})

	// @Summary Generate recovery codes for a user (admin)
	// @Tags admin
	// @Produce json
	// @Success 200 {object} apimodels.RecoveryGenerateResponse
	// @Router /api/v1/admin/users/{id}/recovery/codes/generate [post]
	g.POST("/users/:id/recovery/codes/generate", func(c *gin.Context) {
		ctx, ok := apiauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		uid, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid user id").Write(c.Writer)
			return
		}
		if deps.Recovery == nil || deps.Rand == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		const numCodes = 8
		codes := make([]string, 0, numCodes)
		hashes := make([][]byte, 0, numCodes)
		for i := 0; i < numCodes; i++ {
			code, err := deps.Rand.String(10)
			if err != nil {
				_ = basehttp.NewInternalError().Write(c.Writer)
				return
			}
			codes = append(codes, code)
			hashes = append(hashes, crypto.HashSHA256([]byte(code)))
		}
		if err := deps.Recovery.ReplaceCodes(c.Request.Context(), uid, hashes); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.RecoveryGenerateResponse{Codes: codes})
	})

	// @Summary List audit events (admin)
	// @Tags admin
	// @Produce json
	// @Success 200 {object} apimodels.AuditListResponse
	// @Router /api/v1/admin/audit [get]
	g.GET("/audit", func(c *gin.Context) {
		if _, ok := apiauth.From(c); !ok {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var actorID, userID *uuid.UUID
		if v := c.Query("actor_id"); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				actorID = &id
			}
		}
		if v := c.Query("user_id"); v != "" {
			if id, err := uuid.Parse(v); err == nil {
				userID = &id
			}
		}
		types := []string{}
		if v := c.Query("types"); v != "" {
			for _, t := range strings.Split(v, ",") {
				if s := strings.TrimSpace(t); s != "" {
					types = append(types, s)
				}
			}
		}
		var sincePtr, untilPtr *time.Time
		if v := c.Query("since"); v != "" {
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				sincePtr = &ts
			}
		}
		if v := c.Query("until"); v != "" {
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				untilPtr = &ts
			}
		}
		limit, _ := strconv.Atoi(c.Query("limit"))
		offset, _ := strconv.Atoi(c.Query("offset"))
		if limit <= 0 || limit > 200 {
			limit = 50
		}
		if offset < 0 {
			offset = 0
		}
		if deps.Audit == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		events, total, err := deps.Audit.List(c.Request.Context(), actorID, userID, types, sincePtr, untilPtr, limit, offset)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := make([]apimodels.AuditEvent, 0, len(events))
		for _, e := range events {
			var uidStr, aidStr *string
			if e.UserID != nil {
				s := e.UserID.String()
				uidStr = &s
			}
			if e.ActorID != nil {
				s := e.ActorID.String()
				aidStr = &s
			}
			out = append(out, apimodels.AuditEvent{
				ID:        e.ID.String(),
				Type:      string(e.Type),
				UserID:    uidStr,
				ActorID:   aidStr,
				IPAddress: e.IPAddress,
				UserAgent: e.UserAgent,
				Metadata:  e.Metadata,
				CreatedAt: e.CreatedAt,
			})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.AuditListResponse{Events: out, Total: total})
	})

	// @Summary Update a user (admin)
	// @Tags admin
	// @Accept json
	// @Produce json
	// @Param id path string true "User ID"
	// @Success 204
	// @Router /api/v1/admin/users/{id} [patch]
	// @Summary List access requests (admin)
	// @Tags admin
	// @Produce json
	// @Success 200 {object} apimodels.AccessRequestListResponse
	// @Router /api/v1/admin/access-requests [get]
	g.GET("/access-requests", func(c *gin.Context) {
		if _, ok := apiauth.From(c); !ok {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		depsAll := apiauth.BuildDepsCached()
		if depsAll.Stores.Access == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		status := c.Query("status")
		q := c.Query("q")
		limit := 50
		offset := 0
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if v := c.Query("offset"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				offset = n
			}
		}
		rows, total, err := depsAll.Stores.Access.List(c.Request.Context(), status, q, limit, offset)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := apimodels.AccessRequestListResponse{Requests: make([]apimodels.AccessRequest, 0, len(rows)), Total: total}
		for _, r := range rows {
			out.Requests = append(out.Requests, apimodels.AccessRequest{
				ID:        r.ID.String(),
				Email:     r.Email,
				Reason:    r.Reason,
				Status:    r.Status,
				Attempts:  r.Attempts,
				DecidedAt: r.DecidedAt,
				CreatedAt: r.CreatedAt,
			})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, out)
	})

	// @Summary Decide access request (admin)
	// @Tags admin
	// @Accept json
	// @Param id path string true "Access Request ID"
	// @Param request body apimodels.AccessRequestDecideRequest true "Decision"
	// @Success 204
	// @Router /api/v1/admin/access-requests/{id} [patch]
	g.PATCH("/access-requests/:id", func(c *gin.Context) {
		ctx, ok := apiauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		depsAll := apiauth.BuildDepsCached()
		if depsAll.Stores.Access == nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid id").Write(c.Writer)
			return
		}
		var in apimodels.AccessRequestDecideRequest
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		note := strings.TrimSpace(in.Note)
		if err := depsAll.Stores.Access.Decide(c.Request.Context(), id, in.Approve, ctx.UserID, note, time.Now().UTC()); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	g.PATCH("/users/:id", func(c *gin.Context) {
		if _, ok := apiauth.From(c); !ok {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		uid, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid user id").Write(c.Writer)
			return
		}
		var in struct {
			Suspend *bool    `json:"suspend"`
			Roles   []string `json:"roles"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.Suspend != nil {
			if ctx, ok := apiauth.From(c); ok && ctx.IsAuthenticated() && ctx.UserID.String() == uid.String() && *in.Suspend {
				_ = basehttp.NewForbiddenError("cannot disable your own account").Write(c.Writer)
				return
			}
			if err := deps.Users.UpdateSuspended(c.Request.Context(), uid, *in.Suspend, time.Now().UTC()); err != nil {
				_ = basehttp.NewInternalError().Write(c.Writer)
				return
			}
			// If disabling, revoke all active sessions and bump session version
			if *in.Suspend && deps.RefreshSvc != nil {
				_ = deps.RefreshSvc.RevokeAllUser(c.Request.Context(), uid, "account disabled")
			}
		}
		if in.Roles != nil {
			if err := deps.Users.UpdateRoles(c.Request.Context(), uid, in.Roles); err != nil {
				_ = basehttp.NewInternalError().Write(c.Writer)
				return
			}
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary Delete a user (admin)
	// @Tags admin
	// @Produce json
	// @Success 204
	// @Router /api/v1/admin/users/{id} [delete]
	g.DELETE("/users/:id", func(c *gin.Context) {
		if _, ok := apiauth.From(c); !ok {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		uid, err := uuid.Parse(c.Param("id"))
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid user id").Write(c.Writer)
			return
		}
		// Prevent self-delete via API to avoid lockout during tests
		if ctx, ok := apiauth.From(c); ok && ctx.IsAuthenticated() && ctx.UserID == uid {
			_ = basehttp.NewForbiddenError("cannot delete your own account").Write(c.Writer)
			return
		}
		if deps.RefreshSvc != nil {
			_ = deps.RefreshSvc.RevokeAllUser(c.Request.Context(), uid, "user deleted")
		}
		if deps.Users != nil {
			if err := deps.Users.Delete(c.Request.Context(), uid); err != nil {
				_ = basehttp.NewInternalError().Write(c.Writer)
				return
			}
		}
		c.Status(http.StatusNoContent)
	})
}
