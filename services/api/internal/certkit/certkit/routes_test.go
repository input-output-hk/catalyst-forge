package certkit

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
	authctx "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit"
	rbac "github.com/input-output-hk/catalyst-forge/services/api/internal/authkit/rbac"
	"github.com/stretchr/testify/require"
)

// test PCA implementing certkit.PCAClient
type httpTestPCA struct {
	cert, chain string
	issued      bool
}

func (p *httpTestPCA) Issue(ctx context.Context, in PCAIssueInput) (string, error) {
	p.issued = true
	return "arn:test:1", nil
}
func (p *httpTestPCA) GetCertificate(ctx context.Context, caArn, certArn string) (string, string, error) {
	if !p.issued {
		return "", "", nil
	}
	return p.cert, p.chain, nil
}
func (p *httpTestPCA) GetCACertificate(ctx context.Context, caArn string) (string, string, error) {
	return "-----BEGIN CERTIFICATE-----\nFAKE-CA\n-----END CERTIFICATE-----\n", "", nil
}

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

func genCSR(t *testing.T, cn string, dns []string) string {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	csrTpl := x509.CertificateRequest{Subject: pkix.Name{CommonName: cn}, DNSNames: dns}
	der, err := x509.CreateCertificateRequest(rand.Reader, &csrTpl, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestRoutes_Sign_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pca := &httpTestPCA{cert: "-----BEGIN CERTIFICATE-----\nOK\n-----END CERTIFICATE-----\n"}
	cfg := DefaultConfig()
	cfg.CAArn = "arn:ca:test"
	cfg.PollInterval = 10 * time.Millisecond
	cfg.MaxWait = 250 * time.Millisecond

	deps := Deps{PCA: pca, Clock: sysClock{}, RBAC: fakeRBAC{dec: rbac.DecisionAllow}}
	mgr, err := New(cfg, deps)
	require.NoError(t, err)

	r := gin.New()
	mgr.RegisterRoutes(r.Group("/pki"))

	body := map[string]any{
		"csr_pem":  genCSR(t, "svc", []string{"ok.projectcatalyst.io"}),
		"template": "end-entity",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/pki/sign", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestRoutes_Sign_Denied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pca := &httpTestPCA{cert: "-----BEGIN CERTIFICATE-----\nOK\n-----END CERTIFICATE-----\n"}
	cfg := DefaultConfig()
	cfg.CAArn = "arn:ca:test"
	cfg.PollInterval = 10 * time.Millisecond
	cfg.MaxWait = 50 * time.Millisecond

	deps := Deps{PCA: pca, Clock: sysClock{}, RBAC: fakeRBAC{dec: rbac.DecisionDeny}}
	mgr, err := New(cfg, deps)
	require.NoError(t, err)

	r := gin.New()
	mgr.RegisterRoutes(r.Group("/pki"))

	body := map[string]any{
		"csr_pem": genCSR(t, "svc", []string{"bad.example.com"}),
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/pki/sign", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestRoutes_CA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pca := &httpTestPCA{cert: "-----BEGIN CERTIFICATE-----\nOK\n-----END CERTIFICATE-----\n"}
	cfg := DefaultConfig()
	cfg.CAArn = "arn:ca:test"
	deps := Deps{PCA: pca, Clock: sysClock{}}
	mgr, err := New(cfg, deps)
	require.NoError(t, err)

	r := gin.New()
	mgr.RegisterRoutes(r.Group("/pki"))

	req := httptest.NewRequest(http.MethodGet, "/pki/ca", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
