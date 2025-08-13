package middleware

import (
    "net/http"

    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "github.com/gin-gonic/gin"
)

// SecurityHeaders wraps httpkit.SecurityHeaders for Gin.
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        handler := basehttpkit.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            c.Next()
        }))
        handler.ServeHTTP(c.Writer, c.Request)
    }
}

// CORS wraps httpkit.CORS for Gin.
func CORS(config basehttpkit.CORSConfig) gin.HandlerFunc {
    return func(c *gin.Context) {
        handler := basehttpkit.CORS(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            c.Next()
        }))
        handler.ServeHTTP(c.Writer, c.Request)
    }
}

// ContentTypeJSON wraps httpkit.ContentTypeJSON for Gin.
func ContentTypeJSON() gin.HandlerFunc {
    return func(c *gin.Context) {
        handler := basehttpkit.ContentTypeJSON(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            c.Next()
        }))
        handler.ServeHTTP(c.Writer, c.Request)
    }
}

// RequireCSRF wraps httpkit.RequireCSRF for Gin.
func RequireCSRF(csrf basehttpkit.CSRF) gin.HandlerFunc {
    return func(c *gin.Context) {
        handler := basehttpkit.RequireCSRF(csrf)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            c.Next()
        }))
        handler.ServeHTTP(c.Writer, c.Request)
    }
}
