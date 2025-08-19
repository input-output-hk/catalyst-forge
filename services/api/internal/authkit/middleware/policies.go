package middleware

import (
	"net/http"
	"time"

	basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
)

// PolicyEnforcer provides policy-based authorization middleware.
type PolicyEnforcer struct {
	registry *authkit.PolicyRegistry
	rbacMgr  rbac.Manager // optional; if nil, fallback to AuthContext permissions
}

// NewPolicyEnforcer creates a new policy enforcement middleware.
func NewPolicyEnforcer(registry *authkit.PolicyRegistry) *PolicyEnforcer {
	return &PolicyEnforcer{
		registry: registry,
	}
}

// WithRBAC attaches an RBAC manager to the enforcer.
func (pe *PolicyEnforcer) WithRBAC(m rbac.Manager) *PolicyEnforcer {
	pe.rbacMgr = m
	return pe
}

// EnforcePolicies evaluates authorization rules for the current request.
//
// This middleware:
//   - Looks up all matching rules for the request path and method
//   - Merges the rules to determine the most restrictive requirements
//   - Enforces authentication, step-up, role, and permission requirements
//
// Returns appropriate HTTP status codes:
//   - 401 Unauthorized: No authentication present when required
//   - 403 Forbidden: Authenticated but lacks required roles/permissions
//   - 404 Not Found: Route not declared in policy registry (fail-closed)
//   - 428 Precondition Required: Step-up authentication needed
func (pe *PolicyEnforcer) EnforcePolicies() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get all matching rules for this request
		rules := pe.registry.GetRules(c.Request.Method, c.Request.URL.Path)
		if len(rules) == 0 {
			// No rules apply → fail closed
			basehttpkit.ErrorResponse(c.Writer, http.StatusNotFound, "not_found", "Route not found")
			c.Abort()
			return
		}

		// If any rule is explicitly open (AllowAnonymous), allow
		for _, r := range rules {
			if !r.RequireAuth && !r.RequireStepUp && len(r.Roles) == 0 && len(r.Permissions) == 0 && r.Explicit {
				c.Next()
				return
			}
		}

		// Merge all matching rules
		rule := authkit.MergeRules(rules)

		// Check authentication requirement
		if rule.RequireAuth || rule.RequireStepUp || len(rule.Roles) > 0 || len(rule.Permissions) > 0 {
			// Authentication is required
			ctx, ok := authkit.From(c)
			if !ok || !ctx.IsAuthenticated() {
				basehttpkit.ErrorResponse(c.Writer, http.StatusUnauthorized, "unauthorized", "Authentication required")
				c.Abort()
				return
			}

			// Check step-up requirement FIRST
			if rule.RequireStepUp && ctx.RequiresStepUp(time.Now().UTC()) {
				basehttpkit.ErrorResponse(c.Writer, http.StatusPreconditionRequired, "step_up_required", "Step-up authentication required")
				c.Abort()
				return
			}

			// Check role requirement
			if len(rule.Roles) > 0 && !ctx.HasAnyRole(rule.Roles) {
				basehttpkit.ErrorResponse(c.Writer, http.StatusForbidden, "forbidden", "Insufficient privileges")
				c.Abort()
				return
			}

			// Check permission requirement
			if len(rule.Permissions) > 0 {
				if pe.rbacMgr == nil {
					if !ctx.HasAnyPermission(rule.Permissions) {
						basehttpkit.ErrorResponse(c.Writer, http.StatusForbidden, "forbidden", "Insufficient privileges")
						c.Abort()
						return
					}
				} else {
					// Build subject from AuthContext
					subj := rbac.Subject{Type: rbac.SubjectUser, ID: ctx.UserID.String(), Attrs: map[string]any{"roles": ctx.Roles}}
					// Resolve resource if a resolver was registered for exact path
					res, _, _ := pe.rbacMgr.Resolve(c, c.Request.URL.Path)

					// attach request meta for contextual planning
					ctxWithMeta := rbac.WithRequestMeta(c.Request.Context(), c.FullPath(), c.Request.Method)
					for _, p := range rule.Permissions {
						dec, err := pe.rbacMgr.Check(ctxWithMeta, subj, rbac.PermissionKey(p), res)
						if err == rbac.ErrConditionStepUpRequired {
							basehttpkit.ErrorResponse(c.Writer, http.StatusPreconditionRequired, "step_up_required", "Step-up authentication required")
							c.Abort()
							return
						}
						if dec != rbac.DecisionAllow {
							basehttpkit.ErrorResponse(c.Writer, http.StatusForbidden, "forbidden", "Insufficient privileges")
							c.Abort()
							return
						}
					}
				}
			}
		}

		c.Next()
	}
}

// BuildPolicyRegistry creates a standard policy registry with common patterns.
//
// This is a helper for applications to quickly set up common authorization patterns.
func BuildPolicyRegistry() *authkit.PolicyRegistry {
	registry := authkit.NewPolicyRegistry()

	// Example common patterns (applications should customize):
	// Admin endpoints
	registry.RequireRoles([]string{"admin"}, "*", "/admin/*")

	// User profile endpoints
	registry.RequireAuth("*", "/profile", "/profile/*")

	// Sensitive operations require step-up
	registry.RequireStepUp("POST", "/profile/delete")
	registry.RequireStepUp("PUT", "/profile/email")
	registry.RequireStepUp("POST", "/credentials/*")

	return registry
}
