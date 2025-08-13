package routes

import (
	"context"

	libauth "github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	"github.com/gin-gonic/gin"
	apiAuthHandlers "github.com/input-output-hk/catalyst-forge/foundry/api/internal/api/handlers/authkit"
	apiauth "github.com/input-output-hk/catalyst-forge/foundry/api/internal/authkit"
)

// RegisterAuthKit mounts AuthKit routes and auxiliary API-owned auth endpoints.
func RegisterAuthKit(r *gin.Engine, m libauth.Manager, cfg libauth.Config) {
	// Mount library-provided routes and middlewares
	apiauth.Mount(context.Background(), r, m, apiauth.BuildPolicies())
	if cfg.JWKSRoute {
		m.RegisterJWKS(r.Group("/.well-known"))
	}

	// API-owned helper endpoints
	r.GET("/api/v1/auth/me", apiAuthHandlers.Me)
	r.GET("/api/v1/auth/session", apiAuthHandlers.Session)
}
