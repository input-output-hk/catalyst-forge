package auth

import (
	"encoding/base64"
	"net/http"
	"os"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	apimodels "github.com/input-output-hk/catalyst-forge/services/api/internal/api/models/auth"
	akauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/store"
)

type CredentialsDeps struct {
	Credentials store.CredentialStore
}

func RegisterCredentials(r *gin.Engine, deps CredentialsDeps) {
	g := r.Group("/api/v1/auth")

	// @Summary List credentials
	// @Tags auth
	// @Produce json
	// @Success 200 {object} apimodels.CredentialsListResponse
	// @Router /api/v1/auth/credentials [get]
	g.GET("/credentials", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
		if os.Getenv("TEST_LOG") == "1" {
			if ok {
				println("[handler] credentials: auth ctx present for", ctx.Email)
			} else {
				println("[handler] credentials: no auth ctx")
			}
		}
		if !ok || !ctx.IsAuthenticated() {
			_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
			return
		}

		creds, err := deps.Credentials.GetByUser(c.Request.Context(), ctx.UserID)
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
		if os.Getenv("TEST_LOG") == "1" {
			println("[handler] credentials: returning", len(out), "items for", ctx.Email)
		}
		_ = basehttp.WriteJSON(c.Writer, http.StatusOK, apimodels.CredentialsListResponse{Credentials: out})
	})

	// @Summary Delete credential
	// @Tags auth
	// @Param credentialId path string true "Credential ID"
	// @Success 204
	// @Router /api/v1/auth/credentials/{credentialId} [delete]
	g.DELETE("/credentials/:id", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
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

		creds, _ := deps.Credentials.GetByUser(c.Request.Context(), ctx.UserID)
		if len(creds) <= 1 {
			_ = basehttp.NewBadRequestError("cannot remove last credential").Write(c.Writer)
			return
		}

		if err := deps.Credentials.Revoke(c.Request.Context(), id); err != nil {
			_ = basehttp.NewInternalError().Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})

	// @Summary Update credential device name
	// @Tags auth
	// @Param id path string true "Credential ID (base64url)"
	// @Accept json
	// @Produce json
	// @Success 204
	// @Router /api/v1/auth/credentials/{id} [patch]
	g.PATCH("/credentials/:id", func(c *gin.Context) {
		ctx, ok := akauth.From(c)
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

		var in struct {
			DeviceName string `json:"device_name"`
		}
		if err := basehttp.ParseJSON(c.Writer, c.Request, &in); err != nil {
			return
		}
		if in.DeviceName == "" {
			_ = basehttp.NewBadRequestError("device_name is required").Write(c.Writer)
			return
		}

		if err := deps.Credentials.UpdateDeviceName(c.Request.Context(), id, ctx.UserID, in.DeviceName); err != nil {
			_ = basehttp.NewBadRequestError(err.Error()).Write(c.Writer)
			return
		}
		c.Status(http.StatusNoContent)
	})
}
