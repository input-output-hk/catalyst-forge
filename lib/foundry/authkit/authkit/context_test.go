package authkit

import (
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthContext_SetAndRetrieve(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		ctx  AuthContext
	}{
		{
			name: "complete_context",
			ctx: AuthContext{
				UserID:           uuid.New(),
				Email:            "user@example.com",
				Roles:            []string{"user", "admin"},
				Permissions:      []string{"read", "write", "delete"},
				SessionVersion:   42,
				StepUpValidUntil: time.Now().Add(time.Hour),
				TokenID:          "jwt-token-id",
			},
		},
		{
			name: "minimal_context",
			ctx: AuthContext{
				UserID:         uuid.New(),
				Email:          "minimal@example.com",
				Roles:          []string{"user"},
				SessionVersion: 1,
			},
		},
		{
			name: "empty_context",
			ctx:  AuthContext{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(nil)

			// Test Set method on context
			tt.ctx.Set(c)

			// Test From retrieval
			retrieved, exists := From(c)
			assert.True(t, exists, "context should exist after setting")
			assert.Equal(t, tt.ctx, retrieved, "retrieved context should match original")

			// Test Must retrieval (should not panic)
			mustRetrieved := Must(c)
			assert.Equal(t, tt.ctx, mustRetrieved, "Must should return same context as From")
		})
	}
}

func TestAuthContext_From_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Test From with no context set
	ctx, exists := From(c)
	assert.False(t, exists, "should not find context when none is set")
	assert.Equal(t, AuthContext{}, ctx, "should return empty context when not found")
}

func TestAuthContext_Must_Panic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Test Must panics when no context is set
	assert.Panics(t, func() {
		Must(c)
	}, "Must should panic when no context is found")
}

func TestAuthContext_From_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	// Set wrong type in context
	c.Set(contextKey, "not-an-auth-context")

	// Test From with wrong type
	ctx, exists := From(c)
	assert.False(t, exists, "should not find context when wrong type is set")
	assert.Equal(t, AuthContext{}, ctx, "should return empty context for wrong type")
}

func TestAuthContext_IsAuthenticated(t *testing.T) {
	tests := []struct {
		name           string
		ctx            AuthContext
		wantAuth       bool
		description    string
	}{
		{
			name: "authenticated_user",
			ctx: AuthContext{
				UserID: uuid.New(),
				Email:  "user@example.com",
			},
			wantAuth:    true,
			description: "User with valid UUID should be authenticated",
		},
		{
			name: "unauthenticated_user",
			ctx: AuthContext{
				UserID: uuid.Nil,
				Email:  "",
			},
			wantAuth:    false,
			description: "User with nil UUID should not be authenticated",
		},
		{
			name:        "empty_context",
			ctx:         AuthContext{},
			wantAuth:    false,
			description: "Empty context should not be authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ctx.IsAuthenticated()
			assert.Equal(t, tt.wantAuth, result, tt.description)
		})
	}
}

func TestAuthContext_HasRole(t *testing.T) {
	ctx := AuthContext{
		UserID: uuid.New(),
		Email:  "user@example.com",
		Roles:  []string{"user", "admin", "moderator"},
	}

	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{name: "has_user_role", role: "user", expected: true},
		{name: "has_admin_role", role: "admin", expected: true},
		{name: "has_moderator_role", role: "moderator", expected: true},
		{name: "missing_role", role: "superuser", expected: false},
		{name: "empty_role", role: "", expected: false},
		{name: "case_sensitive", role: "USER", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.HasRole(tt.role)
			assert.Equal(t, tt.expected, result, "role check for %q should be %v", tt.role, tt.expected)
		})
	}
}

func TestAuthContext_HasRole_EmptyRoles(t *testing.T) {
	ctx := AuthContext{
		UserID: uuid.New(),
		Email:  "user@example.com",
		Roles:  []string{},
	}

	assert.False(t, ctx.HasRole("user"), "should not have any role when roles slice is empty")
	assert.False(t, ctx.HasRole("admin"), "should not have any role when roles slice is empty")
}

func TestAuthContext_HasAnyRole(t *testing.T) {
	ctx := AuthContext{
		UserID: uuid.New(),
		Email:  "user@example.com",
		Roles:  []string{"user", "viewer", "editor"},
	}

	tests := []struct {
		name     string
		roles    []string
		expected bool
	}{
		{
			name:     "has_one_matching_role",
			roles:    []string{"admin", "user", "superuser"},
			expected: true,
		},
		{
			name:     "has_multiple_matching_roles",
			roles:    []string{"user", "editor", "admin"},
			expected: true,
		},
		{
			name:     "no_matching_roles",
			roles:    []string{"admin", "superuser", "guest"},
			expected: false,
		},
		{
			name:     "empty_roles_list",
			roles:    []string{},
			expected: false,
		},
		{
			name:     "single_matching_role",
			roles:    []string{"viewer"},
			expected: true,
		},
		{
			name:     "single_non_matching_role",
			roles:    []string{"admin"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.HasAnyRole(tt.roles)
			assert.Equal(t, tt.expected, result, "role check for %v should be %v", tt.roles, tt.expected)
		})
	}
}

