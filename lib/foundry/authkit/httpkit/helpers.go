package httpkit

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MaxRequestBodySize is the maximum allowed request body size (1MB)
const MaxRequestBodySize = 1 << 20 // 1MB

// ParseJSON parses JSON from request body with size limit
func ParseJSON(w http.ResponseWriter, r *http.Request, v any) error {
	// Limit request body size - MUST pass ResponseWriter for proper enforcement
	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)
	
	// Decode JSON
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing
	
	if err := decoder.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return NewBadRequestError("Request body is empty")
		}
		// Check for max bytes error using errors.As (safer than string matching)
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return NewBadRequestError("Request body too large")
		}
		if strings.Contains(err.Error(), "unknown field") {
			return NewBadRequestError("Unknown field in request")
		}
		return NewBadRequestError("Invalid JSON")
	}
	
	// Check for trailing data (more reliable approach)
	if decoder.More() {
		// Attempt to decode extra data to verify it's actually trailing
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return NewBadRequestError("Request body contains multiple JSON values")
		}
	}
	
	return nil
}

// GetBearerToken extracts the bearer token from Authorization header
func GetBearerToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", NewUnauthorizedError("Missing authorization header")
	}
	
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", NewUnauthorizedError("Invalid authorization header format")
	}
	
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", NewUnauthorizedError("Empty bearer token")
	}
	
	return token, nil
}

// GetClientInfo extracts client information from request
type ClientInfo struct {
	UserAgent string
	IP        string
	Origin    string
	Referer   string
}

// GetClientInfo extracts client information from the request
func GetClientInfo(r *http.Request) ClientInfo {
	return ClientInfo{
		UserAgent: r.Header.Get("User-Agent"),
		IP:        getClientIP(r),
		Origin:    r.Header.Get("Origin"),
		Referer:   r.Header.Get("Referer"),
	}
}

// getClientIP attempts to get the real client IP
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Try X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	
	return ip
}

// IsSecureContext checks if the request is over HTTPS
func IsSecureContext(r *http.Request) bool {
	// Check X-Forwarded-Proto header
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto == "https"
	}
	
	// Check if TLS is not nil
	if r.TLS != nil {
		return true
	}
	
	// Check URL scheme
	return r.URL.Scheme == "https"
}

// SetSecurityHeaders sets common security headers
func SetSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")
}

// originAllowed checks if an origin is allowed with proper credential handling
func originAllowed(origin string, allowed []string, allowCreds bool) (bool, string) {
	if origin == "" {
		return false, ""
	}
	for _, ao := range allowed {
		if ao == origin {
			return true, origin
		}
		if ao == "*" && !allowCreds {
			// Only reflect "*" when not sending credentials
			return true, "*"
		}
	}
	return false, ""
}

// SetCORSHeaders sets CORS headers for auth endpoints
func SetCORSHeaders(w http.ResponseWriter, origin string, allowedOrigins []string) {
	// Use the secure origin checking that prevents "*" with credentials
	if ok, allow := originAllowed(origin, allowedOrigins, true); ok {
		w.Header().Set("Access-Control-Allow-Origin", allow)
		if allow != "*" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
	}
}

// HandlePreflight handles CORS preflight requests
func HandlePreflight(w http.ResponseWriter, r *http.Request, allowedOrigins []string) {
	if r.Method != http.MethodOptions {
		return
	}
	
	origin := r.Header.Get("Origin")
	// Reuse SetCORSHeaders which now uses secure origin checking
	SetCORSHeaders(w, origin, allowedOrigins)
	w.WriteHeader(http.StatusNoContent)
}

// ValidateContentType ensures the request has the expected content type
func ValidateContentType(r *http.Request, expected string) error {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return NewBadRequestError("Missing Content-Type header")
	}
	
	// Handle charset suffix
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}
	
	if !strings.EqualFold(ct, expected) {
		return NewBadRequestError("Invalid Content-Type: expected " + expected)
	}
	
	return nil
}

// TimeoutHandler wraps a handler with a timeout
func TimeoutHandler(h http.HandlerFunc, timeout time.Duration, msg string) http.HandlerFunc {
	if msg == "" {
		msg = "Request timeout"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		http.TimeoutHandler(h, timeout, msg).ServeHTTP(w, r)
	}
}

// RecoveryHandler recovers from panics and returns a 500 error
func RecoveryHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic (in production, send to error tracking)
				// fmt.Printf("Panic recovered: %v\n", err)
				
				// Return generic error
				_ = NewInternalError().Write(w)
			}
		}()
		
		next.ServeHTTP(w, r)
	}
}

// ChainMiddleware chains multiple middleware functions
func ChainMiddleware(h http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	// Apply middlewares in reverse order so they execute in the order provided
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

// RequestID generates a unique request ID for tracing
func RequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based ID if crypto/rand fails
		return "req-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	// Use RawURLEncoding for URL-safe base64 without padding
	return base64.RawURLEncoding.EncodeToString(b)
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
// IMPORTANT: For security, no origins are allowed by default when credentials are enabled
// You must explicitly configure allowed origins for your deployment
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{}, // Require explicit allowlist for auth endpoints
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true, // Only safe with explicit origins
		MaxAge:          86400, // 24 hours
	}
}

// CORS returns a middleware that handles CORS
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Use secure origin checking that prevents "*" with credentials
			if ok, allow := originAllowed(origin, config.AllowedOrigins, config.AllowCredentials); ok {
				w.Header().Set("Access-Control-Allow-Origin", allow)
				if config.AllowCredentials && allow != "*" {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				// Set exposed headers on actual responses, not just preflight
				if len(config.ExposedHeaders) > 0 && r.Method != http.MethodOptions {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}
			}
			
			// Handle preflight
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// NewHeaderCSRF creates a CSRF implementation that requires a custom header
func NewHeaderCSRF() CSRF {
	return NewDefaultCSRF(DefaultCookieConfig(), 24*time.Hour)
}

// SecurityHeaders returns middleware that sets security headers
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetSecurityHeaders(w)
		next.ServeHTTP(w, r)
	})
}

// ContentTypeJSON returns middleware that validates JSON content type
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "DELETE" && r.Method != "OPTIONS" {
			if err := ValidateContentType(r, "application/json"); err != nil {
				if httpErr, ok := err.(*HTTPError); ok {
					_ = httpErr.Write(w)
					return
				}
				_ = NewBadRequestError("Invalid content type").Write(w)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequireCSRF returns middleware that enforces CSRF protection
func RequireCSRF(csrf CSRF) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
				if err := csrf.Validate(r); err != nil {
					_ = NewCSRFError("").Write(w)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WithRequestID adds a request ID to the context
func WithRequestID(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = RequestID()
		}
		w.Header().Set("X-Request-ID", reqID)
		next.ServeHTTP(w, r)
	}
}