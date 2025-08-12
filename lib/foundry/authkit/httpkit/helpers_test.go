package httpkit

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseJSON(t *testing.T) {
	type testData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	t.Run("parses valid JSON", func(t *testing.T) {
		data := testData{Name: "test", Value: 42}
		body, _ := json.Marshal(data)
		
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		var result testData
		err := ParseJSON(w, req, &result)
		
		if err != nil {
			t.Errorf("ParseJSON() error = %v, want nil", err)
		}
		
		if result.Name != data.Name || result.Value != data.Value {
			t.Errorf("ParseJSON() got = %+v, want %+v", result, data)
		}
	})

	t.Run("rejects body too large", func(t *testing.T) {
		// Create a body larger than MaxRequestBodySize
		largeBody := make([]byte, MaxRequestBodySize+1)
		for i := range largeBody {
			largeBody[i] = 'a'
		}
		
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		var result testData
		err := ParseJSON(w, req, &result)
		
		if err == nil {
			t.Error("ParseJSON() error = nil, want error for large body")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusBadRequest {
			t.Errorf("ParseJSON() error = %v, want BadRequest error", err)
		}
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		var result testData
		err := ParseJSON(w, req, &result)
		
		if err == nil {
			t.Error("ParseJSON() error = nil, want error for invalid JSON")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusBadRequest {
			t.Errorf("ParseJSON() error = %v, want BadRequest error", err)
		}
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		body := `{"name":"test","value":42,"unknown":"field"}`
		
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		var result testData
		err := ParseJSON(w, req, &result)
		
		if err == nil {
			t.Error("ParseJSON() error = nil, want error for unknown field")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusBadRequest {
			t.Errorf("ParseJSON() error = %v, want BadRequest error", err)
		}
	})

	t.Run("rejects empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		var result testData
		err := ParseJSON(w, req, &result)
		
		if err == nil {
			t.Error("ParseJSON() error = nil, want error for empty body")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusBadRequest || !strings.Contains(httpErr.Message, "empty") {
			t.Errorf("ParseJSON() error = %v, want BadRequest with 'empty' message", err)
		}
	})
}

func TestRequestID(t *testing.T) {
	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		
		for i := 0; i < 100; i++ {
			id := RequestID()
			
			if id == "" {
				t.Error("RequestID() returned empty string")
			}
			
			if ids[id] {
				t.Errorf("RequestID() generated duplicate ID: %s", id)
			}
			
			ids[id] = true
		}
	})

	t.Run("generates URL-safe IDs", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			id := RequestID()
			
			// Check that ID is URL-safe (no padding, no special chars except - and _)
			for _, c := range id {
				if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || 
					(c >= '0' && c <= '9') || c == '-' || c == '_') {
					t.Errorf("RequestID() generated non-URL-safe character: %c in ID: %s", c, id)
				}
			}
		}
	})
}

func TestOriginAllowed(t *testing.T) {
	tests := []struct {
		name         string
		origin       string
		allowed      []string
		allowCreds   bool
		wantOk       bool
		wantAllow    string
	}{
		{
			name:       "exact match allowed",
			origin:     "https://example.com",
			allowed:    []string{"https://example.com", "https://other.com"},
			allowCreds: true,
			wantOk:     true,
			wantAllow:  "https://example.com",
		},
		{
			name:       "wildcard allowed without credentials",
			origin:     "https://example.com",
			allowed:    []string{"*"},
			allowCreds: false,
			wantOk:     true,
			wantAllow:  "*",
		},
		{
			name:       "wildcard blocked with credentials",
			origin:     "https://example.com",
			allowed:    []string{"*"},
			allowCreds: true,
			wantOk:     false,
			wantAllow:  "",
		},
		{
			name:       "empty origin rejected",
			origin:     "",
			allowed:    []string{"*"},
			allowCreds: false,
			wantOk:     false,
			wantAllow:  "",
		},
		{
			name:       "unlisted origin rejected",
			origin:     "https://evil.com",
			allowed:    []string{"https://example.com"},
			allowCreds: true,
			wantOk:     false,
			wantAllow:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, allow := originAllowed(tt.origin, tt.allowed, tt.allowCreds)
			
			if ok != tt.wantOk {
				t.Errorf("originAllowed() ok = %v, want %v", ok, tt.wantOk)
			}
			
			if allow != tt.wantAllow {
				t.Errorf("originAllowed() allow = %q, want %q", allow, tt.wantAllow)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("blocks wildcard with credentials", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
		}
		
		middleware := CORS(config)
		h := middleware(handler)
		
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		h.ServeHTTP(w, req)
		
		// Should not set CORS headers when "*" is used with credentials
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("CORS middleware should not set headers for wildcard with credentials")
		}
	})

	t.Run("allows specific origin with credentials", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins:   []string{"https://example.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
		}
		
		middleware := CORS(config)
		h := middleware(handler)
		
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		h.ServeHTTP(w, req)
		
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Error("CORS middleware should set origin header for allowed origin")
		}
		
		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("CORS middleware should set credentials header")
		}
	})

	t.Run("handles preflight requests", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins:   []string{"https://example.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
			MaxAge:          3600,
		}
		
		middleware := CORS(config)
		h := middleware(handler)
		
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		h.ServeHTTP(w, req)
		
		if w.Code != http.StatusNoContent {
			t.Errorf("Preflight response code = %d, want %d", w.Code, http.StatusNoContent)
		}
		
		if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
			t.Error("Preflight should set allowed methods")
		}
		
		if w.Header().Get("Access-Control-Max-Age") != "3600" {
			t.Error("Preflight should set max age")
		}
	})

	t.Run("sets exposed headers on actual response", func(t *testing.T) {
		config := CORSConfig{
			AllowedOrigins:   []string{"https://example.com"},
			AllowedMethods:   []string{"GET"},
			ExposedHeaders:   []string{"X-Request-ID", "X-Total-Count"},
			AllowCredentials: true,
		}
		
		middleware := CORS(config)
		h := middleware(handler)
		
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		
		h.ServeHTTP(w, req)
		
		exposedHeaders := w.Header().Get("Access-Control-Expose-Headers")
		if exposedHeaders != "X-Request-ID, X-Total-Count" {
			t.Errorf("CORS should expose headers on actual response, got: %s", exposedHeaders)
		}
	})
}

