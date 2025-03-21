package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a middleware that logs request information using the provided slog.Logger
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Stop timer
		latency := time.Since(start)

		// Get request status
		status := c.Writer.Status()
		method := c.Request.Method

		// Create request URL with query parameters if present
		url := path
		if raw != "" {
			url = path + "?" + raw
		}

		// Log error if there was one
		if len(c.Errors) > 0 {
			logger.Error("Request completed with errors",
				"method", method,
				"url", url,
				"status", status,
				"latency", latency,
				"errors", c.Errors.String(),
			)
		} else {
			// Basic request information
			logLevel := slog.LevelInfo
			if status >= 400 {
				logLevel = slog.LevelWarn
			}

			logger.Log(c.Request.Context(), logLevel, "Request completed",
				"method", method,
				"url", url,
				"status", status,
				"latency", latency,
			)
		}
	}
}
