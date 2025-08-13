package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/middleware"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/rate"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Mock rate limiter for testing
type mockRateLimiter struct {
	allowResult     bool
	remainingResult int
	resetResult     time.Time
	errorResult     error
	lastKey         rate.Key
	lastRequests    int
	lastWindow      time.Duration
}

func (m *mockRateLimiter) Allow(ctx context.Context, key rate.Key, n int, per time.Duration) (bool, int, time.Time, error) {
	m.lastKey = key
	m.lastRequests = n
	m.lastWindow = per
	return m.allowResult, m.remainingResult, m.resetResult, m.errorResult
}

func TestNewRateLimiter(t *testing.T) {
	t.Parallel()

	limiter := &mockRateLimiter{}
	rl := middleware.NewRateLimiter(limiter)

	assert.NotNil(t, rl)
}

func TestRateLimiter_LimitByIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		setupLimiter     func() *mockRateLimiter
		clientIP         string
		method           string
		path             string
		expectedStatus   int
		expectHeaders    bool
		expectedKey      string
	}{
		{
			name: "ok/request_allowed",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 9,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			clientIP:       "192.168.1.100",
			method:         "GET",
			path:           "/api/data",
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
			expectedKey:    "ip:192.168.1.100:GET:/api/data",
		},
		{
			name: "error/request_rate_limited",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Now().Add(5 * time.Minute),
					errorResult:     nil,
				}
			},
			clientIP:       "192.168.1.100",
			method:         "POST",
			path:           "/api/submit",
			expectedStatus: http.StatusTooManyRequests,
			expectHeaders:  true,
			expectedKey:    "ip:192.168.1.100:POST:/api/submit",
		},
		{
			name: "ok/limiter_error_allows_request",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     errors.New("rate limiter error"),
				}
			},
			clientIP:       "192.168.1.100",
			method:         "GET",
			path:           "/api/data",
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
		{
			name: "ok/fallback_to_remote_addr",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 5,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			clientIP:       "", // Empty client IP triggers fallback
			method:         "GET",
			path:           "/api/test",
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			limiter := tc.setupLimiter()
			rl := middleware.NewRateLimiter(limiter)

			router := gin.New()
			router.Use(rl.LimitByIP(10, time.Hour))

			router.Handle(tc.method, tc.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.clientIP != "" {
				req.Header.Set("X-Forwarded-For", tc.clientIP)
			} else {
				req.RemoteAddr = "127.0.0.1:12345"
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectHeaders {
				assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
				assert.Equal(t, strconv.Itoa(limiter.remainingResult), w.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

				if tc.expectedStatus == http.StatusTooManyRequests {
					assert.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}

			if limiter.errorResult == nil && tc.expectedKey != "" {
				assert.Equal(t, rate.Key(tc.expectedKey), limiter.lastKey)
				assert.Equal(t, 10, limiter.lastRequests)
				assert.Equal(t, time.Hour, limiter.lastWindow)
			}
		})
	}
}

func TestRateLimiter_LimitByUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	tests := []struct {
		name           string
		setupLimiter   func() *mockRateLimiter
		setupAuth      func() *authkit.AuthContext
		method         string
		path           string
		expectedStatus int
		expectHeaders  bool
		expectedKey    string
	}{
		{
			name: "ok/authenticated_user_allowed",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 15,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
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
			method:         "GET",
			path:           "/api/profile",
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
			expectedKey:    "user:" + userID.String() + ":GET:/api/profile",
		},
		{
			name: "error/authenticated_user_rate_limited",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Now().Add(10 * time.Minute),
					errorResult:     nil,
				}
			},
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
			method:         "POST",
			path:           "/api/action",
			expectedStatus: http.StatusTooManyRequests,
			expectHeaders:  true,
			expectedKey:    "user:" + userID.String() + ":POST:/api/action",
		},
		{
			name: "ok/no_auth_context_skips_limiting",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false, // Would be denied if checked
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     nil,
				}
			},
			setupAuth:      func() *authkit.AuthContext { return nil },
			method:         "GET",
			path:           "/api/public",
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
		{
			name: "ok/unauthenticated_context_skips_limiting",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false, // Would be denied if checked
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     nil,
				}
			},
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{} // Empty/unauthenticated context
			},
			method:         "GET",
			path:           "/api/public",
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
		{
			name: "ok/limiter_error_allows_request",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     errors.New("rate limiter error"),
				}
			},
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
			method:         "GET",
			path:           "/api/data",
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			limiter := tc.setupLimiter()
			rl := middleware.NewRateLimiter(limiter)

			router := gin.New()

			// Set up auth context if provided
			if authCtx := tc.setupAuth(); authCtx != nil {
				router.Use(func(c *gin.Context) {
					authCtx.Set(c)
					c.Next()
				})
			}

			router.Use(rl.LimitByUser(20, time.Hour))

			router.Handle(tc.method, tc.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectHeaders {
				assert.Equal(t, "20", w.Header().Get("X-RateLimit-Limit"))
				assert.Equal(t, strconv.Itoa(limiter.remainingResult), w.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

				if tc.expectedStatus == http.StatusTooManyRequests {
					assert.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}

			if limiter.errorResult == nil && tc.expectedKey != "" {
				assert.Equal(t, rate.Key(tc.expectedKey), limiter.lastKey)
				assert.Equal(t, 20, limiter.lastRequests)
				assert.Equal(t, time.Hour, limiter.lastWindow)
			}
		})
	}
}

