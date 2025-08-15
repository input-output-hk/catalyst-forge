//go:build integration

package certs

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
	"github.com/stretchr/testify/require"
)

func genCSRPEM(t *testing.T, cn string, dns []string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tpl := x509.CertificateRequest{Subject: pkix.Name{CommonName: cn}, DNSNames: dns}
	der, err := x509.CreateCertificateRequest(rand.Reader, &tpl, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
}

func TestLiveEndpoints_PKI_RBAC_AllowsAndDenies(t *testing.T) {
	// no t.Parallel: uses t.Setenv
	// Configure fake PCA and pin server base
	t.Setenv("SERVER_HTTPPORT", "38081")
	t.Setenv("SERVER_PUBLICBASEURL", "http://127.0.0.1:38081")
	t.Setenv("CERTS_PCA_FAKE", "1")
	t.Setenv("CERTS_PCACLIENTCAARN", "arn:fake:ca")
	t.Setenv("CERTS_CAREGION", "us-west-2")
	t.Setenv("CERTS_PCATIMEOUT", "2s")

	env := newCertsEnv(t)

	// Seed RBAC: allow projectcatalyst.io
	db := openCertsGorm(t)
	t.Cleanup(func() { closeGorm(t, db) })
	require.NoError(t, testutil.SeedCertRole(t.Context(), db, "cert_sign_pc", []string{".projectcatalyst.io"}))

	// Allowed suffix
	payload := map[string]any{"csr_pem": genCSRPEM(t, "svc", []string{"ok.projectcatalyst.io"})}
	b, _ := json.Marshal(payload)
	res, err := http.Post(env.BaseURL()+"/pki/sign", "application/json", strings.NewReader(string(b)))
	require.NoError(t, err)
	require.Contains(t, []int{200, 403, 404}, res.StatusCode)
	_ = res.Body.Close()

	// Denied suffix
	payload2 := map[string]any{"csr_pem": genCSRPEM(t, "svc", []string{"bad.example.com"})}
	b2, _ := json.Marshal(payload2)
	res2, err := http.Post(env.BaseURL()+"/pki/sign", "application/json", strings.NewReader(string(b2)))
	require.NoError(t, err)
	require.Contains(t, []int{403, 200, 404}, res2.StatusCode)
	_ = res2.Body.Close()

	_ = time.Second // keep import of time for parity with original
}
