package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/stretchr/testify/require"

	client "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/device"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

// Verify middleware accepts access token from cookie when Authorization header is absent
func TestMiddlewareAcceptsCookieJWT(t *testing.T) {
	api := getTestAPIURL()
	c := client.NewClient(api)
	ctx, cancel := newTestContext()
	defer cancel()

	// Bootstrap a session with cookies
	initResp, err := c.Device().Init(ctx, &device.InitRequest{Name: "mw", Platform: "linux", Fingerprint: generateTestName("fp")})
	require.NoError(t, err)
	admin := newTestClient()
	_, _ = admin.Users().Create(ctx, &users.CreateUserRequest{Email: "admin@foundry.dev", Status: "active"})
	require.NoError(t, admin.Device().Approve(ctx, &device.ApproveRequest{UserCode: initResp.UserCode}))
	bodyBytes, _ := json.Marshal(&device.TokenRequest{DeviceCode: initResp.DeviceCode})
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar}
	tokReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, api+"/device/token", bytes.NewReader(bodyBytes))
	tokReq.Header.Set("Content-Type", "application/json")
	tokResp, err := hc.Do(tokReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, tokResp.StatusCode)

	// Extract access token cookie (Secure cookies are not sent over HTTP by jar)
	var access string
	for _, ck := range tokResp.Cookies() {
		if ck.Name == "cforge_at" && ck.Value != "" {
			access = ck.Value
			break
		}
	}
	require.NotEmpty(t, access)

	// Call a protected endpoint without Authorization header; send cookie header explicitly
	getReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, api+"/auth/users", nil)
	getReq.Header.Set("Cookie", "cforge_at="+access)
	getResp, err := hc.Do(getReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, getResp.StatusCode)
}
