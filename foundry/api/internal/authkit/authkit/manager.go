package authkit

import (
	"github.com/gin-gonic/gin"
)

// Manager is the main interface for the authentication system.
type Manager interface {
	// RegisterRoutes mounts all /auth endpoints on the provided router group.
	RegisterRoutes(rg *gin.RouterGroup)

	// RegisterJWKS mounts the /.well-known/jwks.json endpoint (optional).
	RegisterJWKS(rg *gin.RouterGroup)

	// Authenticate is middleware that parses/validates access JWT and injects AuthContext.
	//
	// This is non-blocking - it attaches auth context if a valid token is present.
	Authenticate() gin.HandlerFunc

	// EnforcePolicies is global middleware that enforces path+method-based authorization rules.
	// It uses the provided PolicyProvider. If nil, a default in-memory registry may be used.
	EnforcePolicies(provider PolicyProvider) gin.HandlerFunc

	// RequireAuth is per-group middleware that requires authentication.
	RequireAuth() gin.HandlerFunc

	// RequireStepUp is per-group middleware that requires recent WebAuthn authentication.
	RequireStepUp() gin.HandlerFunc

	// Handlers returns the set of HTTP handlers for all auth endpoints so that
	// consumers can bind them to their own routes for better documentation control.
	Handlers() Handlers
}
