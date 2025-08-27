package hydra

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLoginRequest_OK(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"skip": true, "subject": "user:1"})
	}))
	defer sr.Close()
	c := NewAdminClient(sr.URL, sr.Client())
	lr, err := c.GetLoginRequest("abc")
	if err != nil || lr == nil || !lr.Skip || lr.Subject != "user:1" {
		t.Fatalf("unexpected response: lr=%v err=%v", lr, err)
	}
}

func TestAcceptLoginRequest_OK(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://hydra/redirect"})
	}))
	defer sr.Close()
	c := NewAdminClient(sr.URL, sr.Client())
	url, err := c.AcceptLoginRequest("abc", "user:1", true, 300)
	if err != nil || url == "" {
		t.Fatalf("unexpected: url=%q err=%v", url, err)
	}
}
