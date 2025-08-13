package rbac

import (
	"context"
	"fmt"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	"github.com/gin-gonic/gin"
)

// Deps bundles external dependencies required by RBAC.
type Deps struct {
	Store  Store
	Clock  authkit.Clock
	Logger authkit.Logger
	Audit  any // optional; align with store.AuditStore when integrated
	Cache  Cache
}

// Manager provides the public RBAC facade and integration points.
type Manager interface {
	// WithPolicyRegistry returns the same registry for fluent wiring.
	WithPolicyRegistry(reg *authkit.PolicyRegistry) *authkit.PolicyRegistry

	// RegisterResolver binds a path pattern to a resource resolver.
	RegisterResolver(pattern string, resolver ResourceResolver)

	// Programmatic checks
	Check(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, error)
	Explain(ctx context.Context, subj Subject, action PermissionKey, res ResourceRef) (Decision, Trace, error)

	// Resolve attempts to resolve a ResourceRef for the given request path.
	// Returns (ref, true, nil) if resolved, (zero, false, nil) if no resolver matches.
	Resolve(c *gin.Context, path string) (ResourceRef, bool, error)
}

// New constructs a new RBAC manager. Implementation added in later phases.
func New(cfg Config, deps Deps) (Manager, error) {
	if deps.Store == nil {
		return nil, fmt.Errorf("rbac: missing Store dependency")
	}
	return newManager(cfg, deps), nil
}
