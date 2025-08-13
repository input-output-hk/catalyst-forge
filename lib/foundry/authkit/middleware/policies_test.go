package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/middleware"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/rbac"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Mock RBAC Manager for testing
type mockRBACManager struct {
	checkDecision rbac.Decision
	checkError    error
	resolveRef    rbac.ResourceRef
	resolveOk     bool
	resolveError  error
}

func (m *mockRBACManager) WithPolicyRegistry(reg *authkit.PolicyRegistry) *authkit.PolicyRegistry {
	return reg
}

func (m *mockRBACManager) RegisterResolver(pattern string, resolver rbac.ResourceResolver) {
	// No-op for tests
}

func (m *mockRBACManager) Check(ctx context.Context, subj rbac.Subject, action rbac.PermissionKey, res rbac.ResourceRef) (rbac.Decision, error) {
	return m.checkDecision, m.checkError
}

func (m *mockRBACManager) Explain(ctx context.Context, subj rbac.Subject, action rbac.PermissionKey, res rbac.ResourceRef) (rbac.Decision, rbac.Trace, error) {
	return m.checkDecision, rbac.Trace{}, m.checkError
}

func (m *mockRBACManager) Resolve(c *gin.Context, path string) (rbac.ResourceRef, bool, error) {
	return m.resolveRef, m.resolveOk, m.resolveError
}

func TestNewPolicyEnforcer(t *testing.T) {
	t.Parallel()

	registry := authkit.NewPolicyRegistry()
	enforcer := middleware.NewPolicyEnforcer(registry)

	assert.NotNil(t, enforcer)
}

func TestPolicyEnforcer_WithRBAC(t *testing.T) {
	t.Parallel()

	registry := authkit.NewPolicyRegistry()
	enforcer := middleware.NewPolicyEnforcer(registry)
	rbacMgr := &mockRBACManager{}

	updatedEnforcer := enforcer.WithRBAC(rbacMgr)

	assert.Equal(t, enforcer, updatedEnforcer) // Should return same instance
}

