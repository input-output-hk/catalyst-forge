package httpkit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultCSRF(t *testing.T) {
	csrf := NewDefaultCSRF(DefaultCookieConfig(), time.Hour)
	
	t.Run("generates static token", func(t *testing.T) {
		token, err := csrf.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		
		// DefaultCSRF returns a static token since it's header-only
		if token != "header-only-csrf-protection" {
			t.Errorf("expected static token, got %q", token)
		}
	})
	
	t.Run("validates with correct header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		
		err := csrf.Validate(req)
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})
	
	t.Run("rejects missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		
		err := csrf.Validate(req)
		if err != ErrCSRFHeaderMissing {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFHeaderMissing)
		}
	})
	
	t.Run("rejects wrong header value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.Header.Set("X-Requested-With", "WrongValue")
		
		err := csrf.Validate(req)
		if err != ErrCSRFHeaderMissing {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFHeaderMissing)
		}
	})
}

func TestDoubleSubmitCSRF(t *testing.T) {
	// Use a fixed secret for deterministic testing
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}
	csrf := NewDoubleSubmitCSRFWithSecret(DefaultCookieConfig(), time.Hour, secret)
	
	t.Run("validates matching tokens", func(t *testing.T) {
		token, err := csrf.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		
		// Create request with both cookie and header
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "__Host-csrf_token",
			Value: token,
		})
		req.Header.Set("X-CSRF-Token", token)
		
		err = csrf.Validate(req)
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})
	
	t.Run("rejects mismatched tokens", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "__Host-csrf_token",
			Value: "token1",
		})
		req.Header.Set("X-CSRF-Token", "token2")
		
		err := csrf.Validate(req)
		if err != ErrCSRFTokenInvalid {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFTokenInvalid)
		}
	})
	
	t.Run("rejects missing cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-CSRF-Token", "token")
		
		err := csrf.Validate(req)
		if err != ErrCSRFTokenMissing {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFTokenMissing)
		}
	})
	
	t.Run("rejects missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "__Host-csrf_token",
			Value: "token",
		})
		
		err := csrf.Validate(req)
		if err != ErrCSRFHeaderMissing {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFHeaderMissing)
		}
	})
	
	t.Run("rejects tampered token", func(t *testing.T) {
		token, err := csrf.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		
		// Tamper with the token by changing the last character
		tamperedToken := token[:len(token)-1] + "X"
		
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "__Host-csrf_token",
			Value: tamperedToken,
		})
		req.Header.Set("X-CSRF-Token", tamperedToken)
		
		err = csrf.Validate(req)
		if err != ErrCSRFTokenInvalid {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFTokenInvalid)
		}
	})
}

func TestMemoryCSRF(t *testing.T) {
	csrf := NewMemoryCSRF(DefaultCookieConfig(), 100*time.Millisecond)
	
	t.Run("validates stored token", func(t *testing.T) {
		token, err := csrf.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-CSRF-Token", token)
		
		err = csrf.Validate(req)
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})
	
	t.Run("rejects unknown token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-CSRF-Token", "unknown-token")
		
		err := csrf.Validate(req)
		if err != ErrCSRFTokenInvalid {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFTokenInvalid)
		}
	})
	
	t.Run("rejects expired token", func(t *testing.T) {
		token, err := csrf.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		
		// Wait for token to expire
		time.Sleep(150 * time.Millisecond)
		
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-CSRF-Token", token)
		
		err = csrf.Validate(req)
		if err != ErrCSRFTokenInvalid {
			t.Errorf("Validate() error = %v, want %v", err, ErrCSRFTokenInvalid)
		}
	})
}

func TestNoOpCSRF(t *testing.T) {
	csrf := &NoOpCSRF{}
	
	t.Run("always validates", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		
		err := csrf.Validate(req)
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})
	
	t.Run("generates test token", func(t *testing.T) {
		token, err := csrf.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		
		if token != "test-token" {
			t.Errorf("token = %q, want test-token", token)
		}
	})
}

