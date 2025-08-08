package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	client "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/device"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/tokens"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

// Exercise refresh rotation and reuse detection using device flow to obtain an initial refresh token
func TestRefreshRotationReuseDetection(t *testing.T) {
	c := client.NewClient(getTestAPIURL())
	ctx, cancel := newTestContext()
	defer cancel()

	// Get a refresh via device flow
	initResp, err := c.Device().Init(ctx, &device.InitRequest{Name: "refresh-test", Platform: "linux", Fingerprint: generateTestName("fp")})
	require.NoError(t, err)
	admin := newTestClient()
	_, _ = admin.Users().Create(ctx, &users.CreateUserRequest{Email: "admin@foundry.dev", Status: "active"})
	require.NoError(t, admin.Device().Approve(ctx, &device.ApproveRequest{UserCode: initResp.UserCode}))
	time.Sleep(time.Duration(initResp.Interval+1) * time.Second)
	poll, err := c.Device().Token(ctx, &device.TokenRequest{DeviceCode: initResp.DeviceCode})
	require.NoError(t, err)
	require.NotEmpty(t, poll.Refresh)

	// rotate
	trc := client.NewClient(getTestAPIURL())
	pair, err := trc.Tokens().Refresh(ctx, &tokens.RefreshRequest{Refresh: poll.Refresh})
	require.NoError(t, err)
	require.NotEmpty(t, pair.Access)
	require.NotEmpty(t, pair.Refresh)

	// reuse old refresh should fail
	_, err = trc.Tokens().Refresh(ctx, &tokens.RefreshRequest{Refresh: poll.Refresh})
	require.Error(t, err)
}
