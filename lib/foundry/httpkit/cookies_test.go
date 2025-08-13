package httpkit

import (
    "net/http/httptest"
    "testing"
    "time"
)

func TestCSRFCookie(t *testing.T) {
    w := httptest.NewRecorder()
    cfg := DefaultCookieConfig()
    SetCSRFCookie(w, "csrf-token", time.Hour, cfg)
    res := w.Result()
    cookies := res.Cookies()
    if len(cookies) != 1 { t.Fatalf("expected 1 cookie, got %d", len(cookies)) }
    c := cookies[0]
    if c.Name != "__Host-csrf_token" || !c.Secure || c.HttpOnly || c.Path != "/" { t.Errorf("bad csrf cookie: %+v", c) }
}