func TestSameSiteAndCustomHeaderCombo(t *testing.T) {
	tests := []struct {
		name             string
		sameSite         http.SameSite
		expectValidation bool
		description      string
	}{
		{
			name:             "strict_samesite_with_custom_header",
			sameSite:         http.SameSiteStrictMode,
			expectValidation: true,
			description:      "SameSite=Strict + custom header provides maximum protection",
		},
		{
			name:             "lax_samesite_with_custom_header", 
			sameSite:         http.SameSiteLaxMode,
			expectValidation: true,
			description:      "SameSite=Lax + custom header provides good protection for some cross-site navigation",
		},
		{
			name:             "none_samesite_with_custom_header",
			sameSite:         http.SameSiteNoneMode,
			expectValidation: true,
			description:      "SameSite=None + custom header relies entirely on custom header protection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := CookieConfig{
				Secure:   true,
				SameSite: tt.sameSite,
				Domain:   "",
			}

			// For DefaultCSRF, only SameSite=Strict is allowed
			if tt.sameSite == http.SameSiteStrictMode {
				t.Run("default_csrf_with_samesite_strict", func(t *testing.T) {
					csrf := NewDefaultCSRF(cfg, time.Hour)
					
					// Test with proper header
					req := httptest.NewRequest(http.MethodPost, "/", nil)
					req.Header.Set("X-Requested-With", "XMLHttpRequest")
					
					err := csrf.Validate(req)
					if err != nil {
						t.Errorf("DefaultCSRF validation failed: %v", err)
					}
				})
			} else {
				t.Run("default_csrf_panics_without_strict", func(t *testing.T) {
					defer func() {
						if r := recover(); r == nil {
							t.Error("expected panic for DefaultCSRF without SameSite=Strict")
						}
					}()
					NewDefaultCSRF(cfg, time.Hour)
				})
			}

			// Test DoubleSubmitCSRF with all SameSite modes
			t.Run("double_submit_csrf_with_"+tt.name, func(t *testing.T) {
				secret := make([]byte, 32)
				for i := range secret {
					secret[i] = byte(i)
				}
				csrf := NewDoubleSubmitCSRFWithSecret(cfg, time.Hour, secret)
				
				// Generate token
				token, err := csrf.Generate()
				if err != nil {
					t.Fatalf("Generate() error = %v", err)
				}

				// Test cookie setting with proper SameSite attribute
				w := httptest.NewRecorder()
				csrf.SetCookie(w, token)
				
				result := w.Result()
				cookies := result.Cookies()
				if len(cookies) != 1 {
					t.Fatalf("expected 1 cookie, got %d", len(cookies))
				}
				
				cookie := cookies[0]
				if cookie.SameSite != tt.sameSite {
					t.Errorf("cookie SameSite = %v, want %v", cookie.SameSite, tt.sameSite)
				}

				// Test validation with both cookie and header
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "__Host-csrf_token",
					Value: token,
				})
				req.Header.Set("X-CSRF-Token", token)
				
				err = csrf.Validate(req)
				if (err == nil) != tt.expectValidation {
					t.Errorf("validation result = %v, expected success = %v", err, tt.expectValidation)
				}
			})
		})
	}
}

