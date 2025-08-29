package handlers

import (
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/claims"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/clients/hydra"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/clients/kratos"
	"github.com/input-output-hk/catalyst-forge/services/ory/auth/internal/config"
)

// Handlers groups dependencies for HTTP handlers.
type Handlers struct {
	cfg          *config.Config
	httpClient   *http.Client
	hydraAdmin   *hydra.AdminClient
	kratosPublic *kratos.PublicClient
}

// NewHandlers constructs handlers with config dependency.
func NewHandlers(cfg *config.Config) *Handlers {
	transport := &http.Transport{Proxy: nil}
	// TLS root CAs from config if provided
	if cfg.TLS.CAFile != "" || cfg.TLS.CAPem != "" {
		pool, _ := x509.SystemCertPool()
		if pool == nil {
			pool = x509.NewCertPool()
		}
		if cfg.TLS.CAFile != "" {
			if pemBytes, err := os.ReadFile(cfg.TLS.CAFile); err == nil {
				pool.AppendCertsFromPEM(pemBytes)
			}
		}
		if cfg.TLS.CAPem != "" {
			pool.AppendCertsFromPEM([]byte(cfg.TLS.CAPem))
		}
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{RootCAs: pool}
		} else {
			transport.TLSClientConfig.RootCAs = pool
		}
	}
	client := &http.Client{Timeout: 5 * time.Second, Transport: transport}
	return &Handlers{
		cfg:          cfg,
		httpClient:   client,
		hydraAdmin:   hydra.NewAdminClient(cfg.Hydra.AdminURL, client),
		kratosPublic: kratos.NewPublicClient(cfg.Kratos.PublicURL, client),
	}
}

// Health returns ok.
func (h *Handlers) Health(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// isValidChallenge performs basic validation on hydra challenges.
func isValidChallenge(s string) bool {
	if s == "" {
		return false
	}
	if len(s) > 2048 {
		return false
	}
	if strings.ContainsAny(s, " \t\n\r") {
		return false
	}
	return true
}

func corrFields(c *gin.Context) map[string]any {
	fields := map[string]any{}
	if v, ok := c.Get("request_id"); ok {
		fields["request_id"] = v
	}
	if v, ok := c.Get("corr_login_challenge"); ok {
		fields["login_challenge"] = v
	}
	if v, ok := c.Get("corr_consent_challenge"); ok {
		fields["consent_challenge"] = v
	}
	return fields
}

// buildReturnTo builds an absolute URL back to the current handler, honoring
// proxy headers for scheme/host when present.
func buildReturnTo(c *gin.Context) string {
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}
	host := c.Request.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	u := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     c.FullPath(),
		RawQuery: c.Request.URL.RawQuery,
	}
	return u.String()
}

// Login handles GET /oauth2/login
func (h *Handlers) Login(c *gin.Context) {
	logger := c.MustGet("logger").(*slog.Logger)
	loginChallenge := c.Query("login_challenge")
	logger.Debug("login phase=start", "raw_query", c.Request.URL.RawQuery)

	if !isValidChallenge(loginChallenge) {
		logger.Debug("login phase=invalid_challenge", "length", len(loginChallenge))
		loginFailureTotal.Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login_challenge", "correlation": corrFields(c)})
		return
	}
	logger.Debug("login phase=challenge_valid", "length", len(loginChallenge))

	// Check Kratos session and decode identity
	logger.Debug("login phase=kratos_whoami_begin")
	whoami, status, err := h.kratosPublic.WhoAmIJSON(c.Request.Cookies())
	if err != nil {
		logger.Debug("login phase=kratos_whoami_error", "error", err.Error())
		loginFailureTotal.Inc()
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}
	if status != http.StatusOK || whoami == nil {
		// Redirect to Kratos login UI with return_to back here
		logger.Debug("login phase=redirect_kratos_login", "return_to", buildReturnTo(c))
		c.Redirect(http.StatusFound, h.kratosPublic.LoginRedirect(buildReturnTo(c)))
		return
	}
	logger.Debug("login phase=kratos_whoami_ok", "identity_id", whoami.Identity.ID)

	// Inspect login request
	logger.Debug("login phase=hydra_get_login_request_begin")
	lr, err := h.hydraAdmin.GetLoginRequest(loginChallenge)
	if err != nil {
		logger.Debug("login phase=hydra_get_login_request_error", "error", err.Error())
		loginFailureTotal.Inc()
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}
	logger.Debug("login phase=hydra_get_login_request_ok", "skip", lr.Skip, "subject", lr.Subject)

	// Fast-path
	if lr.Skip && lr.Subject != "" {
		logger.Debug("login phase=hydra_accept_login_fastpath_begin", "subject", lr.Subject)
		redirectTo, err := h.hydraAdmin.AcceptLoginRequest(loginChallenge, lr.Subject, true, h.cfg.Consent.RememberForSeconds)
		if err != nil {
			logger.Debug("login phase=hydra_accept_login_fastpath_error", "error", err.Error())
			loginFailureTotal.Inc()
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "correlation": corrFields(c)})
			return
		}
		logger.Debug("login phase=hydra_accept_login_ok", "redirect_to", redirectTo)
		c.Redirect(http.StatusFound, redirectTo)
		return
	}

	// Use the Kratos identity id as the subject
	subject := "user:" + whoami.Identity.ID
	logger.Debug("login phase=hydra_accept_login_begin", "subject", subject)
	redirectTo, err := h.hydraAdmin.AcceptLoginRequest(loginChallenge, subject, true, h.cfg.Consent.RememberForSeconds)
	if err != nil {
		logger.Debug("login phase=hydra_accept_login_error", "error", err.Error())
		loginFailureTotal.Inc()
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}

	loginAcceptTotal.Inc()
	logger.Debug("login phase=redirecting", "redirect_to", redirectTo)
	c.Redirect(http.StatusFound, redirectTo)
}

