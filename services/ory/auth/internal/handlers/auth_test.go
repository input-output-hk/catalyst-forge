package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/config"
)

func TestLogin_FastPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Fake Hydra and Kratos by pointing config to test servers
	hydraSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth2/auth/requests/login" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"skip": true, "subject": "user:1"})
		case r.URL.Path == "/oauth2/auth/requests/login/accept" && r.Method == http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://hydra/redirect"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer hydraSrv.Close()

	kratosSrv := httptest.NewServer(http.NotFoundHandler())
	defer kratosSrv.Close()

	cfg := config.Config{}
	cfg.Hydra.AdminURL = hydraSrv.URL
	cfg.Kratos.PublicURL = kratosSrv.URL

	eng := gin.New()
	eng.Use(func(c *gin.Context) { c.Set("logger", slog.Default()); c.Next() })
	h := NewHandlers(&cfg)
	eng.GET("/oauth2/login", h.Login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/login?login_challenge=abc", nil)
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestConsent_FailClosedOnWhoAmI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hydraSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer hydraSrv.Close()

	kratosSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer kratosSrv.Close()

	cfg := config.Config{}
	cfg.Hydra.AdminURL = hydraSrv.URL
	cfg.Kratos.PublicURL = kratosSrv.URL
	cfg.Consent.RememberForSeconds = 300
	cfg.Consent.Scopes = []string{"openid"}

	eng := gin.New()
	eng.Use(func(c *gin.Context) { c.Set("logger", slog.Default()); c.Next() })
	h := NewHandlers(&cfg)
	eng.GET("/oauth2/consent", h.ConsentGet)

	w := httptest.NewRecorder()
	u := url.URL{Path: "/oauth2/consent"}
	q := u.Query()
	q.Set("consent_challenge", "abc")
	u.RawQuery = q.Encode()
	req := httptest.NewRequest(http.MethodGet, u.String(), nil)
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
}

func TestLogin_NoSession_RedirectsToKratos(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hydraSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth2/auth/requests/login" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"skip": false})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer hydraSrv.Close()

	kratosSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer kratosSrv.Close()

	cfg := config.Config{}
	cfg.Hydra.AdminURL = hydraSrv.URL
	cfg.Kratos.PublicURL = kratosSrv.URL

	eng := gin.New()
	eng.Use(func(c *gin.Context) { c.Set("logger", slog.Default()); c.Next() })
	h := NewHandlers(&cfg)
	eng.GET("/oauth2/login", h.Login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/login?login_challenge=abc", nil)
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect to Kratos, got %d", w.Code)
	}
}

func TestLogin_WithSession_Accepts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hydraSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth2/auth/requests/login" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"skip": false})
		case r.URL.Path == "/oauth2/auth/requests/login/accept" && r.Method == http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://hydra/redirect"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer hydraSrv.Close()

	kratosSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"identity": map[string]any{"id": "id-123", "traits": map[string]any{"email": "user@example.com"}},
		})
	}))
	defer kratosSrv.Close()

	cfg := config.Config{}
	cfg.Hydra.AdminURL = hydraSrv.URL
	cfg.Kratos.PublicURL = kratosSrv.URL
	cfg.Consent.RememberForSeconds = 300

	eng := gin.New()
	eng.Use(func(c *gin.Context) { c.Set("logger", slog.Default()); c.Next() })
	h := NewHandlers(&cfg)
	eng.GET("/oauth2/login", h.Login)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oauth2/login?login_challenge=abc", nil)
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}

func TestConsent_Success_Path(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hydraSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth2/auth/requests/consent" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"requested_scope": []string{"openid"}, "requested_access_token_audience": []string{"api://internal"}})
		case r.URL.Path == "/oauth2/auth/requests/consent/accept" && r.Method == http.MethodPut:
			_ = json.NewEncoder(w).Encode(map[string]any{"redirect_to": "https://hydra/redirect"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer hydraSrv.Close()

	kratosSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"identity": map[string]any{"id": "id-123", "traits": map[string]any{"email": "user@example.com", "domain": "EXAMPLE.com"}},
		})
	}))
	defer kratosSrv.Close()

	cfg := config.Config{}
	cfg.Hydra.AdminURL = hydraSrv.URL
	cfg.Kratos.PublicURL = kratosSrv.URL
	cfg.Consent.RememberForSeconds = 300
	cfg.Consent.Scopes = []string{"openid"}

	eng := gin.New()
	eng.Use(func(c *gin.Context) { c.Set("logger", slog.Default()); c.Next() })
	h := NewHandlers(&cfg)
	eng.GET("/oauth2/consent", h.ConsentGet)

	w := httptest.NewRecorder()
	u := url.URL{Path: "/oauth2/consent"}
	q := u.Query()
	q.Set("consent_challenge", "abc")
	u.RawQuery = q.Encode()
	req := httptest.NewRequest(http.MethodGet, u.String(), nil)
	eng.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
}
