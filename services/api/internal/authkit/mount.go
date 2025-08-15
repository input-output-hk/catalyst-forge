package authkit

import (
	"context"

	libauth "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/gin-gonic/gin"
)

// Mount wires AuthKit routes and middlewares onto the given engine.
// The caller is responsible for providing a valid Manager via libauth.New.
func Mount(_ context.Context, r *gin.Engine, manager libauth.Manager, policies *libauth.PolicyRegistry) {
	// Mount /api/v1/auth endpoints
	manager.RegisterRoutes(r.Group("/api/v1/auth"))
	// JWKS at /.well-known if enabled should be mounted by caller via manager.RegisterJWKS
	// Apply global middlewares
	r.Use(manager.Authenticate())
	if policies != nil {
		r.Use(manager.EnforcePolicies(policies))
	}
}