// ConsentGet handles GET /oauth2/consent
func (h *Handlers) ConsentGet(c *gin.Context) {
	consentChallenge := c.Query("consent_challenge")
	logger := c.MustGet("logger").(*slog.Logger)
	logger.Debug("consent_get phase=start", "raw_query", c.Request.URL.RawQuery)
	if !isValidChallenge(consentChallenge) {
		logger.Debug("consent_get phase=invalid_challenge", "length", len(consentChallenge))
		consentFailureTotal.Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid consent_challenge", "correlation": corrFields(c)})
		return
	}

	// Require valid Kratos session; fail-closed on error/non-200
	logger.Debug("consent_get phase=kratos_whoami_begin")
	whoami, status, err := h.kratosPublic.WhoAmIJSON(c.Request.Cookies())
	if err != nil || status != http.StatusOK || whoami == nil {
		if err != nil {
			logger.Debug("consent_get phase=kratos_whoami_error", "error", err.Error(), "status", status)
		} else {
			logger.Debug("consent_get phase=kratos_whoami_invalid", "status", status)
		}
		consentFailureTotal.Inc()
		c.JSON(http.StatusBadGateway, gin.H{"error": "kratos whoami failed", "correlation": corrFields(c)})
		return
	}
	logger.Debug("consent_get phase=kratos_whoami_ok", "identity_id", whoami.Identity.ID)

	idToken, accessExt := claims.MapKratosTraitsToTokens(whoami.Identity.Traits)
	if idToken == nil {
		idToken = map[string]any{}
	}
	if accessExt == nil {
		accessExt = map[string]any{}
	}

	body := hydra.AcceptConsentBody{
		GrantScope:               h.cfg.Consent.Scopes,
		GrantAccessTokenAudience: h.cfg.Consent.Audience,
		Remember:                 true,
		RememberFor:              h.cfg.Consent.RememberForSeconds,
		Session: map[string]any{
			"id_token":     idToken,
			"access_token": map[string]any{"ext": accessExt},
		},
	}
	redirectTo, err := h.hydraAdmin.AcceptConsentRequest(consentChallenge, body)
	if err != nil {
		logger.Debug("consent_get phase=accept_consent_error", "error", err.Error())
		consentFailureTotal.Inc()
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}

	consentAcceptTotal.Inc()
	logger.Debug("consent_get phase=redirecting", "redirect_to", redirectTo)
	c.Redirect(http.StatusFound, redirectTo)
}

// ConsentPost handles POST /oauth2/consent
func (h *Handlers) ConsentPost(c *gin.Context) {
	logger := c.MustGet("logger").(*slog.Logger)
	logger.Debug("consent_post phase=start")
	type consentRequest struct {
		ConsentChallenge string   `json:"consent_challenge" binding:"required"`
		GrantScope       []string `json:"grant_scope"`
		GrantAudience    []string `json:"grant_audience"`
	}
	var req consentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Debug("consent_post phase=bind_error", "error", err.Error())
		consentFailureTotal.Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "correlation": corrFields(c)})
		return
	}
	logger.Debug("consent_post phase=ok", "grant_scope", req.GrantScope, "grant_audience", req.GrantAudience)
	c.JSON(http.StatusOK, gin.H{
		"action":            "accept_consent",
		"consent_challenge": req.ConsentChallenge,
		"grant_scope":       req.GrantScope,
		"grant_audience":    req.GrantAudience,
		"session":           gin.H{"id_token": gin.H{}, "access_token": gin.H{"ext": gin.H{}}},
	})
}
