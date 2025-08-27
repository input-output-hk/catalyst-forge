package api

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/config"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/handlers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRouter configures the Gin router for the Auth Orchestrator.
// All routes are grouped under /api/v1.
func SetupRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	// Initialize handler deps
	h := handlers.NewHandlers(cfg)

	// Versioned API group
	v1 := r.Group("/api/v1")
	{
		v1.GET("/oauth2/login", h.Login)
		v1.GET("/oauth2/consent", h.ConsentGet)
		v1.POST("/oauth2/consent", h.ConsentPost)
		v1.POST("/hydra/token-hook", h.TokenHook)

		// Health within API group per requirement
		v1.GET("/health", h.Health)

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
		log.Printf(
			"request method=%s url=%s status=%d latency=%s request_id=%s login_challenge=%v consent_challenge=%v",
			method, url, status, latency, reqID, lc, cc,
		)
	}
}
