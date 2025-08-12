package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
)

func setupTestCookieManager() *CookieManager {
	cfg := &config.AuthConfig{
		RefreshCookieName:   "test_rt",
		RefreshCookieDomain: "",
		RefreshCookieSecure: false,
	}
	return NewCookieManager(cfg)
}

func setupGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	return c, w
}

func TestNewCookieManager(t *testing.T) {
    t.Parallel()
	cfg := &config.AuthConfig{
		RefreshCookieName: "test_cookie",
	}

	cm := NewCookieManager(cfg)

	assert.NotNil(t, cm)
	assert.Equal(t, cfg, cm.config)
}

func TestCookieManager_SetRefreshTokenCookie(t *testing.T) {
    t.Parallel()
	cm := setupTestCookieManager()
    c, w := setupGinContext()

	jti := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	secret := "test_secret_value"
	ttl := 24 * time.Hour

	cm.SetRefreshTokenCookie(c, jti, secret, ttl)

	// Check that cookie was set in response
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, "test_rt", cookie.Name)
	assert.Equal(t, "/auth", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	assert.Equal(t, int(ttl.Seconds()), cookie.MaxAge)

	// Verify cookie format
	err := cm.ValidateRefreshTokenCookieFormat(cookie.Value)
	assert.NoError(t, err)

	// Verify we can parse it back
	parsed, err := cm.ParseRefreshTokenValue(cookie.Value)
	require.NoError(t, err)
	assert.Equal(t, jti, parsed.JTI)
	assert.Equal(t, secret, parsed.Secret)
}

func TestCookieManager_ParseRefreshTokenCookie(t *testing.T) {
    t.Parallel()
	cm := setupTestCookieManager()

	jti := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	secret := "test_secret"

	tests := []struct {
		name        string
		setupCookie func(c *gin.Context)
		wantErr     bool
		wantErrMsg  string
		wantJTI     uuid.UUID
		wantSecret  string
	}{
		{
			name: "valid cookie",
			setupCookie: func(c *gin.Context) {
				// First set the cookie to get the formatted value
				tempC, tempW := setupGinContext()
				cm.SetRefreshTokenCookie(tempC, jti, secret, time.Hour)
				cookies := tempW.Result().Cookies()
				require.NotEmpty(t, cookies, "Should have at least one cookie")
				cookieStr := cookies[0].Name + "=" + cookies[0].Value
				c.Request.Header.Set("Cookie", cookieStr)
			},
			wantErr:    false,
			wantJTI:    jti,
			wantSecret: secret,
		},
		{
			name: "missing cookie",
			setupCookie: func(c *gin.Context) {
				// Don't set any cookie
			},
			wantErr:    true,
			wantErrMsg: "refresh token cookie not found",
		},
		{
			name: "invalid cookie format",
			setupCookie: func(c *gin.Context) {
				c.Request.Header.Set("Cookie", "test_rt=invalid_format")
			},
			wantErr:    true,
			wantErrMsg: "invalid cookie format",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			c, _ := setupGinContext() // Fresh context for each test
			tt.setupCookie(c)

			result, err := cm.ParseRefreshTokenCookie(c)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantJTI, result.JTI)
				assert.Equal(t, tt.wantSecret, result.Secret)
			}
		})
	}
}

func TestCookieManager_ParseRefreshTokenValue(t *testing.T) {
    t.Parallel()
	cm := setupTestCookieManager()

	validJTI := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	validSecret := "test_secret"

	tests := []struct {
		name        string
		cookieValue string
		wantErr     bool
		wantErrMsg  string
		wantJTI     uuid.UUID
		wantSecret  string
	}{
		{
			name:        "valid format",
			cookieValue: "NTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAw.dGVzdF9zZWNyZXQ",
			wantErr:     false,
			wantJTI:     validJTI,
			wantSecret:  validSecret,
		},
		{
			name:        "empty value",
			cookieValue: "",
			wantErr:     true,
			wantErrMsg:  "cookie value is empty",
		},
		{
			name:        "missing dot separator",
			cookieValue: "invalidformat",
			wantErr:     true,
			wantErrMsg:  "invalid cookie format",
		},
		{
			name:        "invalid JTI encoding",
			cookieValue: "invalid_base64!.dGVzdF9zZWNyZXQ",
			wantErr:     true,
			wantErrMsg:  "invalid JTI encoding",
		},
		{
			name:        "invalid secret encoding",
			cookieValue: "NTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAw.invalid_base64!",
			wantErr:     true,
			wantErrMsg:  "invalid secret encoding",
		},
		{
			name:        "invalid UUID format in JTI",
			cookieValue: "aW52YWxpZC11dWlk.dGVzdF9zZWNyZXQ", // "invalid-uuid" base64url encoded
			wantErr:     true,
			wantErrMsg:  "invalid JTI format",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			result, err := cm.ParseRefreshTokenValue(tt.cookieValue)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantJTI, result.JTI)
				assert.Equal(t, tt.wantSecret, result.Secret)
			}
		})
	}
}

