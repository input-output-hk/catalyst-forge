package api

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/config"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/handlers"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/logging"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter configures the Gin router for the Auth Orchestrator.
// All routes are grouped under /api/v1.
func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(requestLogger())
	// Contextual logger with request-scoped attributes
	r.Use(func(c *gin.Context) {
		reqID, _ := c.Get("request_id")
		logger := logging.L()
		if reqID != nil {
			logger = logger.With(slog.Any("request_id", reqID))
		}
		if lc, ok := c.Get("corr_login_challenge"); ok {
			logger = logger.With(slog.Any("login_challenge", lc))
		}
		if cc, ok := c.Get("corr_consent_challenge"); ok {
			logger = logger.With(slog.Any("consent_challenge", cc))
		}
		c.Set("logger", logger)
		c.Next()
	})

	// Initialize handler deps
	h := handlers.NewHandlers(cfg)

	// Canonical health endpoint
	r.GET("/healthz", h.Health)

	// Versioned API group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/oauth2/login", h.Login)
		v1.GET("/oauth2/consent", h.ConsentGet)
		v1.POST("/oauth2/consent", h.ConsentPost)
		v1.POST("/hydra/token-hook", h.TokenHook)

		// BFF: expose Kratos login flow JSON without cookies for SPA
		v1.GET("/kratos/login-flow", h.KratosLoginFlow)

		// Prometheus metrics endpoint
		v1.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	return r
}

// errorJSON produces a unified error body.
func errorJSON(code int, message string, corr map[string]any) gin.H {
	h := gin.H{"error": message}
	if len(corr) > 0 {
		h["correlation"] = corr
	}
	return h
}

// requestLogger is a lightweight correlation middleware.
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v := c.Query("login_challenge"); v != "" {
			c.Set("corr_login_challenge", v)
		}
		if v := c.Query("consent_challenge"); v != "" {
			c.Set("corr_consent_challenge", v)
		}

		reqID := c.Request.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = time.Now().UTC().Format("20060102T150405.000000000Z07:00")
		}
		c.Writer.Header().Set("X-Request-ID", reqID)
		c.Set("request_id", reqID)

		started := time.Now()
		c.Next()
		latency := time.Since(started)

		status := c.Writer.Status()
		method := c.Request.Method
		url := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			url = url + "?" + raw
		}

		lc, _ := c.Get("corr_login_challenge")
		cc, _ := c.Get("corr_consent_challenge")
		logger := logging.L().With(
			slog.String("request_id", reqID),
		)
		if lc != nil {
			logger = logger.With(slog.Any("login_challenge", lc))
		}
		if cc != nil {
			logger = logger.With(slog.Any("consent_challenge", cc))
		}
		logger.Info("request",
			slog.String("method", method),
			slog.String("url", url),
			slog.Int("status", status),
			slog.Duration("latency", latency),
		)
	}
}
