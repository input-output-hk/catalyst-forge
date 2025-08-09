package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	client "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/device"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

// Validate cookie issuance on device flow and refresh rotation via /sessions/refresh
func TestCookieSessionsFlow(t *testing.T) {
	api := getTestAPIURL()
	c := client.NewClient(api)
	ctx, cancel := newTestContext()
	defer cancel()

	// Start device flow and approve as admin
	initResp, err := c.Device().Init(ctx, &device.InitRequest{Name: "web", Platform: "darwin", Fingerprint: generateTestName("fp")})
	require.NoError(t, err)
	admin := newTestClient()
	_, _ = admin.Users().Create(ctx, &users.CreateUserRequest{Email: "admin@foundry.dev", Status: "active"})
	require.NoError(t, admin.Device().Approve(ctx, &device.ApproveRequest{UserCode: initResp.UserCode}))

	// Build http.Client with cookie jar to capture Set-Cookie
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar}

	// Poll device token endpoint directly to capture cookies
	bodyBytes, _ := json.Marshal(&device.TokenRequest{DeviceCode: initResp.DeviceCode})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, api+"/device/token", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// Wait for interval like device flow test to avoid slow_down
	time.Sleep(time.Duration(initResp.Interval+1) * time.Second)
	// Prefer cookie mode (no CLI headers)
	resp, err := hc.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	// Ensure cookies present (read from response headers to avoid domain mismatch with jar)
	var rtVal string
	for _, ck := range resp.Cookies() {
		if ck.Name == "cforge_rt" && ck.Value != "" {
			rtVal = ck.Value
		}
	}
	require.NotEmpty(t, rtVal)

	// Call /sessions/refresh to rotate cookies
	rreq, _ := http.NewRequestWithContext(ctx, http.MethodPost, api+"/sessions/refresh", nil)
	// In some http clients, cookie jar propagation may lag; explicitly attach RT cookie
	if rtVal != "" {
		rreq.Header.Set("Cookie", "cforge_rt="+rtVal)
	}
	rresp, err := hc.Do(rreq)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, rresp.StatusCode)

	// Call /sessions/logout to revoke and clear cookies
	lreq, _ := http.NewRequestWithContext(ctx, http.MethodPost, api+"/sessions/logout", nil)
	if rtVal != "" {
		lreq.Header.Set("Cookie", "cforge_rt="+rtVal)
	}
	lresp, err := hc.Do(lreq)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, lresp.StatusCode)
}
