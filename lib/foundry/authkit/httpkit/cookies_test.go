package httpkit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRefreshCookie(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		ttl      time.Duration
		cfg      CookieConfig
		wantName string
		wantPath string
	}{
		{
			name:     "sets secure refresh cookie",
			value:    "test-token",
			ttl:      30 * 24 * time.Hour,
			cfg:      DefaultCookieConfig(),
			wantName: "__Host-refresh_token",
			wantPath: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			
			SetRefreshCookie(w, tt.value, tt.ttl, tt.cfg)
			
			result := w.Result()
			cookies := result.Cookies()
			
			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}
			
			cookie := cookies[0]
			
			if cookie.Name != tt.wantName {
				t.Errorf("cookie name = %q, want %q", cookie.Name, tt.wantName)
			}
			
			if cookie.Value != tt.value {
				t.Errorf("cookie value = %q, want %q", cookie.Value, tt.value)
			}
			
			if cookie.Path != tt.wantPath {
				t.Errorf("cookie path = %q, want %q", cookie.Path, tt.wantPath)
			}
			
			if !cookie.HttpOnly {
				t.Error("cookie should be HttpOnly")
			}
			
			if !cookie.Secure {
				t.Error("cookie should be Secure")
			}
			
			if cookie.SameSite != http.SameSiteStrictMode {
				t.Errorf("cookie SameSite = %v, want Strict", cookie.SameSite)
			}
			
			// Check that Expires is set and reasonable
			if cookie.Expires.IsZero() {
				t.Error("cookie should have Expires set")
			}
			
			// Expires should be approximately now + ttl
			expectedExpiry := time.Now().Add(tt.ttl)
			if diff := cookie.Expires.Sub(expectedExpiry); diff > time.Second || diff < -time.Second {
				t.Errorf("cookie Expires is off by %v", diff)
			}
		})
	}
}

func TestGetRefreshCookie(t *testing.T) {
	tests := []struct {
		name      string
		cookie    *http.Cookie
		wantValue string
		wantErr   bool
	}{
		{
			name: "gets existing cookie",
			cookie: &http.Cookie{
				Name:  "__Host-refresh_token",
				Value: "test-token",
			},
			wantValue: "test-token",
			wantErr:   false,
		},
		{
			name:      "returns error for missing cookie",
			cookie:    nil,
			wantValue: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			
			value, err := GetRefreshCookie(req)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if value != tt.wantValue {
				t.Errorf("value = %q, want %q", value, tt.wantValue)
			}
		})
	}
}

func TestClearRefreshCookie(t *testing.T) {
	w := httptest.NewRecorder()
	cfg := DefaultCookieConfig()
	
	ClearRefreshCookie(w, cfg)
	
	result := w.Result()
	cookies := result.Cookies()
	
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	
	cookie := cookies[0]
	
	if cookie.MaxAge != -1 {
		t.Errorf("cookie MaxAge = %d, want -1 for deletion", cookie.MaxAge)
	}
	
	if cookie.Value != "" {
		t.Errorf("cookie value = %q, want empty", cookie.Value)
	}
}

func TestCSRFCookie(t *testing.T) {
	w := httptest.NewRecorder()
	cfg := DefaultCookieConfig()
	ttl := 1 * time.Hour
	token := "csrf-token"
	
	SetCSRFCookie(w, token, ttl, cfg)
	
	result := w.Result()
	cookies := result.Cookies()
	
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	
	cookie := cookies[0]
	
	if cookie.Name != "__Host-csrf_token" {
		t.Errorf("cookie name = %q, want __Host-csrf_token", cookie.Name)
	}
	
	if cookie.HttpOnly {
		t.Error("CSRF cookie should NOT be HttpOnly (needs JS access)")
	}
	
	if !cookie.Secure {
		t.Error("CSRF cookie should be Secure")
	}
}

func TestHostPrefixRequirements(t *testing.T) {
	t.Run("panics with domain set", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for __Host- prefix with domain set")
			}
		}()
		
		w := httptest.NewRecorder()
		cfg := CookieConfig{
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Domain:   "example.com", // This should cause panic
		}
		
		SetRefreshCookie(w, "test", time.Hour, cfg)
	})
	
	t.Run("panics with Secure=false", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for __Host- prefix with Secure=false")
			}
		}()
		
		w := httptest.NewRecorder()
		cfg := CookieConfig{
			Secure:   false, // This should cause panic
			SameSite: http.SameSiteStrictMode,
			Domain:   "",
		}
		
		SetRefreshCookie(w, "test", time.Hour, cfg)
	})
	
	t.Run("panics with zero TTL", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for zero TTL")
			}
		}()
		
		w := httptest.NewRecorder()
		cfg := DefaultCookieConfig()
		
		SetRefreshCookie(w, "test", 0, cfg) // This should cause panic
	})
	
	t.Run("panics with negative TTL", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for negative TTL")
			}
		}()
		
		w := httptest.NewRecorder()
		cfg := DefaultCookieConfig()
		
		SetRefreshCookie(w, "test", -1*time.Hour, cfg) // This should cause panic
	})
}