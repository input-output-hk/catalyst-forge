package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	client "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/device"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

func TestDeviceFlow(t *testing.T) {
	c := client.NewClient(getTestAPIURL())
	ctx, cancel := newTestContext()
	defer cancel()

	// init
	initResp, err := c.Device().Init(ctx, &device.InitRequest{Name: "test", Platform: "darwin", Fingerprint: generateTestName("fp")})
	require.NoError(t, err)
	require.NotEmpty(t, initResp.DeviceCode)
	require.NotEmpty(t, initResp.UserCode)

	// first poll: pending (API returns 401 with error payload)
	_, err = c.Device().Token(ctx, &device.TokenRequest{DeviceCode: initResp.DeviceCode})
	require.Error(t, err)
	if apiErr, ok := err.(*client.APIError); ok {
		assert.Equal(t, "authorization_pending", apiErr.ErrorMessage)
	} else {
		t.Fatalf("expected APIError, got %T", err)
	}

	// ensure admin subject exists and has required permissions; then approve (needs auth; use admin client token)
	admin := newTestClient()
	_, _ = admin.Users().Create(ctx, &users.CreateUserRequest{Email: "admin@foundry.dev", Status: "active"})
	roleName := generateTestName("admin-device")
	role, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: roleName, Permissions: []string{"read", "user:read"}})
	require.NoError(t, err)
	adminUser, err := admin.Users().GetByEmail(ctx, "admin@foundry.dev")
	require.NoError(t, err)
	require.NoError(t, admin.Roles().AssignUser(ctx, adminUser.ID, role.ID))
	require.NoError(t, admin.Device().Approve(ctx, &device.ApproveRequest{UserCode: initResp.UserCode}))

	// wait for interval to avoid slow_down
	time.Sleep(time.Duration(initResp.Interval+1) * time.Second)

	// second poll: should return tokens
	poll2, err := c.Device().Token(ctx, &device.TokenRequest{DeviceCode: initResp.DeviceCode})
	require.NoError(t, err)
	require.NotEmpty(t, poll2.Access)
	require.NotEmpty(t, poll2.Refresh)

	// verify access token works against a protected endpoint (list users)
	authed := client.NewClient(getTestAPIURL(), client.WithToken(poll2.Access))
	_, err = authed.Users().List(ctx)
	require.NoError(t, err)
}
