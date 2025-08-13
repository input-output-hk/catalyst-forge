package authkit

import (
	"net/http"
	"time"

	libauth "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	basehttp "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
)

// Session returns a lightweight view of the current session state.
func Session(c *gin.Context) {
	ctx, ok := libauth.From(c)
	if !ok || !ctx.IsAuthenticated() {
		_ = basehttp.NewUnauthorizedError("").Write(c.Writer)
		return
	}
	now := time.Now().UTC()
	session := map[string]any{
		"valid":            true,
		"session_version":  ctx.SessionVersion,
		"step_up_required": ctx.RequiresStepUp(now),
		"step_up_until":    ctx.StepUpValidUntil,
	}
	_ = basehttp.WriteJSON(c.Writer, http.StatusOK, session)
}
