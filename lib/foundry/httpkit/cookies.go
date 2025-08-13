package httpkit

import (
    "net/http"
    "time"
)

// CookieConfig holds configuration for secure cookies used by httpkit primitives.
// For __Host- cookies, Domain must be empty and Secure must be true.
type CookieConfig struct {
    Secure   bool
    SameSite http.SameSite
    Domain   string // empty for __Host- prefix cookies
}

// DefaultCookieConfig returns a secure default configuration suitable for __Host- cookies.
func DefaultCookieConfig() CookieConfig {
    return CookieConfig{
        Secure:   true,
        SameSite: http.SameSiteStrictMode,
        Domain:   "", // Required empty for __Host- prefix
    }
}

// mustHostCookie enforces invariants common to all __Host- cookies.
func mustHostCookie(cfg CookieConfig) {
    if !cfg.Secure {
        panic("__Host- cookies require Secure=true")
    }
    if cfg.Domain != "" {
        panic("__Host- cookies require empty Domain attribute")
    }
}

// SetCSRFCookie sets a CSRF token cookie (not HttpOnly so JS can read it).
func SetCSRFCookie(w http.ResponseWriter, value string, ttl time.Duration, cfg CookieConfig) {
    mustHostCookie(cfg)
    if ttl <= 0 {
        panic("CSRF cookie TTL must be positive for persistent cookies")
    }
    cookie := &http.Cookie{
        Name:     "__Host-csrf_token",
        Value:    value,
        Path:     "/",          // __Host- requires "/"
        Secure:   true,          // enforce
        HttpOnly: false,         // readable by JS
        SameSite: cfg.SameSite,  // Strict or Lax per your CSRF strategy
        MaxAge:   int(ttl.Seconds()),
        Expires:  time.Now().Add(ttl),
    }
    http.SetCookie(w, cookie)
}

// GetCSRFCookie retrieves the CSRF token from cookies.
func GetCSRFCookie(r *http.Request) (string, error) {
    cookie, err := r.Cookie("__Host-csrf_token")
    if err != nil {
        return "", err
    }
    return cookie.Value, nil
}

// ClearCSRFCookie removes the CSRF token cookie.
func ClearCSRFCookie(w http.ResponseWriter, cfg CookieConfig) {
    mustHostCookie(cfg)
    cookie := &http.Cookie{
        Name:     "__Host-csrf_token",
        Value:    "",
        Path:     "/",
        Secure:   true,
        HttpOnly: false,
        SameSite: cfg.SameSite,
        MaxAge:   -1,
        Expires:  time.Unix(0, 0),
    }
    http.SetCookie(w, cookie)
}

