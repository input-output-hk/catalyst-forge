package kratos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWhoAmIJSON_OK(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"identity": map[string]any{
				"id":     "id-1",
				"traits": map[string]any{"email": "user@example.com"},
			},
		})
	}))
	defer sr.Close()
	c := NewPublicClient(sr.URL, sr.Client())
	res, status, err := c.WhoAmIJSON(nil)
	if err != nil || status != http.StatusOK || res == nil || res.Identity.ID != "id-1" {
		t.Fatalf("unexpected: res=%v status=%d err=%v", res, status, err)
	}
}

func TestWhoAmIJSON_401(t *testing.T) {
	sr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer sr.Close()
	c := NewPublicClient(sr.URL, sr.Client())
	res, status, err := c.WhoAmIJSON(nil)
	if err != nil || status != http.StatusUnauthorized || res != nil {
		t.Fatalf("expected 401 with nil body, got res=%v status=%d err=%v", res, status, err)
	}
}