func TestRateLimiter_LimitByKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupLimiter   func() *mockRateLimiter
		keyFunc        func(*gin.Context) string
		expectedStatus int
		expectHeaders  bool
		expectedKey    string
	}{
		{
			name: "ok/custom_key_allowed",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 7,
					resetResult:     time.Now().Add(30 * time.Minute),
					errorResult:     nil,
				}
			},
			keyFunc: func(c *gin.Context) string {
				return "custom:api_key:" + c.GetHeader("X-API-Key")
			},
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
			expectedKey:    "custom:api_key:test-api-key",
		},
		{
			name: "error/custom_key_rate_limited",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Now().Add(15 * time.Minute),
					errorResult:     nil,
				}
			},
			keyFunc: func(c *gin.Context) string {
				return "tenant:" + c.GetHeader("X-Tenant-ID")
			},
			expectedStatus: http.StatusTooManyRequests,
			expectHeaders:  true,
			expectedKey:    "tenant:tenant-123",
		},
		{
			name: "ok/empty_key_skips_limiting",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false, // Would be denied if checked
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     nil,
				}
			},
			keyFunc: func(c *gin.Context) string {
				return "" // Empty key
			},
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
		{
			name: "ok/limiter_error_allows_request",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     errors.New("rate limiter error"),
				}
			},
			keyFunc: func(c *gin.Context) string {
				return "key:test"
			},
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			limiter := tc.setupLimiter()
			rl := middleware.NewRateLimiter(limiter)

			router := gin.New()
			router.Use(rl.LimitByKey(tc.keyFunc, 10, time.Hour))

			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-API-Key", "test-api-key")
			req.Header.Set("X-Tenant-ID", "tenant-123")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectHeaders {
				assert.Equal(t, "10", w.Header().Get("X-RateLimit-Limit"))
				assert.Equal(t, strconv.Itoa(limiter.remainingResult), w.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

				if tc.expectedStatus == http.StatusTooManyRequests {
					assert.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}

			if limiter.errorResult == nil && tc.expectedKey != "" {
				assert.Equal(t, rate.Key(tc.expectedKey), limiter.lastKey)
				assert.Equal(t, 10, limiter.lastRequests)
				assert.Equal(t, time.Hour, limiter.lastWindow)
			}
		})
	}
}

