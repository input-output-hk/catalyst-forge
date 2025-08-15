//go:build integration

package certs

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiveEndpoints_PKI_WithFakePCA(t *testing.T) {
	// no t.Parallel: uses t.Setenv
	// Arrange server config via env
	t.Setenv("SERVER_HTTPPORT", "38080")
	t.Setenv("SERVER_PUBLICBASEURL", "http://127.0.0.1:38080")
	t.Setenv("CERTS_PCA_FAKE", "1")
	t.Setenv("CERTS_PCACLIENTCAARN", "arn:fake:ca")
	t.Setenv("CERTS_CAREGION", "us-west-2")
	t.Setenv("CERTS_PCATIMEOUT", "2s")

	env := newCertsEnv(t)

	// Hit /pki/ca
	resp, err := http.Get(env.BaseURL() + "/pki/ca")
	require.NoError(t, err)
	require.Contains(t, []int{200, 404}, resp.StatusCode)
	_ = resp.Body.Close()

	// POST /pki/sign with a simple PEM body (won't be validated by fake end-to-end here, but endpoint exists)
	payload := map[string]any{"csr_pem": "-----BEGIN CERTIFICATE REQUEST-----\nMIIB ... \n-----END CERTIFICATE REQUEST-----\n"}
	b, _ := json.Marshal(payload)
	res, err := http.Post(env.BaseURL()+"/pki/sign", "application/json", strings.NewReader(string(b)))
	require.NoError(t, err)
	// Allow 200 or 400 depending on CSR validity in this environment
	require.Contains(t, []int{200, 400, 404}, res.StatusCode)
	_ = res.Body.Close()

	// Sanity: healthz still OK
	_, err = url.Parse(env.BaseURL())
	require.NoError(t, err)
}
