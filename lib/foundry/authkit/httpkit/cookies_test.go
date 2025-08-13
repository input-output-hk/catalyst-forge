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

func TestHostPrefixEnforcement_AllCookieTypes(t *testing.T) {
	// Test that all cookie types enforce __Host- prefix requirements
	tests := []struct {
		name     string
		testFunc func(http.ResponseWriter, CookieConfig)
	}{
		{
			name: "SetRefreshCookie enforces __Host- requirements",
			testFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetRefreshCookie(w, "test", time.Hour, cfg)
			},
		},
		{
			name: "SetCSRFCookie enforces __Host- requirements",
			testFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetCSRFCookie(w, "test", time.Hour, cfg)
			},
		},
		{
			name: "SetBootstrapCookie enforces __Host- requirements",
			testFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetBootstrapCookie(w, "test", time.Hour, cfg)
			},
		},
		{
			name: "ClearRefreshCookie enforces __Host- requirements",
			testFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				ClearRefreshCookie(w, cfg)
			},
		},
		{
			name: "ClearCSRFCookie enforces __Host- requirements",
			testFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				ClearCSRFCookie(w, cfg)
			},
		},
		{
			name: "ClearBootstrapCookie enforces __Host- requirements",
			testFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				ClearBootstrapCookie(w, cfg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with invalid domain (should panic)
			t.Run("panics_with_domain_set", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic for __Host- prefix with domain set")
					}
				}()

				w := httptest.NewRecorder()
				cfg := CookieConfig{
					Secure:   true,
					SameSite: http.SameSiteStrictMode,
					Domain:   "example.com", // Invalid for __Host-
				}

				tt.testFunc(w, cfg)
			})

			// Test with Secure=false (should panic)
			t.Run("panics_with_secure_false", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic for __Host- prefix with Secure=false")
					}
				}()

				w := httptest.NewRecorder()
				cfg := CookieConfig{
					Secure:   false, // Invalid for __Host-
					SameSite: http.SameSiteStrictMode,
					Domain:   "",
				}

				tt.testFunc(w, cfg)
			})
		})
	}
}

func TestHostPrefixCompliance_CookieNames(t *testing.T) {
	// Test that all cookies use proper __Host- prefix names
	tests := []struct {
		name         string
		setFunc      func(http.ResponseWriter, CookieConfig)
		expectedName string
	}{
		{
			name: "refresh_cookie_uses_host_prefix",
			setFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetRefreshCookie(w, "test", time.Hour, cfg)
			},
			expectedName: "__Host-refresh_token",
		},
		{
			name: "csrf_cookie_uses_host_prefix",
			setFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetCSRFCookie(w, "test", time.Hour, cfg)
			},
			expectedName: "__Host-csrf_token",
		},
		{
			name: "bootstrap_cookie_uses_host_prefix",
			setFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetBootstrapCookie(w, "test", time.Hour, cfg)
			},
			expectedName: "__Host-bootstrap_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cfg := DefaultCookieConfig()

			tt.setFunc(w, cfg)

			result := w.Result()
			cookies := result.Cookies()

			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]
			if cookie.Name != tt.expectedName {
				t.Errorf("cookie name = %q, want %q", cookie.Name, tt.expectedName)
			}

			// Verify __Host- prefix requirements are met
			if !cookie.Secure {
				t.Error("__Host- cookie must be Secure")
			}

			if cookie.Path != "/" {
				t.Errorf("__Host- cookie path = %q, want '/'", cookie.Path)
			}

			// Domain should not be set for __Host- cookies
			if cookie.Domain != "" {
				t.Errorf("__Host- cookie domain = %q, want empty", cookie.Domain)
			}
		})
	}
}

