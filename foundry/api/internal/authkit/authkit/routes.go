package authkit

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// New creates a new authentication manager with the provided configuration and dependencies.
func New(cfg Config, deps Deps) (Manager, error) {
	// Validate minimal deps
	if deps.Keys == nil {
		return nil, errors.New("missing Keys dependency")
	}
	if deps.Stores.Users == nil || deps.Stores.Credentials == nil || deps.Stores.Refresh == nil || deps.Stores.Challenges == nil {
		return nil, errors.New("missing required stores")
	}
	if deps.CSRF == nil {
		return nil, errors.New("missing CSRF dependency")
	}

	// Return a thin manager that delegates to provided middlewares and handlers.
	// Handlers and middlewares are assumed to be implemented elsewhere in this package.
	m := &manager{cfg: cfg, deps: deps}
	return m, nil
}

type manager struct {
	cfg  Config
	deps Deps
}

func (m *manager) RegisterRoutes(rg *gin.RouterGroup) { registerRoutes(rg, m.cfg, m.deps) }
func (m *manager) RegisterJWKS(rg *gin.RouterGroup)   { registerJWKS(rg, m.deps) }
func (m *manager) Authenticate() gin.HandlerFunc      { return authenticateMiddleware(m.cfg, m.deps) }
func (m *manager) EnforcePolicies(provider PolicyProvider) gin.HandlerFunc {
	return enforcePoliciesMiddleware(m.cfg, m.deps, provider)
}
func (m *manager) RequireAuth() gin.HandlerFunc   { return requireAuthMiddleware(m.cfg, m.deps) }
func (m *manager) RequireStepUp() gin.HandlerFunc { return requireStepUpMiddleware(m.cfg, m.deps) }
func (m *manager) Handlers() Handlers             { return buildHandlers(m.cfg, m.deps) }
