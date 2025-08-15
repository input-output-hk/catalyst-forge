package httpkit

import (
    "net/http"
    "net/http/httptest"
    basehttpkit "github.com/catalystgo/catalyst-forge/lib/foundry/httpkit"
    "testing"
    "time"
)

func TestRefreshCookie_SetGetClear(t *testing.T) {
    cfg := basehttpkit.DefaultCookieConfig()
    ttl := time.Hour

    // Set
    w := httptest.NewRecorder()
    SetRefreshCookie(w, "refresh123", ttl, cfg)
    res := w.Result()
    if len(res.Cookies()) != 1 { t.Fatalf("expected 1 cookie, got %d", len(res.Cookies())) }
    c := res.Cookies()[0]
    if c.Name != "__Host-refresh_token" || !c.Secure || !c.HttpOnly || c.Path != "/" {
        t.Errorf("refresh cookie flags incorrect: %+v", c)
    }

    // Get
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.AddCookie(c)
    val, err := GetRefreshCookie(req)
    if err != nil || val != "refresh123" { t.Fatalf("get refresh value=%q err=%v", val, err) }

    // Clear
    w2 := httptest.NewRecorder()
    ClearRefreshCookie(w2, cfg)
    res2 := w2.Result()
    if len(res2.Cookies()) != 1 { t.Fatalf("expected 1 cookie on clear, got %d", len(res2.Cookies())) }
    c2 := res2.Cookies()[0]
    if c2.MaxAge != -1 || c2.Value != "" || c2.Path != "/" { t.Errorf("clear flags incorrect: %+v", c2) }
}

func TestBootstrapCookie_SetGetClear(t *testing.T) {
    cfg := basehttpkit.DefaultCookieConfig()
    ttl := time.Hour

    // Set
    w := httptest.NewRecorder()
    SetBootstrapCookie(w, "boot123", ttl, cfg)
    res := w.Result()
    if len(res.Cookies()) != 1 { t.Fatalf("expected 1 cookie, got %d", len(res.Cookies())) }
    c := res.Cookies()[0]
    if c.Name != "__Host-bootstrap_token" || !c.Secure || !c.HttpOnly || c.Path != "/" {
        t.Errorf("bootstrap cookie flags incorrect: %+v", c)
    }

    // Get
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.AddCookie(c)
    val, err := GetBootstrapCookie(req)
    if err != nil || val != "boot123" { t.Fatalf("get bootstrap value=%q err=%v", val, err) }

    // Clear
    w2 := httptest.NewRecorder()
    ClearBootstrapCookie(w2, cfg)
    res2 := w2.Result()
    if len(res2.Cookies()) != 1 { t.Fatalf("expected 1 cookie on clear, got %d", len(res2.Cookies())) }
    c2 := res2.Cookies()[0]
    if c2.MaxAge != -1 || c2.Value != "" || c2.Path != "/" { t.Errorf("clear flags incorrect: %+v", c2) }
}

