//go:build integration

package auth_test

import (
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/stretchr/testify/require"

	tu "github.com/input-output-hk/catalyst-forge/services/api/test/testutil"
)

// TestAuth_Refresh_Positive_BrowserMode exercises a positive refresh flow using cookie jar + double-submit CSRF.
func TestAuth_Refresh_Positive_CLI_Mode(t *testing.T) {
	ctx := t.Context()

	cfg, err := tu.DefaultTestConfig()
	require.NoError(t, err)

	srv, err := tu.StartAPIServer(ctx, cfg, suite.PG)
	require.NoError(t, err)
	defer srv.Stop()

	// Create an HTTP client with a cookie jar to retain the refresh cookie set during bootstrap
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

	// Bootstrap admin (issues refresh cookie + returns optional access token)
	var boot map[string]any
	bootResp, err := tu.DoJSON(client, http.MethodPost, srv.BaseURL+"/api/v1/auth/bootstrap", nil, map[string]any{
		"email":           "admin+refresh@foundry.dev",
		"bootstrap_token": cfg.BootstrapToken,
	}, &boot)
	require.NoError(t, err)

	// Extract refresh token from Set-Cookie header (cookie is Secure and won’t be returned for http by the jar)
	var refresh string
	for _, c := range bootResp.Cookies() {
		if c.Name == "__Host-refresh_token" {
			refresh = c.Value
			break
		}
	}
	require.NotEmpty(t, refresh)

	// Call refresh in CLI mode using Authorization header to bypass CSRF
	headers := map[string]string{
		"X-CLI":         "1",
		"Authorization": "Refresh " + refresh,
	}
	var out map[string]any
	_, err = tu.DoJSON(client, http.MethodPost, srv.BaseURL+"/api/v1/auth/refresh", headers, nil, &out)
	require.NoError(t, err)
}