func TestPolicyEnforcer_EnforcePolicies(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now().UTC()

	tests := []struct {
		name           string
		setupRegistry  func() *authkit.PolicyRegistry
		setupAuth      func() *authkit.AuthContext
		setupRBAC      func() *mockRBACManager
		method         string
		path           string
		expectedStatus int
	}{
		{
			name: "ok/no_rules_allow_request",
			setupRegistry: func() *authkit.PolicyRegistry {
				return authkit.NewPolicyRegistry()
			},
			setupAuth:      func() *authkit.AuthContext { return nil },
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "GET",
			path:           "/api/public",
			expectedStatus: http.StatusOK,
		},
		{
			name: "ok/authenticated_user_passes_auth_requirement",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequireAuth("GET", "/api/protected")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "test@example.com",
					Roles:            []string{"user"},
					Permissions:      []string{"read"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "GET",
			path:           "/api/protected",
			expectedStatus: http.StatusOK,
		},
		{
			name: "ok/user_with_stepup_passes_stepup_requirement",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequireStepUp("POST", "/admin/delete")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "admin@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write", "delete"},
					SessionVersion:   1,
					StepUpValidUntil: now.Add(time.Minute * 5), // Valid step-up
					TokenID:          "token-456",
				}
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "POST",
			path:           "/admin/delete",
			expectedStatus: http.StatusOK,
		},
		{
			name: "ok/user_with_required_role_passes",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequireRoles([]string{"admin", "moderator"}, "GET", "/admin/dashboard")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "admin@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-789",
				}
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "GET",
			path:           "/admin/dashboard",
			expectedStatus: http.StatusOK,
		},
		{
			name: "ok/user_with_required_permission_passes_without_rbac",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequirePermissions([]string{"read:users", "write:users"}, "POST", "/api/users")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"},
					Permissions:      []string{"read:users"}, // Has one of the required permissions
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "POST",
			path:           "/api/users",
			expectedStatus: http.StatusOK,
		},
		{
			name: "ok/user_with_permission_passes_with_rbac_allow",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequirePermissions([]string{"create:posts"}, "POST", "/api/posts")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"author"},
					Permissions:      []string{"read", "write"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-456",
				}
			},
			setupRBAC: func() *mockRBACManager {
				return &mockRBACManager{
					checkDecision: rbac.DecisionAllow,
					checkError:    nil,
					resolveRef:    rbac.ResourceRef{Type: "post", ID: ""},
					resolveOk:     true,
					resolveError:  nil,
				}
			},
			method:         "POST",
			path:           "/api/posts",
			expectedStatus: http.StatusOK,
		},
		{
			name: "error/no_auth_context_returns_401",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequireAuth("GET", "/api/protected")
				return registry
			},
			setupAuth:      func() *authkit.AuthContext { return nil },
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "GET",
			path:           "/api/protected",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error/unauthenticated_context_returns_401",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequireAuth("GET", "/api/protected")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{} // Empty/unauthenticated context
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "GET",
			path:           "/api/protected",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error/expired_stepup_returns_428",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequireStepUp("POST", "/admin/delete")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "admin@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write", "delete"},
					SessionVersion:   1,
					StepUpValidUntil: now.Add(-time.Minute), // Expired step-up
					TokenID:          "token-456",
				}
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "POST",
			path:           "/admin/delete",
			expectedStatus: http.StatusPreconditionRequired,
		},
		{
			name: "error/missing_required_role_returns_403",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequireRoles([]string{"admin"}, "GET", "/admin/dashboard")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"}, // Not admin
					Permissions:      []string{"read"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "GET",
			path:           "/admin/dashboard",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "error/missing_required_permission_returns_403_without_rbac",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequirePermissions([]string{"admin:delete"}, "DELETE", "/api/users/123")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"},
					Permissions:      []string{"read"}, // Missing admin:delete
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
			},
			setupRBAC:      func() *mockRBACManager { return nil },
			method:         "DELETE",
			path:           "/api/users/123",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "error/rbac_deny_returns_403",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequirePermissions([]string{"delete:users"}, "DELETE", "/api/users/123")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"},
					Permissions:      []string{"read", "write"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-456",
				}
			},
			setupRBAC: func() *mockRBACManager {
				return &mockRBACManager{
					checkDecision: rbac.DecisionDeny,
					checkError:    nil,
					resolveRef:    rbac.ResourceRef{Type: "user", ID: "123"},
					resolveOk:     true,
					resolveError:  nil,
				}
			},
			method:         "DELETE",
			path:           "/api/users/123",
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "error/rbac_stepup_required_returns_428",
			setupRegistry: func() *authkit.PolicyRegistry {
				registry := authkit.NewPolicyRegistry()
				registry.RequirePermissions([]string{"admin:deploy"}, "POST", "/admin/deploy")
				return registry
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "admin@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write", "admin"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{}, // No step-up
					TokenID:          "token-789",
				}
			},
			setupRBAC: func() *mockRBACManager {
				return &mockRBACManager{
					checkDecision: rbac.DecisionDeny,
					checkError:    rbac.ErrConditionStepUpRequired,
					resolveRef:    rbac.ResourceRef{Type: "deployment", ID: ""},
					resolveOk:     true,
					resolveError:  nil,
				}
			},
			method:         "POST",
			path:           "/admin/deploy",
			expectedStatus: http.StatusPreconditionRequired,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			// Setup registry and enforcer
			registry := tc.setupRegistry()
			enforcer := middleware.NewPolicyEnforcer(registry)

			// Setup RBAC if provided
			if rbacMgr := tc.setupRBAC(); rbacMgr != nil {
				enforcer = enforcer.WithRBAC(rbacMgr)
			}

			router := gin.New()

			// Set up auth context first if provided
			if authCtx := tc.setupAuth(); authCtx != nil {
				router.Use(func(c *gin.Context) {
					authCtx.Set(c)
					c.Next()
				})
			}

			// Apply policy enforcement middleware
			router.Use(enforcer.EnforcePolicies())

			router.Handle(tc.method, tc.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Make request
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestPolicyEnforcer_MergedRules(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	// Test merging multiple rules for the same endpoint
	registry := authkit.NewPolicyRegistry()
	registry.RequireAuth("POST", "/api/sensitive")
	registry.RequireRoles([]string{"admin"}, "POST", "/api/sensitive")
	registry.RequireStepUp("POST", "/api/sensitive")

	enforcer := middleware.NewPolicyEnforcer(registry)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up authenticated admin user without step-up
	router.Use(func(c *gin.Context) {
		authCtx := &authkit.AuthContext{
			UserID:           userID,
			Email:            "admin@example.com",
			Roles:            []string{"admin"},
			Permissions:      []string{"read", "write"},
			SessionVersion:   1,
			StepUpValidUntil: time.Time{}, // No step-up
			TokenID:          "token-123",
		}
		authCtx.Set(c)
		c.Next()
	})

	router.Use(enforcer.EnforcePolicies())

	router.POST("/api/sensitive", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/api/sensitive", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should require step-up since the merged rule has RequireStepUp = true
	assert.Equal(t, http.StatusPreconditionRequired, w.Code)
}

func TestRequirePolicy(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name           string
		rule           authkit.Rule
		setupAuth      func() *authkit.AuthContext
		expectedStatus int
	}{
		{
			name: "ok/no_requirements_passes",
			rule: authkit.Rule{}, // No requirements
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"},
					Permissions:      []string{"read"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "ok/authenticated_user_passes_auth_requirement",
			rule: authkit.Rule{RequireAuth: true},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"},
					Permissions:      []string{"read"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "ok/user_with_role_passes",
			rule: authkit.Rule{Roles: []string{"admin", "moderator"}},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "admin@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-456",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error/unauthenticated_returns_401",
			rule: authkit.Rule{RequireAuth: true},
			setupAuth: func() *authkit.AuthContext {
				return nil
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error/missing_role_returns_403",
			rule: authkit.Rule{Roles: []string{"admin"}},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"}, // Not admin
					Permissions:      []string{"read"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Set up auth context if provided
			if authCtx := tc.setupAuth(); authCtx != nil {
				router.Use(func(c *gin.Context) {
					authCtx.Set(c)
					c.Next()
				})
			}

			// Apply policy requirement
			router.Use(middleware.RequirePolicy(tc.rule))

			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestRequireRoles(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name           string
		requiredRoles  []string
		userRoles      []string
		expectedStatus int
	}{
		{
			name:           "ok/user_has_required_role",
			requiredRoles:  []string{"admin"},
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ok/user_has_one_of_multiple_required_roles",
			requiredRoles:  []string{"admin", "moderator"},
			userRoles:      []string{"moderator"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ok/user_has_multiple_roles_including_required",
			requiredRoles:  []string{"admin"},
			userRoles:      []string{"user", "admin", "editor"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error/user_missing_required_role",
			requiredRoles:  []string{"admin"},
			userRoles:      []string{"user"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "error/user_has_no_matching_roles",
			requiredRoles:  []string{"admin", "moderator"},
			userRoles:      []string{"user", "editor"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Set up authenticated user with specified roles
			router.Use(func(c *gin.Context) {
				authCtx := &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            tc.userRoles,
					Permissions:      []string{"read"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
				authCtx.Set(c)
				c.Next()
			})

			router.Use(middleware.RequireRoles(tc.requiredRoles...))

			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestRequireRoles_NoAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// No auth context set
	router.Use(middleware.RequireRoles("admin"))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermissions(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name               string
		requiredPermissions []string
		userPermissions     []string
		expectedStatus      int
	}{
		{
			name:               "ok/user_has_required_permission",
			requiredPermissions: []string{"read:users"},
			userPermissions:     []string{"read:users"},
			expectedStatus:      http.StatusOK,
		},
		{
			name:               "ok/user_has_one_of_multiple_required_permissions",
			requiredPermissions: []string{"read:users", "write:users"},
			userPermissions:     []string{"write:users"},
			expectedStatus:      http.StatusOK,
		},
		{
			name:               "ok/user_has_multiple_permissions_including_required",
			requiredPermissions: []string{"read:users"},
			userPermissions:     []string{"read", "read:users", "write"},
			expectedStatus:      http.StatusOK,
		},
		{
			name:               "error/user_missing_required_permission",
			requiredPermissions: []string{"write:users"},
			userPermissions:     []string{"read:users"},
			expectedStatus:      http.StatusForbidden,
		},
		{
			name:               "error/user_has_no_matching_permissions",
			requiredPermissions: []string{"admin:delete", "admin:create"},
			userPermissions:     []string{"read", "write"},
			expectedStatus:      http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Set up authenticated user with specified permissions
			router.Use(func(c *gin.Context) {
				authCtx := &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            []string{"user"},
					Permissions:      tc.userPermissions,
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
				authCtx.Set(c)
				c.Next()
			})

			router.Use(middleware.RequirePermissions(tc.requiredPermissions...))

			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestRequirePermissions_NoAuth(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// No auth context set
	router.Use(middleware.RequirePermissions("read:users"))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireAdmin(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name           string
		userRoles      []string
		expectedStatus int
	}{
		{
			name:           "ok/admin_user_passes",
			userRoles:      []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ok/admin_with_other_roles_passes",
			userRoles:      []string{"user", "admin", "editor"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error/non_admin_user_rejected",
			userRoles:      []string{"user"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "error/user_with_no_roles_rejected",
			userRoles:      []string{},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Set up authenticated user with specified roles
			router.Use(func(c *gin.Context) {
				authCtx := &authkit.AuthContext{
					UserID:           userID,
					Email:            "user@example.com",
					Roles:            tc.userRoles,
					Permissions:      []string{"read"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{},
					TokenID:          "token-123",
				}
				authCtx.Set(c)
				c.Next()
			})

			router.Use(middleware.RequireAdmin())

			router.GET("/admin", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/admin", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestRequireOneOf(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up authenticated user
	router.Use(func(c *gin.Context) {
		authCtx := &authkit.AuthContext{
			UserID:           userID,
			Email:            "user@example.com",
			Roles:            []string{"moderator"},
			Permissions:      []string{"read"},
			SessionVersion:   1,
			StepUpValidUntil: time.Time{},
			TokenID:          "token-123",
		}
		authCtx.Set(c)
		c.Next()
	})

	// RequireOneOf should work exactly like RequireRoles
	router.Use(middleware.RequireOneOf("admin", "moderator"))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBuildPolicyRegistry(t *testing.T) {
	t.Parallel()

	registry := middleware.BuildPolicyRegistry()

	assert.NotNil(t, registry)

	// Test that it includes some common patterns
	rules := registry.GetRules("GET", "/admin/users")
	assert.NotEmpty(t, rules, "Should have rules for admin endpoints")

	rules = registry.GetRules("GET", "/profile")
	assert.NotEmpty(t, rules, "Should have rules for profile endpoints")

	rules = registry.GetRules("POST", "/profile/delete")
	assert.NotEmpty(t, rules, "Should have rules for sensitive operations")

	// Check that sensitive operations require step-up
	found := false
	for _, rule := range rules {
		if rule.RequireStepUp {
			found = true
			break
		}
	}
	assert.True(t, found, "Sensitive operations should require step-up")
}

func TestPolicyEnforcer_WildcardMethod(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	// Test wildcard method matching
	registry := authkit.NewPolicyRegistry()
	registry.RequireAuth("*", "/api/protected/*") // Any method

	enforcer := middleware.NewPolicyEnforcer(registry)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up authenticated user
	router.Use(func(c *gin.Context) {
		authCtx := &authkit.AuthContext{
			UserID:           userID,
			Email:            "user@example.com",
			Roles:            []string{"user"},
			Permissions:      []string{"read"},
			SessionVersion:   1,
			StepUpValidUntil: time.Time{},
			TokenID:          "token-123",
		}
		authCtx.Set(c)
		c.Next()
	})

	router.Use(enforcer.EnforcePolicies())

	// Add handlers for different methods
	router.GET("/api/protected/data", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.POST("/api/protected/data", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Test GET
	req := httptest.NewRequest("GET", "/api/protected/data", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test POST
	req = httptest.NewRequest("POST", "/api/protected/data", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPolicyEnforcer_PathMatching(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name          string
		pattern       string
		testPath      string
		shouldMatch   bool
	}{
		{
			name:        "exact_match",
			pattern:     "/api/users",
			testPath:    "/api/users",
			shouldMatch: true,
		},
		{
			name:        "exact_no_match",
			pattern:     "/api/users",
			testPath:    "/api/posts",
			shouldMatch: false,
		},
		{
			name:        "prefix_wildcard_match_exact",
			pattern:     "/api/users/*",
			testPath:    "/api/users",
			shouldMatch: true,
		},
		{
			name:        "prefix_wildcard_match_subpath",
			pattern:     "/api/users/*",
			testPath:    "/api/users/123",
			shouldMatch: true,
		},
		{
			name:        "prefix_wildcard_match_deep_subpath",
			pattern:     "/api/users/*",
			testPath:    "/api/users/123/profile",
			shouldMatch: true,
		},
		{
			name:        "prefix_wildcard_no_match_similar_prefix",
			pattern:     "/api/users/*",
			testPath:    "/api/userprofile",
			shouldMatch: false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry := authkit.NewPolicyRegistry()
			registry.RequireAuth("GET", tc.pattern)

			enforcer := middleware.NewPolicyEnforcer(registry)

			gin.SetMode(gin.TestMode)
			router := gin.New()

			// Set up authenticated user
			authCtx := &authkit.AuthContext{
				UserID:           userID,
				Email:            "user@example.com",
				Roles:            []string{"user"},
				Permissions:      []string{"read"},
				SessionVersion:   1,
				StepUpValidUntil: time.Time{},
				TokenID:          "token-123",
			}

			if tc.shouldMatch {
				// If should match, provide auth context
				router.Use(func(c *gin.Context) {
					authCtx.Set(c)
					c.Next()
				})
			}
			// If shouldn't match, don't provide auth context (it should pass anyway)

			router.Use(enforcer.EnforcePolicies())

			router.GET(tc.testPath, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", tc.testPath, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// If the pattern should match, we expect 200 (auth provided)
			// If the pattern shouldn't match, we expect 200 (no auth required)
			assert.Equal(t, http.StatusOK, w.Code)

			// Now test without auth context when pattern should match
			if tc.shouldMatch {
				router2 := gin.New()
				router2.Use(enforcer.EnforcePolicies())
				router2.GET(tc.testPath, func(c *gin.Context) {
					c.Status(http.StatusOK)
				})

				req2 := httptest.NewRequest("GET", tc.testPath, nil)
				w2 := httptest.NewRecorder()
				router2.ServeHTTP(w2, req2)

				// Should be unauthorized since pattern matches but no auth provided
				assert.Equal(t, http.StatusUnauthorized, w2.Code)
			}
		})
	}
}

func TestPolicyEnforcer_EmptyAuth(t *testing.T) {
	t.Parallel()

	// Test that empty/invalid auth context is handled properly
	registry := authkit.NewPolicyRegistry()
	registry.RequireAuth("GET", "/protected")

	enforcer := middleware.NewPolicyEnforcer(registry)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set up empty auth context (simulates authentication failure)
	router.Use(func(c *gin.Context) {
		authCtx := &authkit.AuthContext{} // Empty context
		authCtx.Set(c)
		c.Next()
	})

	router.Use(enforcer.EnforcePolicies())

	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}