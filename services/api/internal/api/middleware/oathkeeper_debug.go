package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// OathkeeperDump logs request metadata and all headers for debugging Ory Oathkeeper integration.
func OathkeeperDump(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := make(map[string]any, len(c.Request.Header))
		for k, v := range c.Request.Header {
			if len(v) == 1 {
				headers[k] = v[0]
			} else {
				headers[k] = v
			}
		}

		logger.Info("Oathkeeper request received",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"remote", c.ClientIP(),
			"headers", headers,
		)

		c.Next()
	}
}