func TestSecurityFlags_AllCookieTypes(t *testing.T) {
	// Test that all security flags are properly set on all cookie types
	tests := []struct {
		name             string
		setFunc          func(http.ResponseWriter, CookieConfig)
		expectedHttpOnly bool
		expectedSameSite http.SameSite
	}{
		{
			name: "refresh_cookie_security_flags",
			setFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetRefreshCookie(w, "test", time.Hour, cfg)
			},
			expectedHttpOnly: true,
			expectedSameSite: http.SameSiteStrictMode,
		},
		{
			name: "csrf_cookie_security_flags",
			setFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetCSRFCookie(w, "test", time.Hour, cfg)
			},
			expectedHttpOnly: false, // CSRF tokens need JS access
			expectedSameSite: http.SameSiteStrictMode,
		},
		{
			name: "bootstrap_cookie_security_flags",
			setFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				SetBootstrapCookie(w, "test", time.Hour, cfg)
			},
			expectedHttpOnly: true,
			expectedSameSite: http.SameSiteStrictMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cfg := DefaultCookieConfig()

			tt.setFunc(w, cfg)

			result := w.Result()
			cookies := result.Cookies()

			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]

			// Test all security flags
			if cookie.Secure != true {
				t.Error("cookie must be Secure")
			}

			if cookie.HttpOnly != tt.expectedHttpOnly {
				t.Errorf("cookie HttpOnly = %v, want %v", cookie.HttpOnly, tt.expectedHttpOnly)
			}

			if cookie.SameSite != tt.expectedSameSite {
				t.Errorf("cookie SameSite = %v, want %v", cookie.SameSite, tt.expectedSameSite)
			}

			if cookie.Path != "/" {
				t.Errorf("cookie Path = %q, want '/'", cookie.Path)
			}

			// MaxAge should be positive for set operations
			if cookie.MaxAge <= 0 {
				t.Errorf("cookie MaxAge = %d, want positive value", cookie.MaxAge)
			}

			// Expires should be set and in the future
			if cookie.Expires.IsZero() {
				t.Error("cookie Expires should be set")
			}

			if !cookie.Expires.After(time.Now()) {
				t.Error("cookie Expires should be in the future")
			}
		})
	}
}

func TestSecurityFlags_CustomSameSite(t *testing.T) {
	// Test that CSRF cookies respect custom SameSite settings
	tests := []struct {
		name             string
		sameSite         http.SameSite
		expectedSameSite http.SameSite
	}{
		{
			name:             "csrf_cookie_with_strict_samesite",
			sameSite:         http.SameSiteStrictMode,
			expectedSameSite: http.SameSiteStrictMode,
		},
		{
			name:             "csrf_cookie_with_lax_samesite",
			sameSite:         http.SameSiteLaxMode,
			expectedSameSite: http.SameSiteLaxMode,
		},
		{
			name:             "csrf_cookie_with_none_samesite",
			sameSite:         http.SameSiteNoneMode,
			expectedSameSite: http.SameSiteNoneMode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cfg := CookieConfig{
				Secure:   true,
				SameSite: tt.sameSite,
				Domain:   "",
			}

			SetCSRFCookie(w, "test", time.Hour, cfg)

			result := w.Result()
			cookies := result.Cookies()

			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]
			if cookie.SameSite != tt.expectedSameSite {
				t.Errorf("CSRF cookie SameSite = %v, want %v", cookie.SameSite, tt.expectedSameSite)
			}
		})
	}
}

func TestSecurityFlags_ClearOperations(t *testing.T) {
	// Test that clear operations maintain security flags
	tests := []struct {
		name         string
		clearFunc    func(http.ResponseWriter, CookieConfig)
		expectedName string
	}{
		{
			name: "clear_refresh_cookie_maintains_security",
			clearFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				ClearRefreshCookie(w, cfg)
			},
			expectedName: "__Host-refresh_token",
		},
		{
			name: "clear_csrf_cookie_maintains_security",
			clearFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				ClearCSRFCookie(w, cfg)
			},
			expectedName: "__Host-csrf_token",
		},
		{
			name: "clear_bootstrap_cookie_maintains_security",
			clearFunc: func(w http.ResponseWriter, cfg CookieConfig) {
				ClearBootstrapCookie(w, cfg)
			},
			expectedName: "__Host-bootstrap_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cfg := DefaultCookieConfig()

			tt.clearFunc(w, cfg)

			result := w.Result()
			cookies := result.Cookies()

			if len(cookies) != 1 {
				t.Fatalf("expected 1 cookie, got %d", len(cookies))
			}

			cookie := cookies[0]

			// Verify cookie name uses __Host- prefix
			if cookie.Name != tt.expectedName {
				t.Errorf("cookie name = %q, want %q", cookie.Name, tt.expectedName)
			}

			// Verify deletion properties
			if cookie.Value != "" {
				t.Errorf("cleared cookie value = %q, want empty", cookie.Value)
			}

			if cookie.MaxAge != -1 {
				t.Errorf("cleared cookie MaxAge = %d, want -1", cookie.MaxAge)
			}

			// Verify security flags are maintained even for deletion
			if !cookie.Secure {
				t.Error("cleared cookie must maintain Secure flag")
			}

			if cookie.Path != "/" {
				t.Errorf("cleared cookie Path = %q, want '/'", cookie.Path)
			}

			if cookie.Domain != "" {
				t.Errorf("cleared cookie Domain = %q, want empty", cookie.Domain)
			}

			// Expires should be set to Unix epoch for deletion
			expectedExpires := time.Unix(0, 0)
			if !cookie.Expires.Equal(expectedExpires) {
				t.Errorf("cleared cookie Expires = %v, want %v", cookie.Expires, expectedExpires)
			}
		})
	}
}

