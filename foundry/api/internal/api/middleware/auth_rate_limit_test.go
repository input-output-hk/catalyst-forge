package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/config"
	"github.com/input-output-hk/catalyst-forge/foundry/api/internal/rate"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
)

func TestAuthRateLimitMiddleware(t *testing.T) {
    t.Parallel()
	// Setup
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	authConfig := &config.AuthConfig{
		AuthRateLimitBurst:  2, // Allow 2 requests per window
		AuthRateLimitWindow: 1 * time.Minute,
	}

	limiter := rate.NewInMemoryLimiter()
	middleware := NewAuthRateLimitMiddleware(limiter, authConfig, logger)

	// Create test router
	router := gin.New()
	router.Use(middleware.Handle())
	router.POST("/auth/refresh", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

    t.Run("allows requests within rate limit", func(t *testing.T) {
        t.Parallel()
		// First request should succeed
		req1 := httptest.NewRequest("POST", "/auth/refresh", nil)
		req1.Header.Set("X-Device-Id", "test-device-1")
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request should also succeed
		req2 := httptest.NewRequest("POST", "/auth/refresh", nil)
		req2.Header.Set("X-Device-Id", "test-device-1")
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
	})

    t.Run("blocks requests exceeding rate limit", func(t *testing.T) {
        t.Parallel()
		// Third request should be rate limited
		req3 := httptest.NewRequest("POST", "/auth/refresh", nil)
		req3.Header.Set("X-Device-Id", "test-device-1")
		w3 := httptest.NewRecorder()
		router.ServeHTTP(w3, req3)
		assert.Equal(t, http.StatusTooManyRequests, w3.Code)

		// Check error response
		assert.Contains(t, w3.Body.String(), "Rate limit exceeded")
		assert.Contains(t, w3.Body.String(), "retry_after")
	})

    t.Run("allows requests from different devices", func(t *testing.T) {
        t.Parallel()
		// Request from different device should succeed
		req4 := httptest.NewRequest("POST", "/auth/refresh", nil)
		req4.Header.Set("X-Device-Id", "test-device-2")
		w4 := httptest.NewRecorder()
		router.ServeHTTP(w4, req4)
		assert.Equal(t, http.StatusOK, w4.Code)
	})

    t.Run("handles missing device ID", func(t *testing.T) {
        t.Parallel()
		// Request without device ID should still work (fallback to "token-refresh" key)
		req5 := httptest.NewRequest("POST", "/auth/refresh", nil)
		w5 := httptest.NewRecorder()
		router.ServeHTTP(w5, req5)
		assert.Equal(t, http.StatusOK, w5.Code)
	})
}

func TestAuthRateLimitMiddleware_DeviceRegistration(t *testing.T) {
    t.Parallel()
	// Setup
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	authConfig := &config.AuthConfig{
		AuthRateLimitBurst:  1, // Allow 1 request per window
		AuthRateLimitWindow: 1 * time.Minute,
	}

	limiter := rate.NewInMemoryLimiter()
	middleware := NewAuthRateLimitMiddleware(limiter, authConfig, logger)

	// Create test router
	router := gin.New()
	router.Use(middleware.Handle())
	router.POST("/auth/devices/init", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	t.Run("rate limits device registration", func(t *testing.T) {
		// First request should succeed
		req1 := httptest.NewRequest("POST", "/auth/devices/init", nil)
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second request should be rate limited
		req2 := httptest.NewRequest("POST", "/auth/devices/init", nil)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	})
}
