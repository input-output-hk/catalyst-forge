package auth

import (
	"net/http"

	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	akservice "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/service"
)

type ghaExchangeRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

func RegisterGithubExchange(r *gin.Engine, svc akservice.GHAExchangeService) {
	// @Summary Exchange GitHub OIDC token
	// @Tags auth
	// @Accept json
	// @Produce json
	// @Param request body ghaExchangeRequest true "GitHub OIDC exchange request"
	// @Success 200 {object} map[string]interface{}
	// @Router /api/v1/auth/oidc/github/exchange [post]
	r.POST("/api/v1/auth/oidc/github/exchange", func(c *gin.Context) {
		var req ghaExchangeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			_ = basehttp.NewBadRequestError("invalid request").Write(c.Writer)
			return
		}
		access, expiresIn, subj, err := svc.Exchange(c.Request.Context(), req.IDToken)
		if err != nil {
			// Conservatively return 400 until verifier implements detailed errors
			basehttp.ErrorResponse(c.Writer, http.StatusBadRequest, "invalid_token", "invalid or unsupported token")
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"access_token": access,
			"expires_in":   expiresIn,
			"subject":      subj,
		})
	})
}
