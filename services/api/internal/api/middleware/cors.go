package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a simple CORS middleware for non-auth endpoints.
// It allows credentials and restricts allowed origins to the configured PUBLIC_BASE_URL
// placed into the Gin context as "public_base_url".
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		base := c.GetString("public_base_url")
		if base == "" {
			base = ""
		}
		if origin != "" && base != "" && origin == base {
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token, X-CLI")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		} else if origin != "" && base != "" {
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == "OPTIONS" {
			c.Status(204)
			c.Abort()
			return
		}
		c.Next()
	}
}
