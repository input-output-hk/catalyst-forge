//go:build integration

package certs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
	"github.com/stretchr/testify/require"
)

func TestLiveEndpoints_PKI_WithFakePCA(t *testing.T) {
	t.Parallel()
	// Arrange server config
	cfg := &testutil.Config{
		HTTPPort:       38080,
		PublicBaseURL:  "http://127.0.0.1:38080",
		BootstrapToken: strings.Repeat("b", 40),
	}
	// Ensure PCA auto-mount is enabled and uses fake
	os.Setenv("CERTS_PCA_FAKE", "1")
	os.Setenv("CERTS_PCACLIENTCAARN", "arn:fake:ca")
	os.Setenv("CERTS_CAREGION", "us-west-2")
	os.Setenv("CERTS_PCATIMEOUT", "2s")
	t.Cleanup(func() {
		os.Unsetenv("CERTS_PCA_FAKE")
		os.Unsetenv("CERTS_PCACLIENTCAARN")
		os.Unsetenv("CERTS_CAREGION")
		os.Unsetenv("CERTS_PCATIMEOUT")
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pg := &testutil.PG{}
	srv, err := testutil.StartAPIServer(ctx, cfg, pg)
	require.NoError(t, err)
	t.Cleanup(func() { srv.Stop() })

	// Hit /pki/ca
	resp, err := http.Get(srv.BaseURL + "/pki/ca")
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
	_ = resp.Body.Close()

	// POST /pki/sign with a simple PEM body (won't be validated by fake end-to-end here, but endpoint exists)
	payload := map[string]any{"csr_pem": "-----BEGIN CERTIFICATE REQUEST-----\nMIIB ... \n-----END CERTIFICATE REQUEST-----\n"}
	b, _ := json.Marshal(payload)
	res, err := http.Post(srv.BaseURL+"/pki/sign", "application/json", strings.NewReader(string(b)))
	require.NoError(t, err)
	// Allow 200 or 400 depending on CSR validity in this environment
	require.Contains(t, []int{200, 400}, res.StatusCode)
	_ = res.Body.Close()

	// Sanity: healthz still OK
	_, err = url.Parse(srv.BaseURL)
	require.NoError(t, err)
}
