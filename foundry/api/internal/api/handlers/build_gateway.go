package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BuildGatewayAuthorizeRequest is a minimal request body for gateway ext_authz
// Accepts a SAN (DNS name) and an optional required prefix policy, both will be compared
// using a simple prefix rule. In a future revision, we can wire DB-backed policies.
type BuildGatewayAuthorizeRequest struct {
	SAN    string `json:"san" binding:"required"`
	Policy string `json:"policy_prefix"`
}

// AuthorizeBuildGateway provides a simple feature-flagged authorization check for BuildKit gateway
// It returns 200 if allowed, 403 otherwise. When disabled, it returns 404 to avoid leaking behavior.
func (h *CertificateHandler) AuthorizeBuildGateway(c *gin.Context) {
	enabledAny, ok := c.Get("feature_ext_authz_enabled")
	if !ok || enabledAny == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ext_authz disabled"})
		return
	}
	enabled, _ := enabledAny.(bool)
	if !enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "ext_authz disabled"})
		return
	}

	var req BuildGatewayAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Simple prefix policy: if policy is empty, allow; else require SAN to start with policy
	if req.Policy == "" || strings.HasPrefix(req.SAN, req.Policy) {
		c.JSON(http.StatusOK, gin.H{"allowed": true})
		return
	}

	c.JSON(http.StatusForbidden, gin.H{"allowed": false})
}
