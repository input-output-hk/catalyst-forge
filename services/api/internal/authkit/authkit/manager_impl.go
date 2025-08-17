package authkit

import (
	"net/http"
	"time"

	"crypto/subtle"
	"encoding/base64"
	"encoding/json"

	"fmt"
	"os"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	akcrypto "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/domain"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/handlers"
	authhttp "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/httpkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
)

// registerRoutes mounts all /auth endpoints. Placeholder until route handlers exist.
func registerRoutes(rg *gin.RouterGroup, cfg Config, deps Deps) {
	// Services used by multiple handlers
	tokenSvc := service.NewTokenService(deps.Keys, deps.Rand, cfg.Origin, cfg.AccessTokenTTL)
	refreshSvc := service.NewRefreshService(service.RefreshServiceConfig{
		Store:      deps.Stores.Refresh,
		UserStore:  deps.Stores.Users,
		TokenSvc:   tokenSvc,
		Rand:       deps.Rand,
		TTL:        cfg.RefreshTokenTTL,
		AuditStore: deps.Stores.Audit,
	})
	cookieCfg := basehttp.DefaultCookieConfig()

	// WebAuthn service
	waCfg := service.WebAuthnConfig{
		RPDisplayName:        cfg.RPName,
		RPID:                 cfg.RPID,
		RPOrigins:            []string{cfg.Origin},
		Users:                deps.Stores.Users,
		Credentials:          deps.Stores.Credentials,
		Challenges:           deps.Stores.Challenges,
		Rand:                 deps.Rand,
		AdminAAGUIDAllowlist: cfg.AdminAAGUIDAllowlist,
		ChallengeTTL:         cfg.ChallengeTTL,
	}
	webauthnSvc, _ := service.NewWebAuthnService(waCfg)
	_ = service.NewStepUpService(deps.KV, cfg.StepUpTTL)

	// Recovery services
	recSvc := service.NewRecoveryService(deps.Stores.RecoveryCodes, deps.Rand)
	recFlow := service.NewRecoveryFlowService(deps.Stores.Users, recSvc, deps.KV, deps.Rand)

	// POST /login/begin
	rg.POST("/login/begin", func(c *gin.Context) {
		options, sessionKey, err := webauthnSvc.BeginLogin(c.Request.Context(), "")
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{
			"publicKey":   options,
			"session_key": sessionKey,
		})
	})

	// POST /login/complete
	rg.POST("/login/complete", func(c *gin.Context) {
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, _, err := webauthnSvc.FinishLogin(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("authentication failed").Write(c.Writer)
			return
		}
		// Issue refresh and access
		cookieValue, _, _, err := refreshSvc.Issue(c.Request.Context(), user, time.Now().UTC())
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		authhttp.SetRefreshCookie(c.Writer, cookieValue, cfg.RefreshTokenTTL, cookieCfg)
		claims := service.AccessClaims{Sub: user.ID.String(), Email: user.Email, Roles: user.Roles, SessionVersion: user.SessionVersion}
		access, err := tokenSvc.SignAccess(c.Request.Context(), claims)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}

		authhttp.SetAccessCookie(c.Writer, access, cfg.AccessTokenTTL, cookieCfg)
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{
			"user":         map[string]any{"id": user.ID.String(), "email": user.Email, "roles": user.Roles},
			"access_token": access,
		})
	})

	// GET /credentials
	rg.GET("/credentials", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		creds, err := deps.Stores.Credentials.GetByUser(c.Request.Context(), ctx.UserID)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		// project minimal info
		out := make([]map[string]any, 0, len(creds))
		for _, cd := range creds {
			out = append(out, map[string]any{
				"id":           base64.RawURLEncoding.EncodeToString(cd.ID),
				"aaguid":       cd.AAGUID,
				"device_name":  cd.DeviceName,
				"sign_count":   cd.SignCount,
				"last_used_at": cd.LastUsedAt,
			})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"credentials": out})
	})

	// POST /credentials/add/begin
	rg.POST("/credentials/add/begin", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		// load user from store
		user, err := deps.Stores.Users.GetByID(c.Request.Context(), ctx.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		requireHW := sliceContains(user.Roles, "admin")
		options, sessionKey, err := webauthnSvc.BeginRegistration(c.Request.Context(), user, in.DeviceName, requireHW)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey})
	})

	// POST /credentials/add/complete
	rg.POST("/credentials/add/complete", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		_, err := webauthnSvc.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// DELETE /credentials/:id
	rg.DELETE("/credentials/:id", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		idParam := c.Param("id")
		id, err := base64.RawURLEncoding.DecodeString(idParam)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid credential id").Write(c.Writer)
			return
		}
		// prevent deleting last credential
		creds, _ := deps.Stores.Credentials.GetByUser(c.Request.Context(), ctx.UserID)
		if len(creds) <= 1 {
			_ = basehttp.NewBadRequestError("cannot remove last credential").Write(c.Writer)
			return
		}
		if err := deps.Stores.Credentials.Revoke(c.Request.Context(), id); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// POST /step-up/begin
	rg.POST("/step-up/begin", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		user, err := deps.Stores.Users.GetByID(c.Request.Context(), ctx.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		options, sessionKey, err := webauthnSvc.BeginStepUp(c.Request.Context(), user, "")
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey})
	})

	// POST /step-up/complete
	rg.POST("/step-up/complete", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := webauthnSvc.FinishStepUp(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("step-up failed").Write(c.Writer)
			return
		}
		// Issue a fresh access token with step-up window
		until := time.Now().UTC().Add(cfg.StepUpTTL)
		claims := service.AccessClaims{Sub: user.ID.String(), Email: user.Email, Roles: user.Roles, SessionVersion: user.SessionVersion, StepUpUntil: until.Unix()}
		access, err := tokenSvc.SignAccess(c.Request.Context(), claims)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"access_token": access})
	})
	// POST /refresh
	rg.POST("/refresh", func(c *gin.Context) {
		if err := deps.CSRF.Validate(c.Request); err != nil {
			_ = basehttp.NewCSRFError("").Write(c.Writer)
			return
		}
		cookie, err := authhttp.GetRefreshCookie(c.Request)
		if err != nil || cookie == "" {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		newCookie, accessJWT, _, err := refreshSvc.Rotate(c.Request.Context(), cookie, time.Now().UTC())
		if err != nil {
			_ = basehttp.NewUnauthorizedError("session expired").Write(c.Writer)
			return
		}
		authhttp.SetRefreshCookie(c.Writer, newCookie, cfg.RefreshTokenTTL, cookieCfg)
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"access_token": accessJWT})
	})

	// POST /logout
	rg.POST("/logout", func(c *gin.Context) {
		cookie, err := authhttp.GetRefreshCookie(c.Request)
		if err == nil && cookie != "" {
			_ = refreshSvc.RevokeCurrent(c.Request.Context(), cookie, "user logout")
		}
		authhttp.ClearRefreshCookie(c.Writer, cookieCfg)
		c.Status(http.StatusNoContent)
	})

	// POST /logout-all
	rg.POST("/logout-all", func(c *gin.Context) {
		actx, ok := From(c)
		if !ok || !actx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = refreshSvc.RevokeAllUser(c.Request.Context(), actx.UserID, "user logout all")
		c.Status(http.StatusNoContent)
	})

	// GET /me and /session - convenience endpoints
	rg.GET("/me", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{
			"id":    ctx.UserID.String(),
			"email": ctx.Email,
			"roles": ctx.Roles,
		})
	})

	rg.GET("/session", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{
			"valid":            true,
			"session_version":  ctx.SessionVersion,
			"step_up_required": ctx.RequiresStepUp(time.Now().UTC()),
			"step_up_until":    ctx.StepUpValidUntil,
		})
	})

	// TODO: add invites, onboarding, and recovery flows
	// POST /invites (admin)
	rg.POST("/invites", func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			Email    string   `json:"email"`
			Roles    []string `json:"roles"`
			TTLHours int      `json:"ttl_hours"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.TTLHours <= 0 {
			in.TTLHours = 168 // default 7 days
		}
		tokenBytes, err := deps.Rand.Bytes(32)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		hash := akcrypto.HashSHA256(tokenBytes)
		inv := &domain.Invite{
			ID:        uuid.New(),
			Email:     in.Email,
			Roles:     in.Roles,
			TokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(time.Duration(in.TTLHours) * time.Hour),
			CreatedBy: ctx.UserID,
		}
		if err := deps.Stores.Invites.Create(c.Request.Context(), inv); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		if deps.Mailer != nil {
			link := fmt.Sprintf("%s/onboard?invite_id=%s&token=%s", cfg.Origin, inv.ID.String(), token)
			_ = deps.Mailer.SendInviteEmail(c.Request.Context(), in.Email, link)
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusCreated, map[string]any{"invite_id": inv.ID.String(), "token": token})
	})

	// POST /onboard/begin
	rg.POST("/onboard/begin", func(c *gin.Context) {
		var in struct {
			InviteID   string `json:"invite_id"`
			Token      string `json:"token"`
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		invID, err := uuid.Parse(in.InviteID)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid invite id").Write(c.Writer)
			return
		}
		inv, err := deps.Stores.Invites.Get(c.Request.Context(), invID)
		if err != nil || inv == nil {
			_ = basehttp.NewBadRequestError("invalid invite").Write(c.Writer)
			return
		}
		if time.Now().UTC().After(inv.ExpiresAt) {
			_ = basehttp.NewBadRequestError("invite expired").Write(c.Writer)
			return
		}
		tokBytes, err := base64.RawURLEncoding.DecodeString(in.Token)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid token").Write(c.Writer)
			return
		}
		if subtle.ConstantTimeCompare(inv.TokenHash, akcrypto.HashSHA256(tokBytes)) != 1 {
			_ = deps.Stores.Invites.IncrementAttempts(c.Request.Context(), inv.ID)
			_ = basehttp.NewBadRequestError("invalid invite").Write(c.Writer)
			return
		}
		user, err := deps.Stores.Users.Create(c.Request.Context(), inv.Email, inv.Roles)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		requireHW := sliceContains(inv.Roles, "admin")
		options, sessionKey, err := webauthnSvc.BeginRegistration(c.Request.Context(), user, in.DeviceName, requireHW)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey, "user_id": user.ID.String()})
	})

	// POST /onboard/complete
	rg.POST("/onboard/complete", func(c *gin.Context) {
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
			InviteID   string          `json:"invite_id"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := webauthnSvc.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		if in.InviteID != "" {
			if invID, err := uuid.Parse(in.InviteID); err == nil {
				_ = deps.Stores.Invites.Redeem(c.Request.Context(), invID, time.Now().UTC())
			}
		}
		c.Status(http.StatusNoContent)
	})

	// Recovery: init
	rg.POST("/recovery/init", func(c *gin.Context) {
		var in struct {
			Email string `json:"email"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		flowID, err := recFlow.InitiateRecovery(c.Request.Context(), in.Email)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"flow_id": flowID})
	})

	// Recovery: verify code
	rg.POST("/recovery/verify", func(c *gin.Context) {
		var in struct {
			FlowID string `json:"flow_id"`
			Code   string `json:"code"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := recFlow.VerifyRecoveryCode(c.Request.Context(), in.FlowID, in.Code)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"flow_id": in.FlowID, "user_id": user.ID.String()})
	})

	// Recovery: register new credential begin
	rg.POST("/recovery/register/begin", func(c *gin.Context) {
		var in struct {
			FlowID     string `json:"flow_id"`
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		flow, err := recFlow.GetFlow(c.Request.Context(), in.FlowID)
		if err != nil || !flow.Verified || time.Now().UTC().After(flow.ExpiresAt) {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		user, err := deps.Stores.Users.GetByID(c.Request.Context(), flow.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		options, sessionKey, err := webauthnSvc.BeginRegistration(c.Request.Context(), user, in.DeviceName, false)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey})
	})

	// Recovery: register new credential complete
	rg.POST("/recovery/register/complete", func(c *gin.Context) {
		var in struct {
			FlowID     string          `json:"flow_id"`
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := webauthnSvc.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		if err := recFlow.CompleteRecovery(c.Request.Context(), in.FlowID); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})
}

// Handlers groups gin handlers for external route binding.
type Handlers struct {
	LoginBegin               gin.HandlerFunc
	LoginComplete            gin.HandlerFunc
	CredentialsList          gin.HandlerFunc
	CredentialsAddBegin      gin.HandlerFunc
	CredentialsAddComplete   gin.HandlerFunc
	CredentialsDelete        gin.HandlerFunc
	StepUpBegin              gin.HandlerFunc
	StepUpComplete           gin.HandlerFunc
	Refresh                  gin.HandlerFunc
	Logout                   gin.HandlerFunc
	LogoutAll                gin.HandlerFunc
	Me                       gin.HandlerFunc
	Session                  gin.HandlerFunc
	Invites                  gin.HandlerFunc
	OnboardBegin             gin.HandlerFunc
	OnboardComplete          gin.HandlerFunc
	RecoveryInit             gin.HandlerFunc
	RecoveryVerify           gin.HandlerFunc
	RecoveryRegisterBegin    gin.HandlerFunc
	RecoveryRegisterComplete gin.HandlerFunc
}

func buildHandlers(cfg Config, deps Deps) Handlers {
	// recreate the same closures used in registerRoutes but return them instead of mounting
	tokenSvc := service.NewTokenService(deps.Keys, deps.Rand, cfg.Origin, cfg.AccessTokenTTL)
	refreshSvc := service.NewRefreshService(service.RefreshServiceConfig{
		Store:      deps.Stores.Refresh,
		UserStore:  deps.Stores.Users,
		TokenSvc:   tokenSvc,
		Rand:       deps.Rand,
		TTL:        cfg.RefreshTokenTTL,
		AuditStore: deps.Stores.Audit,
	})
	cookieCfg := basehttp.DefaultCookieConfig()
	waCfg := service.WebAuthnConfig{
		RPDisplayName:        cfg.RPName,
		RPID:                 cfg.RPID,
		RPOrigins:            []string{cfg.Origin},
		Users:                deps.Stores.Users,
		Credentials:          deps.Stores.Credentials,
		Challenges:           deps.Stores.Challenges,
		Rand:                 deps.Rand,
		AdminAAGUIDAllowlist: cfg.AdminAAGUIDAllowlist,
		ChallengeTTL:         cfg.ChallengeTTL,
	}
	webauthnSvc, _ := service.NewWebAuthnService(waCfg)
	_ = service.NewStepUpService(deps.KV, cfg.StepUpTTL)
	recSvc := service.NewRecoveryService(deps.Stores.RecoveryCodes, deps.Rand)
	recFlow := service.NewRecoveryFlowService(deps.Stores.Users, recSvc, deps.KV, deps.Rand)

	h := Handlers{}
	h.LoginBegin = func(c *gin.Context) {
		options, sessionKey, err := webauthnSvc.BeginLogin(c.Request.Context(), "")
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey})
	}
	h.LoginComplete = func(c *gin.Context) {
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, _, err := webauthnSvc.FinishLogin(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("authentication failed").Write(c.Writer)
			return
		}
		cookieValue, _, _, err := refreshSvc.Issue(c.Request.Context(), user, time.Now().UTC())
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		authhttp.SetRefreshCookie(c.Writer, cookieValue, cfg.RefreshTokenTTL, cookieCfg)
		claims := service.AccessClaims{Sub: user.ID.String(), Email: user.Email, Roles: user.Roles, SessionVersion: user.SessionVersion}
		access, err := tokenSvc.SignAccess(c.Request.Context(), claims)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		// Also issue access cookie for browser flows
		authhttp.SetAccessCookie(c.Writer, access, cfg.AccessTokenTTL, cookieCfg)
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"user": map[string]any{"id": user.ID.String(), "email": user.Email, "roles": user.Roles}, "access_token": access})
	}
	h.CredentialsList = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		creds, err := deps.Stores.Credentials.GetByUser(c.Request.Context(), ctx.UserID)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		out := make([]map[string]any, 0, len(creds))
		for _, cd := range creds {
			out = append(out, map[string]any{"id": base64.RawURLEncoding.EncodeToString(cd.ID), "aaguid": cd.AAGUID, "device_name": cd.DeviceName, "sign_count": cd.SignCount, "last_used_at": cd.LastUsedAt})
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"credentials": out})
	}
	h.CredentialsAddBegin = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := deps.Stores.Users.GetByID(c.Request.Context(), ctx.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		requireHW := sliceContains(user.Roles, "admin")
		options, sessionKey, err := webauthnSvc.BeginRegistration(c.Request.Context(), user, in.DeviceName, requireHW)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey})
	}
	h.CredentialsAddComplete = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := webauthnSvc.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	}
	h.CredentialsDelete = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		idParam := c.Param("id")
		id, err := base64.RawURLEncoding.DecodeString(idParam)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid credential id").Write(c.Writer)
			return
		}
		creds, _ := deps.Stores.Credentials.GetByUser(c.Request.Context(), ctx.UserID)
		if len(creds) <= 1 {
			_ = basehttp.NewBadRequestError("cannot remove last credential").Write(c.Writer)
			return
		}
		if err := deps.Stores.Credentials.Revoke(c.Request.Context(), id); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	}
	h.StepUpBegin = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		user, err := deps.Stores.Users.GetByID(c.Request.Context(), ctx.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		options, sessionKey, err := webauthnSvc.BeginStepUp(c.Request.Context(), user, "")
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey})
	}
	h.StepUpComplete = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := webauthnSvc.FinishStepUp(c.Request.Context(), in.SessionKey, in.Credential)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("step-up failed").Write(c.Writer)
			return
		}
		until := time.Now().UTC().Add(cfg.StepUpTTL)
		claims := service.AccessClaims{Sub: user.ID.String(), Email: user.Email, Roles: user.Roles, SessionVersion: user.SessionVersion, StepUpUntil: until.Unix()}
		access, err := tokenSvc.SignAccess(c.Request.Context(), claims)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"access_token": access})
	}
	h.Refresh = func(c *gin.Context) {
		if err := deps.CSRF.Validate(c.Request); err != nil {
			_ = basehttp.NewCSRFError("").Write(c.Writer)
			return
		}
		cookie, err := authhttp.GetRefreshCookie(c.Request)
		if err != nil || cookie == "" {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		newCookie, accessJWT, _, err := refreshSvc.Rotate(c.Request.Context(), cookie, time.Now().UTC())
		if err != nil {
			_ = basehttp.NewUnauthorizedError("session expired").Write(c.Writer)
			return
		}
		authhttp.SetRefreshCookie(c.Writer, newCookie, cfg.RefreshTokenTTL, cookieCfg)
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"access_token": accessJWT})
	}
	h.Logout = func(c *gin.Context) {
		cookie, err := authhttp.GetRefreshCookie(c.Request)
		if err == nil && cookie != "" {
			_ = refreshSvc.RevokeCurrent(c.Request.Context(), cookie, "user logout")
		}
		authhttp.ClearRefreshCookie(c.Writer, cookieCfg)
		c.Status(http.StatusNoContent)
	}
	h.LogoutAll = func(c *gin.Context) {
		actx, ok := From(c)
		if !ok || !actx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = refreshSvc.RevokeAllUser(c.Request.Context(), actx.UserID, "user logout all")
		c.Status(http.StatusNoContent)
	}
	h.Me = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"id": ctx.UserID.String(), "email": ctx.Email, "roles": ctx.Roles})
	}
	h.Session = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"valid": true, "session_version": ctx.SessionVersion, "step_up_required": ctx.RequiresStepUp(time.Now().UTC()), "step_up_until": ctx.StepUpValidUntil})
	}
	h.Invites = func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		var in struct {
			Email    string   `json:"email"`
			Roles    []string `json:"roles"`
			TTLHours int      `json:"ttl_hours"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.TTLHours <= 0 {
			in.TTLHours = 168
		}
		tokenBytes, err := deps.Rand.Bytes(32)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		token := base64.RawURLEncoding.EncodeToString(tokenBytes)
		hash := akcrypto.HashSHA256(tokenBytes)
		inv := &domain.Invite{ID: uuid.New(), Email: in.Email, Roles: in.Roles, TokenHash: hash, ExpiresAt: time.Now().UTC().Add(time.Duration(in.TTLHours) * time.Hour), CreatedBy: ctx.UserID}
		if err := deps.Stores.Invites.Create(c.Request.Context(), inv); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		if deps.Mailer != nil {
			link := fmt.Sprintf("%s/onboard?invite_id=%s&token=%s", cfg.Origin, inv.ID.String(), token)
			_ = deps.Mailer.SendInviteEmail(c.Request.Context(), in.Email, link)
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusCreated, map[string]any{"invite_id": inv.ID.String(), "token": token})
	}
	h.OnboardBegin = func(c *gin.Context) {
		var in struct {
			InviteID   string `json:"invite_id"`
			Token      string `json:"token"`
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		invID, err := uuid.Parse(in.InviteID)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid invite id").Write(c.Writer)
			return
		}
		inv, err := deps.Stores.Invites.Get(c.Request.Context(), invID)
		if err != nil || inv == nil {
			_ = basehttp.NewBadRequestError("invalid invite").Write(c.Writer)
			return
		}
		if time.Now().UTC().After(inv.ExpiresAt) {
			_ = basehttp.NewBadRequestError("invite expired").Write(c.Writer)
			return
		}
		tokBytes, err := base64.RawURLEncoding.DecodeString(in.Token)
		if err != nil {
			_ = basehttp.NewBadRequestError("invalid token").Write(c.Writer)
			return
		}
		if subtle.ConstantTimeCompare(inv.TokenHash, akcrypto.HashSHA256(tokBytes)) != 1 {
			_ = deps.Stores.Invites.IncrementAttempts(c.Request.Context(), inv.ID)
			_ = basehttp.NewBadRequestError("invalid invite").Write(c.Writer)
			return
		}
		user, err := deps.Stores.Users.Create(c.Request.Context(), inv.Email, inv.Roles)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		requireHW := sliceContains(inv.Roles, "admin")
		options, sessionKey, err := webauthnSvc.BeginRegistration(c.Request.Context(), user, in.DeviceName, requireHW)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey, "user_id": user.ID.String()})
	}
	h.OnboardComplete = func(c *gin.Context) {
		var in struct {
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
			InviteID   string          `json:"invite_id"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := webauthnSvc.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		if in.InviteID != "" {
			if invID, err := uuid.Parse(in.InviteID); err == nil {
				_ = deps.Stores.Invites.Redeem(c.Request.Context(), invID, time.Now().UTC())
			}
		}
		c.Status(http.StatusNoContent)
	}
	h.RecoveryInit = func(c *gin.Context) {
		var in struct {
			Email string `json:"email"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		flowID, err := recFlow.InitiateRecovery(c.Request.Context(), in.Email)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"flow_id": flowID})
	}
	h.RecoveryVerify = func(c *gin.Context) {
		var in struct {
			FlowID string `json:"flow_id"`
			Code   string `json:"code"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := recFlow.VerifyRecoveryCode(c.Request.Context(), in.FlowID, in.Code)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"flow_id": in.FlowID, "user_id": user.ID.String()})
	}
	h.RecoveryRegisterBegin = func(c *gin.Context) {
		var in struct {
			FlowID     string `json:"flow_id"`
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		flow, err := recFlow.GetFlow(c.Request.Context(), in.FlowID)
		if err != nil || !flow.Verified || time.Now().UTC().After(flow.ExpiresAt) {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		user, err := deps.Stores.Users.GetByID(c.Request.Context(), flow.UserID)
		if err != nil || user == nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		options, sessionKey, err := webauthnSvc.BeginRegistration(c.Request.Context(), user, in.DeviceName, false)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, map[string]any{"publicKey": options, "session_key": sessionKey})
	}
	h.RecoveryRegisterComplete = func(c *gin.Context) {
		var in struct {
			FlowID     string          `json:"flow_id"`
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := webauthnSvc.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		if err := recFlow.CompleteRecovery(c.Request.Context(), in.FlowID); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	}
	return h
}

func sliceContains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// registerJWKS mounts the JWKS endpoint under /.well-known.
func registerJWKS(rg *gin.RouterGroup, deps Deps) {
	rg.GET("/jwks.json", func(c *gin.Context) { handlers.JWKSHandler(c.Writer, deps.Keys.JWKS()) })
}

// authenticateMiddleware returns the non-blocking auth middleware.
func authenticateMiddleware(cfg Config, deps Deps) gin.HandlerFunc {
	ts := newTokenServiceForMiddleware(cfg, deps)
	return func(c *gin.Context) {
		// Extract bearer token
		token, err := basehttp.GetBearerToken(c.Request)
		if err != nil || token == "" {
			// Try cookie-based access token
			if v, cerr := authhttp.GetAccessCookie(c.Request); cerr == nil && v != "" {
				token = v
			} else {
				if os.Getenv("TEST_LOG") == "1" {
					println("[authkit] no bearer token on", c.Request.Method, c.Request.URL.Path)
				}
				c.Next()
				return
			}
		}
		// Verify JWT
		claims, err := ts.ParseAccess(c.Request.Context(), token)
		if err != nil {
			if os.Getenv("TEST_LOG") == "1" {
				println("[authkit] ParseAccess failed:", err.Error())
			}
			c.Next()
			return
		}
		if claims.Issuer != cfg.Origin {
			if os.Getenv("TEST_LOG") == "1" {
				println("[authkit] issuer mismatch: got=", claims.Issuer, " want=", cfg.Origin)
			}
			c.Next()
			return
		}
		// Parse user ID
		userID, err := uuid.Parse(claims.Sub)
		if err != nil {
			c.Next()
			return
		}
		// Fetch user
		user, err := deps.Stores.Users.GetByID(c.Request.Context(), userID)
		if err != nil || user == nil {
			c.Next()
			return
		}
		// Check session version
		if user.SessionVersion != claims.SessionVersion {
			c.Next()
			return
		}
		// Step-up expiry
		var stepUp time.Time
		if claims.StepUpUntil > 0 {
			stepUp = time.Unix(claims.StepUpUntil, 0)
		}
		// Attach context
		ac := AuthContext{
			UserID:           userID,
			Email:            user.Email,
			FullName:         user.FullName,
			Roles:            user.Roles,
			Permissions:      user.Permissions,
			SessionVersion:   user.SessionVersion,
			StepUpValidUntil: stepUp,
			TokenID:          claims.JTI,
		}
		ac.Set(c)
		if os.Getenv("TEST_LOG") == "1" {
			println("[authkit] authenticated:", user.Email, "on", c.Request.Method, c.Request.URL.Path)
		}
		c.Next()
	}
}

// requireAuthMiddleware enforces authentication per-group.
func requireAuthMiddleware(_ Config, _ Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			basehttp.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// requireStepUpMiddleware enforces step-up per-group.
func requireStepUpMiddleware(_ Config, _ Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, ok := From(c)
		if !ok || !ctx.IsAuthenticated() {
			basehttp.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
			c.Abort()
			return
		}
		if ctx.RequiresStepUp(time.Now().UTC()) {
			basehttp.ErrorResponse(c.Writer, http.StatusPreconditionRequired, "step_up_required", "Step-up authentication required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// enforcePoliciesMiddleware returns global policy enforcer middleware.
func enforcePoliciesMiddleware(_ Config, _ Deps, provider PolicyProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rules []Rule
		if provider != nil {
			rules = provider.GetRules(c.Request.Method, c.Request.URL.Path)
		}
		if len(rules) == 0 {
			c.Next()
			return
		}
		rule := MergeRules(rules)
		if rule.RequireAuth || rule.RequireStepUp || len(rule.Roles) > 0 || len(rule.Permissions) > 0 {
			ctx, ok := From(c)
			if !ok || !ctx.IsAuthenticated() {
				basehttp.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
				c.Abort()
				return
			}
			// Enforce step-up first for everyone, including admin
			if rule.RequireStepUp && ctx.RequiresStepUp(time.Now().UTC()) {
				basehttp.ErrorResponse(c.Writer, http.StatusPreconditionRequired, "step_up_required", "Step-up authentication required")
				c.Abort()
				return
			}
			// Admin bypass for roles/permissions
			if ctx.HasRole("admin") {
				c.Next()
				return
			}
			if len(rule.Roles) > 0 && !ctx.HasAnyRole(rule.Roles) {
				basehttp.ErrorResponse(c.Writer, http.StatusForbidden, "forbidden", "Insufficient privileges")
				c.Abort()
				return
			}
			if len(rule.Permissions) > 0 && !ctx.HasAnyPermission(rule.Permissions) {
				basehttp.ErrorResponse(c.Writer, http.StatusForbidden, "forbidden", "Insufficient privileges")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