func TestDefaultAuthLimits(t *testing.T) {
	t.Parallel()

	limits := middleware.DefaultAuthLimits()

	assert.NotNil(t, limits.Login)
	assert.Equal(t, 5, limits.Login.Requests)
	assert.Equal(t, 15*time.Minute, limits.Login.Window)

	assert.NotNil(t, limits.InviteUse)
	assert.Equal(t, 5, limits.InviteUse.Requests)
	assert.Equal(t, time.Duration(0), limits.InviteUse.Window) // Lifetime limit

	assert.NotNil(t, limits.RecoveryInit)
	assert.Equal(t, 3, limits.RecoveryInit.Requests)
	assert.Equal(t, time.Hour, limits.RecoveryInit.Window)

	assert.NotNil(t, limits.RecoveryVerify)
	assert.Equal(t, 5, limits.RecoveryVerify.Requests)
	assert.Equal(t, time.Hour, limits.RecoveryVerify.Window)

	assert.NotNil(t, limits.Refresh)
	assert.Equal(t, 100, limits.Refresh.Requests)
	assert.Equal(t, time.Hour, limits.Refresh.Window)

	assert.NotNil(t, limits.CredentialsAdd)
	assert.Equal(t, 10, limits.CredentialsAdd.Requests)
	assert.Equal(t, time.Hour, limits.CredentialsAdd.Window)
}

