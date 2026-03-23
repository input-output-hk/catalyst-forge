package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/golang-jwt/jwt/v5"
	apiclient "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/invites"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/tokens"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
)

// End-to-end invite → verify → KET → key register → challenge/login → protected call
func TestOnboardingFlow(t *testing.T) {
	admin := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()
	_, _ = admin.Users().Create(ctx, &users.CreateUserRequest{Email: "admin@foundry.dev", Status: "active"})

	email := generateTestEmail()
	roleName := generateTestName("e2e-role")

	// Ensure role exists and assign to user later
	_, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: roleName, Permissions: []string{"read", "user:read"}})
	require.NoError(t, err)

	// 1) Admin creates invite
	inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{Email: email, Roles: []string{roleName}, TTL: "24h"})
	require.NoError(t, err)
	require.NotZero(t, inv.ID)
	require.NotEmpty(t, inv.Token)

	// 2) User verifies invite (public endpoint)
	// Use a raw client with no token for public GET /verify
	pub := apiclient.NewClient(getTestAPIURL())
	require.NoError(t, pub.Invites().Verify(ctx, inv.Token))

	// 3) Bootstrap KET for the user (admin context acceptable for test)
	ket, err := admin.Keys().BootstrapKET(ctx, &users.BootstrapKETRequest{Email: email})
	require.NoError(t, err)
	require.NotEmpty(t, ket.KET)
	require.NotEmpty(t, ket.Nonce)

	// 4) Generate ed25519 keypair; sign nonce
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	nonceBytes, err := base64.RawURLEncoding.DecodeString(ket.Nonce)
	require.NoError(t, err)
	sig := ed25519.Sign(privKey, nonceBytes)
	kid := generateTestKid()
	pubB64 := base64.StdEncoding.EncodeToString(pubKey)

	// 5) Register the key via KET
	uk, err := admin.Keys().RegisterWithKET(ctx, &users.RegisterWithKETClientRequest{
		KET: ket.KET, Kid: kid, PubKeyB64: pubB64, SigBase64: base64.StdEncoding.EncodeToString(sig),
	})
	require.NoError(t, err)
	require.Equal(t, kid, uk.Kid)

	// 6) Challenge/login using the new key
	ch, err := admin.Auth().CreateChallenge(ctx, &auth.ChallengeRequest{Email: email, Kid: kid})
	require.NoError(t, err)
	// Parse the JWT to extract nonce and sign it
	token, _, err := new(jwt.Parser).ParseUnverified(ch.Token, jwt.MapClaims{})
	require.NoError(t, err)
	claims := token.Claims.(jwt.MapClaims)
	nonce, _ := claims["nonce"].(string)
	sig2 := ed25519.Sign(privKey, []byte(nonce))
	lr, err := admin.Auth().Login(ctx, &auth.LoginRequest{Token: ch.Token, Signature: base64.StdEncoding.EncodeToString(sig2)})
	require.NoError(t, err)
	require.NotEmpty(t, lr.Token)

	// 7) Assign role to user to ensure permissions then use token
	urec, err := admin.Users().GetByEmail(ctx, email)
	require.NoError(t, err)
	rrec, err := admin.Roles().GetByName(ctx, roleName)
	require.NoError(t, err)
	require.NoError(t, admin.Roles().AssignUser(ctx, urec.ID, rrec.ID))
	// Use access token to call a protected endpoint
	userClient := apiclient.NewClient(getTestAPIURL(), apiclient.WithToken(lr.Token))
	fetched, err := userClient.Users().GetByEmail(ctx, email)
	require.NoError(t, err)
	assert.Equal(t, email, fetched.Email)

	// 8) Test refresh token rotation path quickly using tokens client (if we had refresh here we'd test it).
	_ = tokens.RefreshRequest{}
}
