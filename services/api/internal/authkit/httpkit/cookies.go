package httpkit

import (
	"net/http"
	"time"

	basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
)

// local invariant check for __Host- cookies in auth-specific helpers
func mustHostCookie(cfg basehttpkit.CookieConfig) {
	if !cfg.Secure {
		panic("__Host- cookies require Secure=true")
	}
	if cfg.Domain != "" {
		panic("__Host- cookies require empty Domain attribute")
	}
}

// SetRefreshCookie sets a secure refresh token cookie with __Host- prefix
func SetRefreshCookie(w http.ResponseWriter, value string, ttl time.Duration, cfg basehttpkit.CookieConfig) {
	mustHostCookie(cfg)
	if ttl <= 0 {
		panic("refresh cookie TTL must be positive for persistent cookies")
	}
	cookie := &http.Cookie{
		Name:     "__Host-refresh_token",
		Value:    value,
		Path:     "/",  // __Host- requires "/"
		Secure:   true, // enforce
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode, // enforce Strict per spec
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl), // for broader compatibility
	}
	http.SetCookie(w, cookie)
}

// ClearRefreshCookie removes the refresh token cookie
func ClearRefreshCookie(w http.ResponseWriter, cfg basehttpkit.CookieConfig) {
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

// SetBootstrapCookie sets a temporary cookie for bootstrap flow
func SetBootstrapCookie(w http.ResponseWriter, value string, ttl time.Duration, cfg basehttpkit.CookieConfig) {
	mustHostCookie(cfg)
	if ttl <= 0 {
		panic("bootstrap cookie TTL must be positive for persistent cookies")
	}
	cookie := &http.Cookie{
		Name:     "__Host-bootstrap_token",
		Value:    value,
		Path:     "/",                     // __Host- requires "/"
		Secure:   true,                    // enforce
		HttpOnly: true,                    // session-like
		SameSite: http.SameSiteStrictMode, // align with session semantics
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl), // for broader compatibility
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
func ClearBootstrapCookie(w http.ResponseWriter, cfg basehttpkit.CookieConfig) {
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

// Note: CSRF cookie helpers are provided by lib/foundry/httpkit directly

// SetAccessCookie sets a secure access token cookie with __Host- prefix
func SetAccessCookie(w http.ResponseWriter, value string, ttl time.Duration, cfg basehttpkit.CookieConfig) {
	mustHostCookie(cfg)
	if ttl <= 0 {
		// Access tokens are short-lived but still require a positive TTL
		panic("access cookie TTL must be positive for persistent cookies")
	}
	cookie := &http.Cookie{
		Name:     "__Host-access_token",
		Value:    value,
		Path:     "/",  // __Host- requires "/"
		Secure:   true, // enforce
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode, // align with session semantics
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl), // for broader compatibility
	}
	http.SetCookie(w, cookie)
}

// GetAccessCookie retrieves the access token from cookies
func GetAccessCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("__Host-access_token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

// ClearAccessCookie removes the access token cookie
func ClearAccessCookie(w http.ResponseWriter, cfg basehttpkit.CookieConfig) {
	mustHostCookie(cfg)
	cookie := &http.Cookie{
		Name:     "__Host-access_token",
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