func TestTimingAttackResistance(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}
	csrf := NewDoubleSubmitCSRFWithSecret(DefaultCookieConfig(), time.Hour, secret)
	
	// Generate a valid token for reference
	validToken, err := csrf.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Test different types of invalid tokens
	invalidTokenTests := []struct {
		name        string
		tokenValue  string
		description string
	}{
		{
			name:        "empty_token",
			tokenValue:  "",
			description: "Empty token should be rejected",
		},
		{
			name:        "short_token",
			tokenValue:  "abc",
			description: "Short token should be rejected",
		},
		{
			name:        "long_token",
			tokenValue:  validToken + "extra",
			description: "Token with extra data should be rejected",
		},
		{
			name:        "invalid_base64",
			tokenValue:  "invalid-base64!!!",
			description: "Invalid base64 should be rejected",
		},
		{
			name:        "tampered_token",
			tokenValue:  validToken[:len(validToken)-1] + "X",
			description: "Tampered token should be rejected",
		},
		{
			name:        "different_valid_length_token",
			tokenValue:  "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXpBQkNERUZHSElKS0xNTk9QUVJTVFVWV1hZWjAxMjM0NTY3ODk",
			description: "Different valid-length token should be rejected",
		},
	}

	for _, tt := range invalidTokenTests {
		t.Run(tt.name, func(t *testing.T) {
			// Test constant-time behavior by measuring timing of rejections
			const iterations = 100
			timings := make([]time.Duration, iterations)
			
			for i := 0; i < iterations; i++ {
				req := httptest.NewRequest(http.MethodPost, "/", nil)
				req.AddCookie(&http.Cookie{
					Name:  "__Host-csrf_token",
					Value: tt.tokenValue,
				})
				req.Header.Set("X-CSRF-Token", tt.tokenValue)
				
				start := time.Now()
				err := csrf.Validate(req)
				elapsed := time.Since(start)
				
				timings[i] = elapsed
				
				// All invalid tokens should be rejected
				if err == nil {
					t.Errorf("expected error for %s, got nil", tt.description)
				}
			}
			
			// Calculate basic timing statistics
			var total time.Duration
			var min, max time.Duration = timings[0], timings[0]
			
			for _, timing := range timings {
				total += timing
				if timing < min {
					min = timing
				}
				if timing > max {
					max = timing
				}
			}
			
			avg := total / time.Duration(iterations)
			
			// Log timing info for manual inspection
			t.Logf("%s: avg=%v, min=%v, max=%v, ratio=%.2f", 
				tt.name, avg, min, max, float64(max)/float64(min))
			
			// Basic sanity check: timing shouldn't vary by more than 10x
			// This is a very loose check to catch obvious timing leaks
			if max > min*10 {
				t.Logf("WARNING: Large timing variation detected for %s (may indicate timing attack vulnerability)", tt.name)
			}
		})
	}
	
	// Test that valid token comparison is in similar time range
	t.Run("valid_token_timing", func(t *testing.T) {
		const iterations = 50
		var total time.Duration
		
		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  "__Host-csrf_token",
				Value: validToken,
			})
			req.Header.Set("X-CSRF-Token", validToken)
			
			start := time.Now()
			err := csrf.Validate(req)
			elapsed := time.Since(start)
			
			total += elapsed
			
			if err != nil {
				t.Errorf("unexpected error for valid token: %v", err)
			}
		}
		
		avg := total / time.Duration(iterations)
		t.Logf("valid_token: avg=%v", avg)
	})
	
	// Test mismatched cookie vs header (most common attack vector)
	t.Run("mismatched_tokens_timing", func(t *testing.T) {
		const iterations = 50
		
		// Generate a second valid token for mismatch test
		otherToken, err := csrf.Generate()
		if err != nil {
			t.Fatalf("Generate() error = %v", err)
		}
		
		var total time.Duration
		
		for i := 0; i < iterations; i++ {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.AddCookie(&http.Cookie{
				Name:  "__Host-csrf_token", 
				Value: validToken,
			})
			req.Header.Set("X-CSRF-Token", otherToken)
			
			start := time.Now()
			err := csrf.Validate(req)
			elapsed := time.Since(start)
			
			total += elapsed
			
			// Should reject mismatched tokens
			if err != ErrCSRFTokenInvalid {
				t.Errorf("expected ErrCSRFTokenInvalid for mismatched tokens, got %v", err)
			}
		}
		
		avg := total / time.Duration(iterations)
		t.Logf("mismatched_tokens: avg=%v", avg)
	})
}