func TestAuthContext_HasPermission(t *testing.T) {
	ctx := AuthContext{
		UserID:      uuid.New(),
		Email:       "user@example.com",
		Permissions: []string{"read", "write", "delete", "admin:users"},
	}

	tests := []struct {
		name       string
		permission string
		expected   bool
	}{
		{name: "has_read_permission", permission: "read", expected: true},
		{name: "has_write_permission", permission: "write", expected: true},
		{name: "has_delete_permission", permission: "delete", expected: true},
		{name: "has_namespaced_permission", permission: "admin:users", expected: true},
		{name: "missing_permission", permission: "execute", expected: false},
		{name: "empty_permission", permission: "", expected: false},
		{name: "case_sensitive", permission: "READ", expected: false},
		{name: "partial_match", permission: "rea", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.HasPermission(tt.permission)
			assert.Equal(t, tt.expected, result, "permission check for %q should be %v", tt.permission, tt.expected)
		})
	}
}

func TestAuthContext_HasPermission_EmptyPermissions(t *testing.T) {
	ctx := AuthContext{
		UserID:      uuid.New(),
		Email:       "user@example.com",
		Permissions: []string{},
	}

	assert.False(t, ctx.HasPermission("read"), "should not have any permission when permissions slice is empty")
	assert.False(t, ctx.HasPermission("write"), "should not have any permission when permissions slice is empty")
}

func TestAuthContext_HasAnyPermission(t *testing.T) {
	ctx := AuthContext{
		UserID:      uuid.New(),
		Email:       "user@example.com",
		Permissions: []string{"read", "write", "admin:view"},
	}

	tests := []struct {
		name        string
		permissions []string
		expected    bool
	}{
		{
			name:        "has_one_matching_permission",
			permissions: []string{"execute", "read", "admin:delete"},
			expected:    true,
		},
		{
			name:        "has_multiple_matching_permissions",
			permissions: []string{"read", "write", "delete"},
			expected:    true,
		},
		{
			name:        "no_matching_permissions",
			permissions: []string{"execute", "admin:delete", "superuser"},
			expected:    false,
		},
		{
			name:        "empty_permissions_list",
			permissions: []string{},
			expected:    false,
		},
		{
			name:        "single_matching_permission",
			permissions: []string{"admin:view"},
			expected:    true,
		},
		{
			name:        "single_non_matching_permission",
			permissions: []string{"admin:delete"},
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ctx.HasAnyPermission(tt.permissions)
			assert.Equal(t, tt.expected, result, "permission check for %v should be %v", tt.permissions, tt.expected)
		})
	}
}

func TestAuthContext_RequiresStepUp(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name             string
		stepUpValidUntil time.Time
		checkTime        time.Time
		expected         bool
		description      string
	}{
		{
			name:             "step_up_still_valid",
			stepUpValidUntil: now.Add(30 * time.Minute),
			checkTime:        now,
			expected:         false,
			description:      "Should not require step-up when still within valid period",
		},
		{
			name:             "step_up_expired",
			stepUpValidUntil: now.Add(-30 * time.Minute),
			checkTime:        now,
			expected:         true,
			description:      "Should require step-up when past valid period",
		},
		{
			name:             "step_up_exactly_expired",
			stepUpValidUntil: now,
			checkTime:        now.Add(time.Nanosecond),
			expected:         true,
			description:      "Should require step-up when exactly at expiry",
		},
		{
			name:             "no_step_up_time_set",
			stepUpValidUntil: time.Time{}, // Zero time
			checkTime:        now,
			expected:         true,
			description:      "Should require step-up when no step-up time is set",
		},
		{
			name:             "far_future_step_up",
			stepUpValidUntil: now.Add(24 * time.Hour),
			checkTime:        now,
			expected:         false,
			description:      "Should not require step-up when far in the future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := AuthContext{
				UserID:           uuid.New(),
				Email:            "user@example.com",
				StepUpValidUntil: tt.stepUpValidUntil,
			}

			result := ctx.RequiresStepUp(tt.checkTime)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestSetContext_ConvenienceFunction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(nil)

	ctx := AuthContext{
		UserID:         uuid.New(),
		Email:          "test@example.com",
		Roles:          []string{"user"},
		SessionVersion: 1,
	}

	// Test SetContext convenience function
	SetContext(c, ctx)

	// Verify it was set correctly
	retrieved, exists := From(c)
	assert.True(t, exists, "context should exist after using SetContext")
	assert.Equal(t, ctx, retrieved, "retrieved context should match original")
}