func TestGetCookieFunctions_AllTypes(t *testing.T) {
	// Test all Get* functions for proper cookie retrieval
	tests := []struct {
		name       string
		cookieName string
		getFunc    func(*http.Request) (string, error)
	}{
		{
			name:       "get_refresh_cookie",
			cookieName: "__Host-refresh_token",
			getFunc:    GetRefreshCookie,
		},
		{
			name:       "get_csrf_cookie",
			cookieName: "__Host-csrf_token",
			getFunc:    GetCSRFCookie,
		},
		{
			name:       "get_bootstrap_cookie",
			cookieName: "__Host-bootstrap_token",
			getFunc:    GetBootstrapCookie,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("retrieves_existing_cookie", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  tt.cookieName,
					Value: "test-value",
				})

				value, err := tt.getFunc(req)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if value != "test-value" {
					t.Errorf("value = %q, want 'test-value'", value)
				}
			})

			t.Run("returns_error_for_missing_cookie", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)

				value, err := tt.getFunc(req)
				if err == nil {
					t.Error("expected error for missing cookie")
				}

				if value != "" {
					t.Errorf("value = %q, want empty string", value)
				}
			})

			t.Run("ignores_cookies_without_host_prefix", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				// Add cookie without __Host- prefix
				cookieNameWithoutPrefix := tt.cookieName[8:] // Remove "__Host-"
				req.AddCookie(&http.Cookie{
					Name:  cookieNameWithoutPrefix,
					Value: "should-be-ignored",
				})

				value, err := tt.getFunc(req)
				if err == nil {
					t.Error("expected error when __Host- prefixed cookie is missing")
				}

				if value != "" {
					t.Errorf("value = %q, want empty string for missing __Host- cookie", value)
				}
			})
		})
	}
}

func TestSecurityFlags_TTLValidation(t *testing.T) {
	// Test TTL validation for all cookie set functions
	setCookieFuncs := []struct {
		name     string
		setFunc  func(http.ResponseWriter, string, time.Duration, CookieConfig)
		funcName string
	}{
		{
			name:     "refresh_cookie_ttl_validation",
			setFunc:  SetRefreshCookie,
			funcName: "SetRefreshCookie",
		},
		{
			name:     "csrf_cookie_ttl_validation",
			setFunc:  SetCSRFCookie,
			funcName: "SetCSRFCookie",
		},
		{
			name:     "bootstrap_cookie_ttl_validation",
			setFunc:  SetBootstrapCookie,
			funcName: "SetBootstrapCookie",
		},
	}

	for _, tt := range setCookieFuncs {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultCookieConfig()

			t.Run("accepts_positive_ttl", func(t *testing.T) {
				w := httptest.NewRecorder()

				// Should not panic with positive TTL
				tt.setFunc(w, "test", time.Hour, cfg)

				result := w.Result()
				cookies := result.Cookies()

				if len(cookies) != 1 {
					t.Fatalf("expected 1 cookie, got %d", len(cookies))
				}

				cookie := cookies[0]
				if cookie.MaxAge <= 0 {
					t.Errorf("cookie MaxAge = %d, want positive value", cookie.MaxAge)
				}
			})

			t.Run("panics_with_zero_ttl", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("%s should panic with zero TTL", tt.funcName)
					}
				}()

				w := httptest.NewRecorder()
				tt.setFunc(w, "test", 0, cfg)
			})

			t.Run("panics_with_negative_ttl", func(t *testing.T) {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("%s should panic with negative TTL", tt.funcName)
					}
				}()

				w := httptest.NewRecorder()
				tt.setFunc(w, "test", -time.Hour, cfg)
			})
		})
	}
}