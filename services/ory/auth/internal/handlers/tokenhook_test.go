package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/config"
)

func makeJWT(payload map[string]any) string {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	b, _ := json.Marshal(payload)
	pl := base64.RawURLEncoding.EncodeToString(b)
	return hdr + "." + pl + "."
}

func TestTokenHook_GHA_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := config.Config{}
	cfg.Hydra.AdminURL = "http://example"
	cfg.Kratos.PublicURL = "http://example"

	eng := gin.New()
	h := NewHandlers(&cfg)
	eng.POST("/hydra/token-hook", h.TokenHook)

	w := httptest.NewRecorder()
	assertion := makeJWT(map[string]any{"repository": "org/repo", "ref": "refs/heads/main", "sha": "abc"})
	body := map[string]any{
		"grant_type": "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"request": map[string]any{
			"client":                          map[string]any{"client_id": "ci"},
			"requested_scope":                 []string{"openid"},
			"requested_access_token_audience": []string{"api://internal"},
			"payload":                         map[string]any{"assertion": assertion},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/hydra/token-hook", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
