package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/authkit"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/domain"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/middleware"
	"github.com/catalystgo/catalyst-forge/lib/foundry/authkit/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock TokenService for testing
type mockTokenService struct {
	parseResult *service.AccessClaims
	parseError  error
}

func (m *mockTokenService) SignAccess(ctx context.Context, claims service.AccessClaims) (string, error) {
	return "mock-token", nil
}

func (m *mockTokenService) ParseAccess(ctx context.Context, token string) (*service.AccessClaims, error) {
	if m.parseError != nil {
		return nil, m.parseError
	}
	return m.parseResult, nil
}

func (m *mockTokenService) JWKS() interface{} {
	return map[string]interface{}{"keys": []interface{}{}}
}

// Mock UserStore for testing
type mockUserStore struct {
	user  *domain.User
	error error
}

func (m *mockUserStore) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.user, nil
}

func (m *mockUserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.user, m.error
}

func (m *mockUserStore) Create(ctx context.Context, email string, roles []string) (*domain.User, error) {
	if m.error != nil {
		return nil, m.error
	}
	return m.user, nil
}

func (m *mockUserStore) UpdateRoles(ctx context.Context, userID uuid.UUID, roles []string) error {
	return m.error
}

func (m *mockUserStore) BumpSessionVersion(ctx context.Context, userID uuid.UUID) error {
	return m.error
}

func TestNewAuthenticator(t *testing.T) {
	t.Parallel()
	
	tokenService := &mockTokenService{}
	userStore := &mockUserStore{}
	issuer := "test-issuer"
	
	auth := middleware.NewAuthenticator(tokenService, userStore, issuer)
	
	assert.NotNil(t, auth)
}

