package httpkit

import (
	"net/http"
	"time"
)

// CookieConfig holds configuration for secure cookies
type CookieConfig struct {
	Secure   bool
	SameSite http.SameSite
	Domain   string // empty for __Host- prefix cookies
}

// DefaultCookieConfig returns secure cookie configuration
func DefaultCookieConfig() CookieConfig {
	return CookieConfig{
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Domain:   "", // Required empty for __Host- prefix
	}
}

// mustHostCookie enforces invariants common to all __Host- cookies
func mustHostCookie(cfg CookieConfig) {
	if !cfg.Secure {
		panic("__Host- cookies require Secure=true")
	}
	if cfg.Domain != "" {
		panic("__Host- cookies require empty Domain attribute")
	}
}

// SetRefreshCookie sets a secure refresh token cookie with __Host- prefix
func SetRefreshCookie(w http.ResponseWriter, value string, ttl time.Duration, cfg CookieConfig) {
	mustHostCookie(cfg)
	if ttl <= 0 {
		panic("refresh cookie TTL must be positive for persistent cookies")
	}
	cookie := &http.Cookie{
		Name:     "__Host-refresh_token",
		Value:    value,
		Path:     "/",                      // __Host- requires "/"
		Secure:   true,                     // enforce
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode, // enforce Strict per spec
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),     // for broader compatibility
	}
	http.SetCookie(w, cookie)
}

// ClearRefreshCookie removes the refresh token cookie
func ClearRefreshCookie(w http.ResponseWriter, cfg CookieConfig) {
	mustHostCookie(cfg)
	cookie := &http.Cookie{
		Name:     "__Host-refresh_token",
		Value:    "",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0), // optional hard-delete
	}
	http.SetCookie(w, cookie)
}

// GetRefreshCookie retrieves the refresh token from cookies
func GetRefreshCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("__Host-refresh_token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// SetCSRFCookie sets a CSRF token cookie (not HttpOnly so JS can read it)
func SetCSRFCookie(w http.ResponseWriter, value string, ttl time.Duration, cfg CookieConfig) {
	mustHostCookie(cfg) // also __Host-
	if ttl <= 0 {
		panic("CSRF cookie TTL must be positive for persistent cookies")
	}
	cookie := &http.Cookie{
		Name:     "__Host-csrf_token",
		Value:    value,
		Path:     "/",           // __Host- requires "/"
		Secure:   true,          // enforce
		HttpOnly: false,         // readable by JS
		SameSite: cfg.SameSite,  // Strict or Lax per your CSRF strategy
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl), // for broader compatibility
	}
	http.SetCookie(w, cookie)
}

// GetCSRFCookie retrieves the CSRF token from cookies
func GetCSRFCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("__Host-csrf_token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// ClearCSRFCookie removes the CSRF token cookie
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

// SetBootstrapCookie sets a temporary cookie for bootstrap flow
func SetBootstrapCookie(w http.ResponseWriter, value string, ttl time.Duration, cfg CookieConfig) {
	mustHostCookie(cfg)
	if ttl <= 0 {
		panic("bootstrap cookie TTL must be positive for persistent cookies")
	}
	cookie := &http.Cookie{
		Name:     "__Host-bootstrap_token",
		Value:    value,
		Path:     "/",                      // __Host- requires "/"
		Secure:   true,                     // enforce
		HttpOnly: true,                     // session-like
		SameSite: http.SameSiteStrictMode, // align with session semantics
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),     // for broader compatibility
	}
	http.SetCookie(w, cookie)
}

// GetBootstrapCookie retrieves the bootstrap token from cookies
func GetBootstrapCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("__Host-bootstrap_token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// ClearBootstrapCookie removes the bootstrap token cookie
func ClearBootstrapCookie(w http.ResponseWriter, cfg CookieConfig) {
	mustHostCookie(cfg)
	cookie := &http.Cookie{
		Name:     "__Host-bootstrap_token",
		Value:    "",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	}
	http.SetCookie(w, cookie)
}