package authkit

import (
	"net/http"

	libauth "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
)

// Me returns the current authenticated user's profile and roles.
func Me(c *gin.Context) {
	ctx, ok := libauth.From(c)
	if !ok || !ctx.IsAuthenticated() {
		_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
		return
	}
	resp := map[string]any{
		"id":    ctx.UserID.String(),
		"email": ctx.Email,
		"roles": ctx.Roles,
	}
	_ = basehttp.WriteJSON(c.Writer, http.StatusOK, resp)
}