func TestAuthContext_ConcurrentAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test concurrent access to context methods
	ctx := AuthContext{
		UserID:           uuid.New(),
		Email:            "concurrent@example.com",
		Roles:            []string{"user", "admin", "moderator"},
		Permissions:      []string{"read", "write", "delete", "admin:all"},
		SessionVersion:   42,
		StepUpValidUntil: time.Now().Add(time.Hour),
	}

	numGoroutines := 100
	numOperations := 50

	var wg sync.WaitGroup
	results := make(chan bool, numGoroutines*numOperations)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Test various concurrent operations
				switch j % 8 {
				case 0:
					results <- ctx.IsAuthenticated()
				case 1:
					results <- ctx.HasRole("user")
				case 2:
					results <- ctx.HasRole("admin")
				case 3:
					results <- ctx.HasAnyRole([]string{"user", "guest"})
				case 4:
					results <- ctx.HasPermission("read")
				case 5:
					results <- ctx.HasPermission("write")
				case 6:
					results <- ctx.HasAnyPermission([]string{"read", "execute"})
				case 7:
					results <- ctx.RequiresStepUp(time.Now())
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)

	// Verify all operations completed successfully
	resultCount := 0
	for range results {
		resultCount++
	}

	expectedResults := numGoroutines * numOperations
	assert.Equal(t, expectedResults, resultCount, "all concurrent operations should complete")
}

func TestAuthContext_ConcurrentSetAndRetrieve(t *testing.T) {
	gin.SetMode(gin.TestMode)

	numGoroutines := 50
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			// Each goroutine gets its own gin context
			c, _ := gin.CreateTestContext(nil)

			ctx := AuthContext{
				UserID:         uuid.New(),
				Email:          "concurrent-test@example.com",
				Roles:          []string{string(rune('A' + goroutineID%26))}, // Different roles
				SessionVersion: int64(goroutineID),
			}

			// Set context
			ctx.Set(c)

			// Retrieve and verify
			retrieved, exists := From(c)
			if !exists {
				errors <- assert.AnError
				return
			}

			if retrieved.UserID != ctx.UserID || retrieved.SessionVersion != ctx.SessionVersion {
				errors <- assert.AnError
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	var errorList []error
	for err := range errors {
		errorList = append(errorList, err)
	}

	assert.Empty(t, errorList, "no errors should occur during concurrent set/retrieve operations")
}

func TestAuthContext_StepUpValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name             string
		stepUpValidUntil time.Time
		description      string
	}{
		{
			name:             "unix_epoch",
			stepUpValidUntil: time.Unix(0, 0),
			description:      "Unix epoch time should require step-up",
		},
		{
			name:             "far_past",
			stepUpValidUntil: time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC),
			description:      "Far past time should require step-up",
		},
		{
			name:             "far_future",
			stepUpValidUntil: time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
			description:      "Far future time should not require step-up",
		},
	}

	now := time.Now()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := AuthContext{
				UserID:           uuid.New(),
				StepUpValidUntil: tt.stepUpValidUntil,
			}

			result := ctx.RequiresStepUp(now)

			if tt.stepUpValidUntil.IsZero() || now.After(tt.stepUpValidUntil) {
				assert.True(t, result, tt.description)
			} else {
				assert.False(t, result, tt.description)
			}
		})
	}
}

func TestAuthContext_FieldImmutability(t *testing.T) {
	// Test that modifying slices doesn't affect the original context
	originalRoles := []string{"user", "admin"}
	originalPermissions := []string{"read", "write"}

	ctx := AuthContext{
		UserID:      uuid.New(),
		Email:       "test@example.com",
		Roles:       originalRoles,
		Permissions: originalPermissions,
	}

	// Create copies and modify them
	modifiedRoles := make([]string, len(ctx.Roles))
	copy(modifiedRoles, ctx.Roles)
	_ = append(modifiedRoles, "hacker") // Explicitly ignore to test immutability

	modifiedPermissions := make([]string, len(ctx.Permissions))
	copy(modifiedPermissions, ctx.Permissions)
	_ = append(modifiedPermissions, "exploit") // Explicitly ignore to test immutability

	// Original context should be unchanged
	assert.Equal(t, originalRoles, ctx.Roles, "original roles should be unchanged")
	assert.Equal(t, originalPermissions, ctx.Permissions, "original permissions should be unchanged")
	assert.False(t, ctx.HasRole("hacker"), "should not have the added role")
	assert.False(t, ctx.HasPermission("exploit"), "should not have the added permission")
}