func TestSetCORSHeaders(t *testing.T) {
	t.Run("allows specific origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		
		SetCORSHeaders(w, "https://example.com", []string{"https://example.com"})
		
		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Error("SetCORSHeaders should set origin for allowed origin")
		}
		
		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("SetCORSHeaders should set credentials for specific origin")
		}
	})

	t.Run("blocks wildcard with implicit credentials", func(t *testing.T) {
		w := httptest.NewRecorder()
		
		// SetCORSHeaders always uses credentials=true internally
		SetCORSHeaders(w, "https://example.com", []string{"*"})
		
		// Should not set any headers since "*" with credentials is blocked
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("SetCORSHeaders should not set headers for wildcard (credentials are always true)")
		}
	})

	t.Run("rejects unlisted origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		
		SetCORSHeaders(w, "https://evil.com", []string{"https://example.com"})
		
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("SetCORSHeaders should not set headers for unlisted origin")
		}
	})
}

func TestGetBearerToken(t *testing.T) {
	t.Run("extracts valid bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer token123")
		
		token, err := GetBearerToken(req)
		
		if err != nil {
			t.Errorf("GetBearerToken() error = %v, want nil", err)
		}
		
		if token != "token123" {
			t.Errorf("GetBearerToken() token = %q, want %q", token, "token123")
		}
	})

	t.Run("rejects missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		
		_, err := GetBearerToken(req)
		
		if err == nil {
			t.Error("GetBearerToken() error = nil, want error for missing header")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusUnauthorized {
			t.Errorf("GetBearerToken() error = %v, want Unauthorized error", err)
		}
	})

	t.Run("rejects invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "NotBearer token123")
		
		_, err := GetBearerToken(req)
		
		if err == nil {
			t.Error("GetBearerToken() error = nil, want error for invalid format")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusUnauthorized {
			t.Errorf("GetBearerToken() error = %v, want Unauthorized error", err)
		}
	})

	t.Run("rejects empty token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer ")
		
		_, err := GetBearerToken(req)
		
		if err == nil {
			t.Error("GetBearerToken() error = nil, want error for empty token")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusUnauthorized {
			t.Errorf("GetBearerToken() error = %v, want Unauthorized error", err)
		}
	})
}

func TestValidateContentType(t *testing.T) {
	t.Run("accepts matching content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Content-Type", "application/json")
		
		err := ValidateContentType(req, "application/json")
		
		if err != nil {
			t.Errorf("ValidateContentType() error = %v, want nil", err)
		}
	})

	t.Run("accepts with charset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		
		err := ValidateContentType(req, "application/json")
		
		if err != nil {
			t.Errorf("ValidateContentType() error = %v, want nil", err)
		}
	})

	t.Run("rejects missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		
		err := ValidateContentType(req, "application/json")
		
		if err == nil {
			t.Error("ValidateContentType() error = nil, want error for missing header")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusBadRequest {
			t.Errorf("ValidateContentType() error = %v, want BadRequest error", err)
		}
	})

	t.Run("rejects wrong content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("Content-Type", "text/plain")
		
		err := ValidateContentType(req, "application/json")
		
		if err == nil {
			t.Error("ValidateContentType() error = nil, want error for wrong type")
		}
		
		httpErr, ok := err.(*HTTPError)
		if !ok || httpErr.Status != http.StatusBadRequest {
			t.Errorf("ValidateContentType() error = %v, want BadRequest error", err)
		}
	})
}

func TestIsSecureContext(t *testing.T) {
	t.Run("detects HTTPS from TLS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://example.com/", nil)
		req.TLS = &tls.ConnectionState{} // Non-nil indicates HTTPS
		
		if !IsSecureContext(req) {
			t.Error("IsSecureContext() = false, want true for HTTPS with TLS")
		}
	})

	t.Run("detects HTTPS from X-Forwarded-Proto", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		
		if !IsSecureContext(req) {
			t.Error("IsSecureContext() = false, want true for X-Forwarded-Proto=https")
		}
	})

	t.Run("detects HTTP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
		
		if IsSecureContext(req) {
			t.Error("IsSecureContext() = true, want false for HTTP")
		}
	})
}

func TestSetSecurityHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	
	SetSecurityHeaders(w)
	
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Cache-Control":          "no-store, no-cache, must-revalidate, private",
		"Pragma":                 "no-cache",
	}
	
	for header, expected := range expectedHeaders {
		got := w.Header().Get(header)
		if got != expected {
			t.Errorf("SetSecurityHeaders() header %q = %q, want %q", header, got, expected)
		}
	}
}