func TestRateLimiter_ApplyAuthLimits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		endpoint       string
		setupLimiter   func() *mockRateLimiter
		expectedStatus int
		expectLimiting bool
	}{
		{
			name:     "ok/login_endpoint_with_limits",
			endpoint: "login",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 4,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectLimiting: true,
		},
		{
			name:     "error/login_endpoint_rate_limited",
			endpoint: "login",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusTooManyRequests,
			expectLimiting: true,
		},
		{
			name:     "ok/invite_use_endpoint_no_limits",
			endpoint: "invite_use",
			setupLimiter: func() *mockRateLimiter {
				// Should not be called for invite_use
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectLimiting: false, // Invite limits handled by service
		},
		{
			name:     "ok/recovery_init_endpoint_with_limits",
			endpoint: "recovery_init",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 2,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectLimiting: true,
		},
		{
			name:     "ok/recovery_verify_endpoint_with_limits",
			endpoint: "recovery_verify",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 4,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectLimiting: true,
		},
		{
			name:     "ok/refresh_endpoint_with_limits",
			endpoint: "refresh",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 99,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectLimiting: true,
		},
		{
			name:     "ok/credentials_add_endpoint_with_limits",
			endpoint: "credentials_add",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 9,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectLimiting: true,
		},
		{
			name:     "ok/unknown_endpoint_no_limits",
			endpoint: "unknown",
			setupLimiter: func() *mockRateLimiter {
				// Should not be called for unknown endpoints
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectLimiting: false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			limiter := tc.setupLimiter()
			rl := middleware.NewRateLimiter(limiter)
			limits := middleware.DefaultAuthLimits()

			router := gin.New()
			router.Use(rl.ApplyAuthLimits(tc.endpoint, limits))

			router.POST("/auth/endpoint", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("POST", "/auth/endpoint", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.100") // For IP-based limiting

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectLimiting {
				// Should have rate limit headers
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
			}
		})
	}
}

func TestRateLimiter_ApplyAuthLimits_CustomLimits(t *testing.T) {
	t.Parallel()

	// Test with custom limits including nil limit
	customLimits := middleware.AuthEndpointLimits{
		Login: nil, // No limit for login
		Refresh: &middleware.RateLimit{
			Requests: 50,
			Window:   30 * time.Minute,
		},
	}

	gin.SetMode(gin.TestMode)

	limiter := &mockRateLimiter{
		allowResult:     false, // Would be denied if checked
		remainingResult: 0,
		resetResult:     time.Time{},
		errorResult:     nil,
	}
	rl := middleware.NewRateLimiter(limiter)

	router := gin.New()
	router.Use(rl.ApplyAuthLimits("login", customLimits)) // No limit configured

	router.POST("/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should pass because no limit is configured for login
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimiter_ApplyAuthLimits_ZeroRequests(t *testing.T) {
	t.Parallel()

	// Test with zero requests (disabled limit)
	customLimits := middleware.AuthEndpointLimits{
		Login: &middleware.RateLimit{
			Requests: 0, // Disabled
			Window:   time.Hour,
		},
	}

	gin.SetMode(gin.TestMode)

	limiter := &mockRateLimiter{
		allowResult:     false, // Would be denied if checked
		remainingResult: 0,
		resetResult:     time.Time{},
		errorResult:     nil,
	}
	rl := middleware.NewRateLimiter(limiter)

	router := gin.New()
	router.Use(rl.ApplyAuthLimits("login", customLimits))

	router.POST("/auth/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/auth/login", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should pass because limit is disabled (0 requests)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimiter_GlobalRateLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupLimiter   func() *mockRateLimiter
		expectedStatus int
		expectHeaders  bool
		expectedKey    string
	}{
		{
			name: "ok/global_limit_allowed",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 999,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
			expectedKey:    "global",
		},
		{
			name: "error/global_limit_exceeded",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			expectedStatus: http.StatusTooManyRequests,
			expectHeaders:  true,
			expectedKey:    "global",
		},
		{
			name: "ok/limiter_error_allows_request",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     errors.New("global limiter error"),
				}
			},
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			limiter := tc.setupLimiter()
			rl := middleware.NewRateLimiter(limiter)

			router := gin.New()
			router.Use(rl.GlobalRateLimit(1000, time.Hour))

			router.GET("/api/global", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/api/global", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectHeaders {
				assert.Equal(t, "1000", w.Header().Get("X-RateLimit-Limit"))
				assert.Equal(t, strconv.Itoa(limiter.remainingResult), w.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

				if tc.expectedStatus == http.StatusTooManyRequests {
					assert.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}

			if limiter.errorResult == nil && tc.expectedKey != "" {
				assert.Equal(t, rate.Key(tc.expectedKey), limiter.lastKey)
				assert.Equal(t, 1000, limiter.lastRequests)
				assert.Equal(t, time.Hour, limiter.lastWindow)
			}
		})
	}
}

func TestRateLimiter_PerRouteGlobalRateLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupLimiter   func() *mockRateLimiter
		method         string
		path           string
		expectedStatus int
		expectHeaders  bool
		expectedKey    string
	}{
		{
			name: "ok/per_route_limit_allowed",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 49,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			method:         "GET",
			path:           "/api/users",
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
			expectedKey:    "global:GET:/api/users",
		},
		{
			name: "error/per_route_limit_exceeded",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Now().Add(30 * time.Minute),
					errorResult:     nil,
				}
			},
			method:         "POST",
			path:           "/api/posts",
			expectedStatus: http.StatusTooManyRequests,
			expectHeaders:  true,
			expectedKey:    "global:POST:/api/posts",
		},
		{
			name: "ok/different_routes_have_separate_limits",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     true,
					remainingResult: 50,
					resetResult:     time.Now().Add(time.Hour),
					errorResult:     nil,
				}
			},
			method:         "GET",
			path:           "/api/posts", // Different from previous test
			expectedStatus: http.StatusOK,
			expectHeaders:  true,
			expectedKey:    "global:GET:/api/posts",
		},
		{
			name: "ok/limiter_error_allows_request",
			setupLimiter: func() *mockRateLimiter {
				return &mockRateLimiter{
					allowResult:     false,
					remainingResult: 0,
					resetResult:     time.Time{},
					errorResult:     errors.New("per-route limiter error"),
				}
			},
			method:         "GET",
			path:           "/api/test",
			expectedStatus: http.StatusOK,
			expectHeaders:  false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			limiter := tc.setupLimiter()
			rl := middleware.NewRateLimiter(limiter)

			router := gin.New()
			router.Use(rl.PerRouteGlobalRateLimit(50, time.Hour))

			router.Handle(tc.method, tc.path, func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)

			if tc.expectHeaders {
				assert.Equal(t, "50", w.Header().Get("X-RateLimit-Limit"))
				assert.Equal(t, strconv.Itoa(limiter.remainingResult), w.Header().Get("X-RateLimit-Remaining"))
				assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))

				if tc.expectedStatus == http.StatusTooManyRequests {
					assert.NotEmpty(t, w.Header().Get("Retry-After"))
				}
			}

			if limiter.errorResult == nil && tc.expectedKey != "" {
				assert.Equal(t, rate.Key(tc.expectedKey), limiter.lastKey)
				assert.Equal(t, 50, limiter.lastRequests)
				assert.Equal(t, time.Hour, limiter.lastWindow)
			}
		})
	}
}