func TestCookieManager_ClearRefreshTokenCookie(t *testing.T) {
    t.Parallel()
    cm := setupTestCookieManager()
    c, _ := setupGinContext()

	// First set a cookie
	jti := uuid.New()
	cm.SetRefreshTokenCookie(c, jti, "secret", time.Hour)

	// Clear the response and set up fresh context
    c, w := setupGinContext()

	// Clear the cookie
	cm.ClearRefreshTokenCookie(c)

	// Check that clearing cookie was set
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, "test_rt", cookie.Name)
	assert.Empty(t, cookie.Value)
	assert.Equal(t, "/auth", cookie.Path)
	assert.Equal(t, -1, cookie.MaxAge)
	assert.True(t, cookie.HttpOnly)
}

func TestCookieManager_SetAccessTokenCookie(t *testing.T) {
    t.Parallel()
	cm := setupTestCookieManager()
	c, w := setupGinContext()

	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test"
	ttl := 30 * time.Minute

	cm.SetAccessTokenCookie(c, jwt, ttl)

	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, "cforge_at", cookie.Name)
	assert.Equal(t, jwt, cookie.Value)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
}

func TestCookieManager_ShouldUseSecureFlag(t *testing.T) {
    t.Parallel()
	tests := []struct {
		name         string
		configSecure bool
		setupRequest func(c *gin.Context)
		wantSecure   bool
	}{
		{
			name:         "config forces secure",
			configSecure: true,
			setupRequest: func(c *gin.Context) {},
			wantSecure:   true,
		},
		{
			name:         "TLS detected",
			configSecure: false,
			setupRequest: func(c *gin.Context) {
				c.Request.TLS = &tls.ConnectionState{} // Non-nil TLS
			},
			wantSecure: true,
		},
		{
			name:         "X-Forwarded-Proto: https",
			configSecure: false,
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("X-Forwarded-Proto", "https")
			},
			wantSecure: true,
		},
		{
			name:         "X-Forwarded-Scheme: https",
			configSecure: false,
			setupRequest: func(c *gin.Context) {
				c.Request.Header.Set("X-Forwarded-Scheme", "https")
			},
			wantSecure: true,
		},
		{
			name:         "no secure indicators",
			configSecure: false,
			setupRequest: func(c *gin.Context) {},
			wantSecure:   false,
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			cfg := &config.AuthConfig{
				RefreshCookieSecure: tt.configSecure,
			}
			cm := NewCookieManager(cfg)

			c, _ := setupGinContext()
			tt.setupRequest(c)

			result := cm.shouldUseSecureFlag(c)
			assert.Equal(t, tt.wantSecure, result)
		})
	}
}

func TestCookieManager_GetCookieDomain(t *testing.T) {
    t.Parallel()
	tests := []struct {
		name         string
		configDomain string
		requestHost  string
		wantDomain   string
	}{
		{
			name:         "config domain takes precedence",
			configDomain: "example.com",
			requestHost:  "api.test.com:8080",
			wantDomain:   "example.com",
		},
		{
			name:         "localhost returns empty",
			configDomain: "",
			requestHost:  "localhost:3000",
			wantDomain:   "",
		},
		{
			name:         "127.0.0.1 returns empty",
			configDomain: "",
			requestHost:  "127.0.0.1:8080",
			wantDomain:   "",
		},
		{
			name:         "192.168.x.x returns empty",
			configDomain: "",
			requestHost:  "192.168.1.100:3000",
			wantDomain:   "",
		},
		{
			name:         "public domain with port",
			configDomain: "",
			requestHost:  "api.example.com:443",
			wantDomain:   "api.example.com",
		},
		{
			name:         "public domain without port",
			configDomain: "",
			requestHost:  "api.example.com",
			wantDomain:   "api.example.com",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			cfg := &config.AuthConfig{
				RefreshCookieDomain: tt.configDomain,
			}
			cm := NewCookieManager(cfg)

			c, _ := setupGinContext()
			c.Request.Host = tt.requestHost

			result := cm.getCookieDomain(c)
			assert.Equal(t, tt.wantDomain, result)
		})
	}
}

