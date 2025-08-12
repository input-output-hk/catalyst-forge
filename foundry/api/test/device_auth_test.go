//go:build integration

package test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
	apiclient "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/auth"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/invites"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
	"github.com/stretchr/testify/require"
)

// Reuse buildDeviceRegisterProof and makeECJWK from existing test helpers in this package.

// Minimal happy-path device registration using Testcontainers-backed server.
func TestDeviceAuth_TC_DeviceRegistrationFlow(t *testing.T) {
	env := NewTestEnv(t)
	admin := env.AdminClient()

	ctx, cancel := newTestContext()
	defer cancel()
	email := generateTestEmail()
	roleName := generateTestName("tc-device-auth-role")

	// Create role and invite
	_, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: roleName, Permissions: []string{"read", "user:read"}})
	require.NoError(t, err)
	inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{Email: email, Roles: []string{roleName}, TTL: "24h"})
	require.NoError(t, err)

	// Public client (no token)
	pub := apiclient.NewClient(env.BaseURL())

	// Device registration init
	initResp, err := pub.Auth().DeviceRegistrationInit(ctx, &auth.DeviceRegistrationInitRequest{Token: inv.Token, InviteID: int(inv.ID)})
	require.NoError(t, err)

	// Generate device key
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ts := time.Now().Unix()
	proof, err := buildDeviceRegisterProof(privKey, initResp.DeviceID, initResp.Challenge, ts)
	require.NoError(t, err)

	jwk := makeECJWK(&privKey.PublicKey)

	regResp, err := pub.Auth().DeviceRegister(ctx, &auth.DeviceRegisterRequest{
		DeviceID:     initResp.DeviceID,
		DeviceName:   "TC Test Device",
		PublicKeyJWK: jwk,
		DeviceProof:  proof,
		Timestamp:    ts,
	})
	require.NoError(t, err)
	require.NotEmpty(t, regResp.AccessToken)

	// Verify token works
	userClient := apiclient.NewClient(env.BaseURL(), apiclient.WithToken(regResp.AccessToken))
	fetched, err := userClient.Users().GetByEmail(ctx, email)
	require.NoError(t, err)
	require.Equal(t, email, fetched.Email)
}

func TestDeviceAuth_TC_RefreshAndLogout(t *testing.T) {
	env := NewTestEnv(t)
	admin := env.AdminClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	email := generateTestEmail()
	role := generateTestName("tc-role")
	_, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: role, Permissions: []string{"read", "user:read"}})
	require.NoError(t, err)
	inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{Email: email, Roles: []string{role}, TTL: "24h"})
	require.NoError(t, err)

	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	httpClient := &http.Client{Jar: jar}

	var initResp auth.DeviceRegistrationInitResponse
	_, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/init", nil,
		&auth.DeviceRegistrationInitRequest{Token: inv.Token, InviteID: int(inv.ID)}, &initResp)
	require.NoError(t, err)

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ts := time.Now().Unix()
	regProof, err := buildDeviceRegisterProof(priv, initResp.DeviceID, initResp.Challenge, ts)
	require.NoError(t, err)
	jwk := makeECJWK(&priv.PublicKey)
	_, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
		&auth.DeviceRegisterRequest{DeviceID: initResp.DeviceID, DeviceName: "TC Dev", PublicKeyJWK: jwk, DeviceProof: regProof, Timestamp: ts}, &auth.DeviceRegisterResponse{})
	require.NoError(t, err)

	u, _ := url.Parse(env.BaseURL())
	now := time.Now().Unix()
	refreshProof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/refresh", now)
	require.NoError(t, err)
	headers := map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": refreshProof}
	var refreshResp auth.DeviceRefreshResponse
	_, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/refresh", headers, map[string]string{}, &refreshResp)
	require.NoError(t, err)

	userClient := apiclient.NewClient(env.BaseURL(), apiclient.WithToken(refreshResp.AccessToken))
	fetched, err := userClient.Users().GetByEmail(ctx, email)
	require.NoError(t, err)
	require.Equal(t, email, fetched.Email)

	logoutProof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/logout", time.Now().Unix())
	require.NoError(t, err)
	_, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/logout",
		map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": logoutProof}, map[string]string{}, nil)
	require.NoError(t, err)
}


// buildDeviceRegisterProof builds the device proof for registration in the format:
//
//	timestamp.base64url(ASN.1 DER ECDSA signature over canonical string)
//
// Canonical string: "DEVICE-REGISTER\n<timestamp>\n<device_id>\n<challenge>"
func buildDeviceRegisterProof(priv *ecdsa.PrivateKey, deviceID string, challenge string, ts int64) (string, error) {
	canonical := fmt.Sprintf("DEVICE-REGISTER\n%d\n%s\n%s", ts, deviceID, challenge)
	hash := sha256.Sum256([]byte(canonical))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, hash[:])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%s", ts, base64.RawURLEncoding.EncodeToString(sig)), nil
}

// buildRefreshProof builds the device proof for refresh/logout in the format:
//
//	timestamp.base64url(r||s) where r||s is 64-byte raw concatenation
//
// Canonical string: "AUTH-REFRESH\n<timestamp>\n<device_id>\n<origin>\n<method> <path>"
func buildRefreshProof(priv *ecdsa.PrivateKey, deviceID string, origin string, method string, path string, ts int64) (string, error) {
	canonical := fmt.Sprintf("AUTH-REFRESH\n%d\n%s\n%s\n%s %s", ts, deviceID, origin, method, path)
	hash := sha256.Sum256([]byte(canonical))
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	if err != nil {
		return "", err
	}
	rb := r.FillBytes(make([]byte, 32))
	sb := s.FillBytes(make([]byte, 32))
	sig := append(rb, sb...)
	return fmt.Sprintf("%d.%s", ts, base64.RawURLEncoding.EncodeToString(sig)), nil
}

// makeECJWK creates a minimal EC P-256 JWK map from an ECDSA public key
func makeECJWK(pub *ecdsa.PublicKey) map[string]interface{} {
	x := base64.RawURLEncoding.EncodeToString(pub.X.Bytes())
	y := base64.RawURLEncoding.EncodeToString(pub.Y.Bytes())
	return map[string]interface{}{
		"kty": "EC",
		"crv": "P-256",
		"x":   x,
		"y":   y,
	}
}