func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	t.Parallel()

	// Test retry-after header calculation
	gin.SetMode(gin.TestMode)

	now := time.Now()
	resetTime := now.Add(5*time.Minute + 30*time.Second) // 5.5 minutes from now

	limiter := &mockRateLimiter{
		allowResult:     false,
		remainingResult: 0,
		resetResult:     resetTime,
		errorResult:     nil,
	}
	rl := middleware.NewRateLimiter(limiter)

	router := gin.New()
	router.Use(rl.LimitByIP(10, time.Hour))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	retryAfterStr := w.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfterStr)

	retryAfter, err := strconv.Atoi(retryAfterStr)
	assert.NoError(t, err)

	// Should be approximately 331 seconds (5.5 minutes), but allow some variance for test execution time
	assert.True(t, retryAfter >= 330 && retryAfter <= 335, "Retry-After should be around 331 seconds, got %d", retryAfter)
}

func TestRateLimiter_RetryAfterHeader_PastReset(t *testing.T) {
	t.Parallel()

	// Test retry-after header when reset time is in the past
	gin.SetMode(gin.TestMode)

	resetTime := time.Now().Add(-time.Minute) // 1 minute ago

	limiter := &mockRateLimiter{
		allowResult:     false,
		remainingResult: 0,
		resetResult:     resetTime,
		errorResult:     nil,
	}
	rl := middleware.NewRateLimiter(limiter)

	router := gin.New()
	router.Use(rl.LimitByIP(10, time.Hour))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	retryAfterStr := w.Header().Get("Retry-After")
	assert.Equal(t, "0", retryAfterStr) // Should be 0 when reset time is in the past
}

func TestRateLimiter_HeadersFormat(t *testing.T) {
	t.Parallel()

	// Test that all rate limit headers are set correctly
	gin.SetMode(gin.TestMode)

	resetTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	limiter := &mockRateLimiter{
		allowResult:     true,
		remainingResult: 42,
		resetResult:     resetTime,
		errorResult:     nil,
	}
	rl := middleware.NewRateLimiter(limiter)

	router := gin.New()
	router.Use(rl.LimitByIP(100, time.Hour))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "42", w.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, "1704110400", w.Header().Get("X-RateLimit-Reset")) // Unix timestamp for resetTime
}

// countingMockLimiter tracks the number of Allow calls
type countingMockLimiter struct {
	allowResult     bool
	remainingResult int
	resetResult     time.Time
	errorResult     error
	callCount       int
}

func (c *countingMockLimiter) Allow(ctx context.Context, key rate.Key, n int, per time.Duration) (bool, int, time.Time, error) {
	c.callCount++
	return c.allowResult, c.remainingResult, c.resetResult, c.errorResult
}

func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	// Test that the rate limiter works correctly with concurrent requests
	gin.SetMode(gin.TestMode)

	limiter := &countingMockLimiter{
		allowResult:     true,
		remainingResult: 10,
		resetResult:     time.Now().Add(time.Hour),
		errorResult:     nil,
		callCount:       0,
	}

	rl := middleware.NewRateLimiter(limiter)

	router := gin.New()
	router.Use(rl.LimitByIP(100, time.Hour))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Make multiple concurrent requests
	numRequests := 5
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("X-Forwarded-For", "192.168.1.100")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// All requests should have called the limiter
	assert.Equal(t, numRequests, limiter.callCount)
}