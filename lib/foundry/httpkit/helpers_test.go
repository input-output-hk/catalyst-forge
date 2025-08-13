package httpkit

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestParseJSON_Success(t *testing.T) {
    body, _ := json.Marshal(map[string]any{"name":"test","value":42})
    req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    var out map[string]any
    if err := ParseJSON(w, req, &out); err != nil { t.Fatalf("unexpected error: %v", err) }
}

func TestParseJSON_EmptyBody(t *testing.T) {
    req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(nil))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    var out any
    if err := ParseJSON(w, req, &out); err == nil { t.Fatal("expected error") }
}

func TestCORS_Preflight(t *testing.T) {
    cfg := DefaultCORSConfig()
    cfg.AllowedOrigins = []string{"https://example.com"}
    mw := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
    req := httptest.NewRequest(http.MethodOptions, "/", nil)
    req.Header.Set("Origin", "https://example.com")
    w := httptest.NewRecorder()
    mw.ServeHTTP(w, req)
    if w.Code != http.StatusNoContent { t.Fatalf("want %d got %d", http.StatusNoContent, w.Code) }
}

