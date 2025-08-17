package auth

import (
	"encoding/json"
	"net/http"
	"time"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/crypto"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

type RecoveryDeps struct {
	Flow  akservice.RecoveryFlowService
	Store store.RecoveryCodeStore
	Rand  crypto.Rand
}

// RegisterRecovery binds recovery init/verify/register handlers and codes generation.
func RegisterRecovery(r *gin.Engine, deps RecoveryDeps, wa akservice.WebAuthnService) {
	g := r.Group("/api/v1/auth")

	// @Summary Recovery init
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.RecoveryInitRequest true "Recovery init request"
	// @Success 200 {object} apimodels.RecoveryInitResponse
	// @Router /api/v1/auth/recovery/init [post]
	g.POST("/recovery/init", func(c *gin.Context) {
		var in struct {
			Email string `json:"email"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		flowID, err := deps.Flow.InitiateRecovery(c.Request.Context(), in.Email)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.RecoveryInitResponse{FlowID: flowID})
	})

	// @Summary Recovery verify
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.RecoveryVerifyRequest true "Recovery verify request"
	// @Success 200 {object} apimodels.RecoveryVerifyResponse
	// @Router /api/v1/auth/recovery/verify [post]
	g.POST("/recovery/verify", func(c *gin.Context) {
		var in struct {
			FlowID string `json:"flow_id"`
			Code   string `json:"code"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		user, err := deps.Flow.VerifyRecoveryCode(c.Request.Context(), in.FlowID, in.Code)
		if err != nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.RecoveryVerifyResponse{FlowID: in.FlowID, UserID: user.ID.String()})
	})

	// @Summary Recovery register begin
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body apimodels.RecoveryRegisterBeginRequest true "Recovery register begin request"
	// @Success 200 {object} apimodels.PublicKeyOptionsResponse
	// @Router /api/v1/auth/recovery/register/begin [post]
	g.POST("/recovery/register/begin", func(c *gin.Context) {
		var in struct {
			FlowID     string `json:"flow_id"`
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		flow, err := deps.Flow.GetFlow(c.Request.Context(), in.FlowID)
		if err != nil || !flow.Verified || flow.ExpiresAt.Before(time.Now().UTC()) {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		user, err := deps.Flow.VerifyRecoveryCode(c.Request.Context(), in.FlowID, "")
		_ = user // assumes Verify returns user when verified; otherwise load by flow.UserID
		if err != nil {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}
		options, sessionKey, err := wa.BeginRegistration(c.Request.Context(), user, in.DeviceName, false)
		if err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.PublicKeyOptionsResponse{PublicKey: options, SessionKey: sessionKey})
	})

	// @Summary Recovery register complete
	// @Tags auth
	// @Accept json
	// @Param request body apimodels.RecoveryRegisterCompleteRequest true "Recovery register complete request"
	// @Success 204
	// @Router /api/v1/auth/recovery/register/complete [post]
	g.POST("/recovery/register/complete", func(c *gin.Context) {
		var in struct {
			FlowID     string          `json:"flow_id"`
			SessionKey string          `json:"session_key"`
			Credential json.RawMessage `json:"credential"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if _, err := wa.FinishRegistration(c.Request.Context(), in.SessionKey, in.Credential); err != nil {
			_ = basehttp.NewBadRequestError("registration failed").Write(c.Writer)
			return
		}
		if err := deps.Flow.CompleteRecovery(c.Request.Context(), in.FlowID); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary Generate new recovery codes (one-time view)
	// @Tags auth
	// @Produce json
	// @Success 200 {object} apimodels.RecoveryGenerateResponse
	// @Router /api/v1/auth/recovery/codes/generate [post]
	g.POST("/recovery/codes/generate", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
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
		if err := deps.Store.ReplaceCodes(c.Request.Context(), ctx.UserID, hashes); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.RecoveryGenerateResponse{Codes: codes})
	})
}
