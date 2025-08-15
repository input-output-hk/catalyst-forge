package certs

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	authctx "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/authkit"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rate"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
	"github.com/input-output-hk/catalyst-forge/services/api/internal/certkit/certkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakePCA implements certkit.PCAClient for tests
type fakePCA struct {
	issued bool
	cert   string
	chain  string
	delay  time.Duration
}

func (p *fakePCA) Issue(ctx context.Context, in certkit.PCAIssueInput) (string, error) {
	p.issued = true
	return "arn:test:1", nil
}

func (p *fakePCA) GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error) {
	if p.delay > 0 {
		time.Sleep(p.delay)
	}
	if !p.issued {
		return "", "", nil
	}
	return p.cert, p.chain, nil
}

func (p *fakePCA) GetCACertificate(ctx context.Context, caArn string) (string, string, error) {
	return "-----BEGIN CERTIFICATE-----\nFAKE-CA\n-----END CERTIFICATE-----\n", "", nil
}

// fakeRBAC implements rbac.Manager minimally for programmatic Check
type fakeRBAC struct{ dec rbac.Decision }

func (f fakeRBAC) WithPolicyRegistry(reg *authctx.PolicyRegistry) *authctx.PolicyRegistry { return reg }
func (f fakeRBAC) RegisterResolver(pattern string, resolver rbac.ResourceResolver)        {}
func (f fakeRBAC) Check(ctx context.Context, subj rbac.Subject, action rbac.PermissionKey, res rbac.ResourceRef) (rbac.Decision, error) {
	return f.dec, nil
}
func (f fakeRBAC) Explain(ctx context.Context, subj rbac.Subject, action rbac.PermissionKey, res rbac.ResourceRef) (rbac.Decision, rbac.Trace, error) {
	return f.dec, rbac.Trace{}, nil
}
func (f fakeRBAC) Resolve(_ *gin.Context, _ string) (rbac.ResourceRef, bool, error) {
	return rbac.ResourceRef{}, false, nil
}

type sysClock struct{}

func (sysClock) Now() time.Time { return time.Now().UTC() }

func genCSR(t *testing.T, cn string, dns []string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csrTpl := x509.CertificateRequest{Subject: pkix.Name{CommonName: cn}, DNSNames: dns}
	der, err := x509.CreateCertificateRequest(rand.Reader, &csrTpl, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestPKI_Sign(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pca        *fakePCA
		rbacDec    rbac.Decision
		csrDNS     []string
		invalidCSR bool
		limiter    rate.Limiter
		ttl        string
		wantCode   int
	}{
		{name: "ok/basic", pca: &fakePCA{cert: "-----BEGIN CERTIFICATE-----\nOK\n-----END CERTIFICATE-----\n"}, rbacDec: rbac.DecisionAllow, csrDNS: []string{"ok.projectcatalyst.io"}, wantCode: http.StatusOK},
		{name: "deny/rbac", pca: &fakePCA{cert: "-----BEGIN CERTIFICATE-----\nOK\n-----END CERTIFICATE-----\n"}, rbacDec: rbac.DecisionDeny, csrDNS: []string{"bad.example.com"}, wantCode: http.StatusForbidden},
		{name: "timeout", pca: &fakePCA{cert: "", delay: 200 * time.Millisecond}, rbacDec: rbac.DecisionAllow, csrDNS: []string{"ok.projectcatalyst.io"}, wantCode: http.StatusGatewayTimeout},
		{name: "error/invalid_csr", pca: &fakePCA{}, rbacDec: rbac.DecisionAllow, invalidCSR: true, wantCode: http.StatusBadRequest},
		{name: "error/invalid_ttl", pca: &fakePCA{}, rbacDec: rbac.DecisionAllow, csrDNS: []string{"ok.projectcatalyst.io"}, ttl: "not-a-duration", wantCode: http.StatusBadRequest},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()

			cfg := certkit.DefaultConfig()
			cfg.CAArn = "arn:ca:test"
			cfg.PollInterval = 10 * time.Millisecond
			cfg.MaxWait = 100 * time.Millisecond

			deps := certkit.Deps{PCA: tc.pca, RBAC: fakeRBAC{dec: tc.rbacDec}, Clock: sysClock{}}
			if tc.limiter != nil {
				deps.Limiter = tc.limiter
			}
			ck, err := certkit.New(cfg, deps)
			require.NoError(t, err)
			ck.RegisterRoutes(r.Group("/pki"))

			csr := ""
			if tc.invalidCSR {
				csr = "garbage"
			} else {
				csr = genCSR(t, "svc", tc.csrDNS)
			}
			body := map[string]any{"csr_pem": csr, "template": "end-entity"}
			if tc.ttl != "" {
				body["ttl"] = tc.ttl
			}
			b, _ := json.Marshal(body)
			req := httptest.NewRequest(http.MethodPost, "/pki/sign", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantCode, w.Code)
			if tc.wantCode == http.StatusOK {
				assert.Contains(t, w.Body.String(), "certificate_pem", "should include cert pem")
			}
		})
	}
}

// memoryLimiter denies the first call, then allows; window ignored for simplicity
type memoryLimiter struct{ denied bool }

func (m *memoryLimiter) Allow(_ context.Context, _ rate.Key, _ int, _ time.Duration) (bool, int, time.Time, error) {
	if !m.denied {
		m.denied = true
		return false, 0, time.Now().UTC().Add(1 * time.Second), nil
	}
	return true, 1, time.Now().UTC().Add(1 * time.Second), nil
}

func TestPKI_Sign_RateLimit(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := certkit.DefaultConfig()
	cfg.CAArn = "arn:ca:test"
	deps := certkit.Deps{PCA: &fakePCA{cert: "-----BEGIN CERTIFICATE-----\nOK\n-----END CERTIFICATE-----\n"}, RBAC: fakeRBAC{dec: rbac.DecisionAllow}, Clock: sysClock{}, Limiter: &memoryLimiter{}}
	ck, err := certkit.New(cfg, deps)
	require.NoError(t, err)
	ck.RegisterRoutes(r.Group("/pki"))

	body := map[string]any{"csr_pem": genCSR(t, "svc", []string{"ok.projectcatalyst.io"})}
	b, _ := json.Marshal(body)

	// First call should be 429
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/pki/sign", bytes.NewReader(b))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusTooManyRequests, w1.Code)

	// Second call should pass
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/pki/sign", bytes.NewReader(b))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
}