func TestAuthenticator_Authenticate(t *testing.T) {
	t.Parallel()
	
	userID := uuid.New()
	tokenID := uuid.New().String()
	
	tests := []struct {
		name           string
		authHeader     string
		setupMocks     func(*mockTokenService, *mockUserStore)
		expectAuth     bool
		expectedUserID uuid.UUID
	}{
		{
			name:       "ok/valid_token_creates_auth_context",
			authHeader: "Bearer valid-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseResult = &service.AccessClaims{
					Sub:            userID.String(),
					Email:          "test@example.com",
					Roles:          []string{"user"},
					Permissions:    []string{"read"},
					SessionVersion: 1,
					JTI:            tokenID,
					Issuer:         "test-issuer",
					Audience:       "test-audience",
					IssuedAt:       time.Now().Unix(),
					ExpiresAt:      time.Now().Add(time.Hour).Unix(),
					StepUpUntil:    0,
				}
				userStore.user = &domain.User{
					ID:             userID,
					Email:          "test@example.com",
					Roles:          []string{"user"},
					Permissions:    []string{"read"},
					SessionVersion: 1,
				}
			},
			expectAuth:     true,
			expectedUserID: userID,
		},
		{
			name:       "ok/valid_token_with_stepup",
			authHeader: "Bearer stepup-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseResult = &service.AccessClaims{
					Sub:            userID.String(),
					Email:          "test@example.com",
					Roles:          []string{"admin"},
					Permissions:    []string{"read", "write"},
					SessionVersion: 1,
					JTI:            tokenID,
					Issuer:         "test-issuer",
					Audience:       "test-audience",
					IssuedAt:       time.Now().Unix(),
					ExpiresAt:      time.Now().Add(time.Hour).Unix(),
					StepUpUntil:    time.Now().Add(time.Minute * 5).Unix(),
				}
				userStore.user = &domain.User{
					ID:             userID,
					Email:          "test@example.com",
					Roles:          []string{"admin"},
					Permissions:    []string{"read", "write"},
					SessionVersion: 1,
				}
			},
			expectAuth:     true,
			expectedUserID: userID,
		},
		{
			name:       "ok/no_token_continues_without_auth",
			authHeader: "",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				// No setup needed
			},
			expectAuth: false,
		},
		{
			name:       "ok/invalid_bearer_format_continues_without_auth",
			authHeader: "Basic dXNlcjpwYXNz",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				// No setup needed
			},
			expectAuth: false,
		},
		{
			name:       "ok/invalid_token_continues_without_auth",
			authHeader: "Bearer invalid-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseError = errors.New("invalid token")
			},
			expectAuth: false,
		},
		{
			name:       "ok/wrong_issuer_continues_without_auth",
			authHeader: "Bearer wrong-issuer-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseResult = &service.AccessClaims{
					Sub:            userID.String(),
					Email:          "test@example.com",
					Roles:          []string{"user"},
					SessionVersion: 1,
					JTI:            tokenID,
					Issuer:         "wrong-issuer", // Different from configured issuer
					Audience:       "test-audience",
					IssuedAt:       time.Now().Unix(),
					ExpiresAt:      time.Now().Add(time.Hour).Unix(),
				}
			},
			expectAuth: false,
		},
		{
			name:       "ok/invalid_user_id_continues_without_auth",
			authHeader: "Bearer invalid-userid-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseResult = &service.AccessClaims{
					Sub:            "not-a-uuid", // Invalid UUID
					Email:          "test@example.com",
					Roles:          []string{"user"},
					SessionVersion: 1,
					JTI:            tokenID,
					Issuer:         "test-issuer",
					Audience:       "test-audience",
					IssuedAt:       time.Now().Unix(),
					ExpiresAt:      time.Now().Add(time.Hour).Unix(),
				}
			},
			expectAuth: false,
		},
		{
			name:       "ok/user_not_found_continues_without_auth",
			authHeader: "Bearer user-not-found-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseResult = &service.AccessClaims{
					Sub:            userID.String(),
					Email:          "test@example.com",
					Roles:          []string{"user"},
					SessionVersion: 1,
					JTI:            tokenID,
					Issuer:         "test-issuer",
					Audience:       "test-audience",
					IssuedAt:       time.Now().Unix(),
					ExpiresAt:      time.Now().Add(time.Hour).Unix(),
				}
				userStore.error = errors.New("user not found")
			},
			expectAuth: false,
		},
		{
			name:       "ok/nil_user_continues_without_auth",
			authHeader: "Bearer nil-user-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseResult = &service.AccessClaims{
					Sub:            userID.String(),
					Email:          "test@example.com",
					Roles:          []string{"user"},
					SessionVersion: 1,
					JTI:            tokenID,
					Issuer:         "test-issuer",
					Audience:       "test-audience",
					IssuedAt:       time.Now().Unix(),
					ExpiresAt:      time.Now().Add(time.Hour).Unix(),
				}
				userStore.user = nil // User not found
			},
			expectAuth: false,
		},
		{
			name:       "ok/session_version_mismatch_continues_without_auth",
			authHeader: "Bearer old-session-token",
			setupMocks: func(tokenSvc *mockTokenService, userStore *mockUserStore) {
				tokenSvc.parseResult = &service.AccessClaims{
					Sub:            userID.String(),
					Email:          "test@example.com",
					Roles:          []string{"user"},
					SessionVersion: 1, // Old session version
					JTI:            tokenID,
					Issuer:         "test-issuer",
					Audience:       "test-audience",
					IssuedAt:       time.Now().Unix(),
					ExpiresAt:      time.Now().Add(time.Hour).Unix(),
				}
				userStore.user = &domain.User{
					ID:             userID,
					Email:          "test@example.com",
					Roles:          []string{"user"},
					SessionVersion: 2, // Current session version
				}
			},
			expectAuth: false,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			gin.SetMode(gin.TestMode)
			
			// Setup mocks
			tokenService := &mockTokenService{}
			userStore := &mockUserStore{}
			tc.setupMocks(tokenService, userStore)
			
			// Create authenticator and router
			auth := middleware.NewAuthenticator(tokenService, userStore, "test-issuer")
			router := gin.New()
			router.Use(auth.Authenticate())
			
			// Add a test endpoint that captures the auth context
			var capturedAuth *authkit.AuthContext
			var hasAuth bool
			router.GET("/test", func(c *gin.Context) {
				authCtx, ok := authkit.From(c)
				hasAuth = ok
				if ok {
					capturedAuth = &authCtx
				}
				c.Status(http.StatusOK)
			})
			
			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Verify response
			assert.Equal(t, http.StatusOK, w.Code)
			
			// Check auth context
			if tc.expectAuth {
				require.True(t, hasAuth, "Auth context should be present")
				require.NotNil(t, capturedAuth, "Auth context should not be nil")
				assert.True(t, capturedAuth.IsAuthenticated(), "Should be authenticated")
				assert.Equal(t, tc.expectedUserID, capturedAuth.UserID)
				assert.Equal(t, tokenService.parseResult.Email, capturedAuth.Email)
				assert.Equal(t, tokenService.parseResult.Roles, capturedAuth.Roles)
				assert.Equal(t, tokenService.parseResult.Permissions, capturedAuth.Permissions)
				assert.Equal(t, tokenService.parseResult.SessionVersion, capturedAuth.SessionVersion)
				assert.Equal(t, tokenService.parseResult.JTI, capturedAuth.TokenID)
				
				// Check step-up if present
				if tokenService.parseResult.StepUpUntil > 0 {
					expectedStepUp := time.Unix(tokenService.parseResult.StepUpUntil, 0)
					assert.Equal(t, expectedStepUp, capturedAuth.StepUpValidUntil)
				} else {
					assert.True(t, capturedAuth.StepUpValidUntil.IsZero())
				}
			} else {
				if hasAuth {
					assert.False(t, capturedAuth.IsAuthenticated(), "Should not be authenticated")
				}
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	t.Parallel()
	
	tests := []struct {
		name           string
		setupAuth      func() *authkit.AuthContext
		expectedStatus int
	}{
		{
			name: "ok/authenticated_user_passes",
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           uuid.New(),
					Email:            "test@example.com",
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
			name:           "error/no_auth_context_returns_401",
			setupAuth:      func() *authkit.AuthContext { return nil },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error/empty_auth_context_returns_401",
			setupAuth: func() *authkit.AuthContext {
				// Return empty auth context (not authenticated)
				return &authkit.AuthContext{}
			},
			expectedStatus: http.StatusUnauthorized,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			gin.SetMode(gin.TestMode)
			router := gin.New()
			
			// Set up auth context first if provided
			if authCtx := tc.setupAuth(); authCtx != nil {
				router.Use(func(c *gin.Context) {
					authCtx.Set(c)
					c.Next()
				})
			}
			
			// Then apply RequireAuth middleware
			router.Use(middleware.RequireAuth())
			
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			
			// Setup request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestRequireStepUp(t *testing.T) {
	t.Parallel()
	
	now := time.Now().UTC()
	
	tests := []struct {
		name           string
		setupAuth      func() *authkit.AuthContext
		expectedStatus int
	}{
		{
			name: "ok/valid_stepup_passes",
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           uuid.New(),
					Email:            "test@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write"},
					SessionVersion:   1,
					StepUpValidUntil: now.Add(time.Minute * 5), // Valid for 5 more minutes
					TokenID:          "token-123",
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error/no_auth_context_returns_401",
			setupAuth:      func() *authkit.AuthContext { return nil },
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error/not_authenticated_returns_401",
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error/expired_stepup_returns_428",
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           uuid.New(),
					Email:            "test@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write"},
					SessionVersion:   1,
					StepUpValidUntil: now.Add(-time.Minute), // Expired 1 minute ago
					TokenID:          "token-123",
				}
			},
			expectedStatus: http.StatusPreconditionRequired,
		},
		{
			name: "error/no_stepup_returns_428",
			setupAuth: func() *authkit.AuthContext {
				return &authkit.AuthContext{
					UserID:           uuid.New(),
					Email:            "test@example.com",
					Roles:            []string{"admin"},
					Permissions:      []string{"read", "write"},
					SessionVersion:   1,
					StepUpValidUntil: time.Time{}, // No step-up
					TokenID:          "token-123",
				}
			},
			expectedStatus: http.StatusPreconditionRequired,
		},
	}
	
	for _, tc := range tests {
		tc := tc // capture range variable for parallel tests
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			gin.SetMode(gin.TestMode)
			router := gin.New()
			
			// Set up auth context first if provided
			if authCtx := tc.setupAuth(); authCtx != nil {
				router.Use(func(c *gin.Context) {
					authCtx.Set(c)
					c.Next()
				})
			}
			
			// Then apply RequireStepUp middleware
			router.Use(middleware.RequireStepUp())
			
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			
			// Setup request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestOptionalAuth(t *testing.T) {
	t.Parallel()
	
	// Setup mocks
	tokenService := &mockTokenService{}
	userStore := &mockUserStore{}
	
	// Create authenticator
	auth := middleware.NewAuthenticator(tokenService, userStore, "test-issuer")
	
	// Verify OptionalAuth returns a handler
	optionalHandler := auth.OptionalAuth()
	assert.NotNil(t, optionalHandler)
}