func TestCookieManager_ValidateRefreshTokenCookieFormat(t *testing.T) {
    t.Parallel()
    cm := setupTestCookieManager()

	tests := []struct {
		name        string
		cookieValue string
		wantErr     bool
		wantErrMsg  string
	}{
		{
			name:        "valid format",
			cookieValue: "NTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAw.dGVzdF9zZWNyZXQ",
			wantErr:     false,
		},
		{
			name:        "empty value",
			cookieValue: "",
			wantErr:     true,
			wantErrMsg:  "cookie value is empty",
		},
		{
			name:        "missing dot",
			cookieValue: "nodothere",
			wantErr:     true,
			wantErrMsg:  "invalid format: expected 'jti.secret'",
		},
		{
			name:        "empty JTI",
			cookieValue: ".dGVzdF9zZWNyZXQ",
			wantErr:     true,
			wantErrMsg:  "both JTI and secret must be non-empty",
		},
		{
			name:        "empty secret",
			cookieValue: "NTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAw.",
			wantErr:     true,
			wantErrMsg:  "both JTI and secret must be non-empty",
		},
		{
			name:        "invalid JTI base64",
			cookieValue: "invalid_base64!.dGVzdF9zZWNyZXQ",
			wantErr:     true,
			wantErrMsg:  "invalid JTI encoding",
		},
		{
			name:        "invalid secret base64",
			cookieValue: "NTUwZTg0MDAtZTI5Yi00MWQ0LWE3MTYtNDQ2NjU1NDQwMDAw.invalid_base64!",
			wantErr:     true,
			wantErrMsg:  "invalid secret encoding",
		},
	}

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
			err := cm.ValidateRefreshTokenCookieFormat(tt.cookieValue)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCookieManager_GetRefreshTokenCookieName(t *testing.T) {
    t.Parallel()
    cfg := &config.AuthConfig{
		RefreshCookieName: "custom_refresh_token",
	}
	cm := NewCookieManager(cfg)

	result := cm.GetRefreshTokenCookieName()
	assert.Equal(t, "custom_refresh_token", result)
}

func TestCookieManager_EndToEndFlow(t *testing.T) {
    t.Parallel()
	// Test complete flow: set -> parse -> clear
	cm := setupTestCookieManager()

	// Step 1: Set a refresh token cookie
	c1, w1 := setupGinContext()
	jti := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	secret := "test_secret_value"

	cm.SetRefreshTokenCookie(c1, jti, secret, time.Hour)

	// Extract cookie value from response
	cookies := w1.Result().Cookies()
	require.Len(t, cookies, 1)
	cookieValue := cookies[0].Value

	// Step 2: Parse the cookie value
	parsed, err := cm.ParseRefreshTokenValue(cookieValue)
	require.NoError(t, err)
	assert.Equal(t, jti, parsed.JTI)
	assert.Equal(t, secret, parsed.Secret)

	// Step 3: Set up a new context with the cookie and parse it
	c2, _ := setupGinContext()
	c2.Request.Header.Set("Cookie", "test_rt="+cookieValue)

	parsed2, err := cm.ParseRefreshTokenCookie(c2)
	require.NoError(t, err)
	assert.Equal(t, jti, parsed2.JTI)
	assert.Equal(t, secret, parsed2.Secret)

	// Step 4: Clear the cookie
	c3, w3 := setupGinContext()
	cm.ClearRefreshTokenCookie(c3)

	clearCookies := w3.Result().Cookies()
	require.Len(t, clearCookies, 1)
	assert.Empty(t, clearCookies[0].Value)
	assert.Equal(t, -1, clearCookies[0].MaxAge)
}
