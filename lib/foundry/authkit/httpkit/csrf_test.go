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