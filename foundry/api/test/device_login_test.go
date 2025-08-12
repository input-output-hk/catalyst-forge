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

// wire types for new endpoints (until typed client is updated)
type loginInitResp struct {
    Challenge string    `json:"challenge"`
    Algorithm string    `json:"alg"`
    ExpiresAt time.Time `json:"expires_at"`
    DeviceID  string    `json:"device_id"`
}

type loginResp struct {
    AccessToken string                 `json:"access_token"`
    TokenType   string                 `json:"token_type"`
    ExpiresIn   int                    `json:"expires_in"`
    User        map[string]interface{} `json:"user"`
}

// Test returning-device login happy path: register, logout, login via challenge
func TestDeviceLogin_TC_ReturningDeviceFlow(t *testing.T) {
    env := NewTestEnv(t)
    admin := env.AdminClient()
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Invite setup
    email := generateTestEmail()
    role := generateTestName("role-login")
    _, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: role, Permissions: []string{"read", "user:read"}})
    require.NoError(t, err)
    inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{Email: email, Roles: []string{role}, TTL: "24h"})
    require.NoError(t, err)

    // Cookie jar client to hold refresh cookie
    jar, err := cookiejar.New(nil)
    require.NoError(t, err)
    httpClient := &http.Client{Jar: jar}

    // Start registration
    var initResp auth.DeviceRegistrationInitResponse
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/init", nil,
        &auth.DeviceRegistrationInitRequest{Token: inv.Token, InviteID: int(inv.ID)}, &initResp)
    require.NoError(t, err)

    // Device key and registration
    priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    require.NoError(t, err)
    ts := time.Now().Unix()
    regProof, err := buildDeviceRegisterProof(priv, initResp.DeviceID, initResp.Challenge, ts)
    require.NoError(t, err)
    jwk := makeECJWK(&priv.PublicKey)
    var regResp auth.DeviceRegisterResponse
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
        &auth.DeviceRegisterRequest{DeviceID: initResp.DeviceID, DeviceName: "Login Dev", PublicKeyJWK: jwk, DeviceProof: regProof, Timestamp: ts}, &regResp)
    require.NoError(t, err)
    require.NotEmpty(t, regResp.AccessToken)

    // Verify access works
    userClient := apiclient.NewClient(env.BaseURL(), apiclient.WithToken(regResp.AccessToken))
    fetched, err := userClient.Users().GetByEmail(ctx, email)
    require.NoError(t, err)
    require.Equal(t, email, fetched.Email)

    // Logout to clear the refresh cookie and revoke family
    u, _ := url.Parse(env.BaseURL())
    logoutProof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/logout", time.Now().Unix())
    require.NoError(t, err)
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/logout",
        map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": logoutProof}, map[string]string{}, nil)
    require.NoError(t, err)

    // Returning-device login init
    var lInit loginInitResp
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/login/init", nil,
        map[string]string{"device_id": initResp.DeviceID}, &lInit)
    require.NoError(t, err)
    require.Equal(t, initResp.DeviceID, lInit.DeviceID)
    require.Equal(t, "ES256", lInit.Algorithm)
    require.NotEmpty(t, lInit.Challenge)

    // Sign the challenge using raw r||s signature, base64url-encoded
    ts2 := time.Now().Unix()
    loginSig, err := buildRegisterChallengeRawSig(priv, initResp.DeviceID, lInit.Challenge, ts2)
    require.NoError(t, err)

    // Complete login
    var lResp loginResp
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/login", nil,
        map[string]any{
            "device_id":    initResp.DeviceID,
            "timestamp":    ts2,
            "device_proof": loginSig,
        }, &lResp)
    require.NoError(t, err)
    require.NotEmpty(t, lResp.AccessToken)

    // Access token works after login
    userClient2 := apiclient.NewClient(env.BaseURL(), apiclient.WithToken(lResp.AccessToken))
    fetched2, err := userClient2.Users().GetByEmail(ctx, email)
    require.NoError(t, err)
    require.Equal(t, email, fetched2.Email)
}

// Negative: invalid signature should fail login
func TestDeviceLogin_Negative_BadProof(t *testing.T) {
    env := NewTestEnv(t)
    admin := env.AdminClient()
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    email := generateTestEmail()
    role := generateTestName("role-login-neg")
    _, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: role, Permissions: []string{"read"}})
    require.NoError(t, err)
    inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{Email: email, Roles: []string{role}, TTL: "24h"})
    require.NoError(t, err)

    jar, err := cookiejar.New(nil)
    require.NoError(t, err)
    httpClient := &http.Client{Jar: jar}

    // Register device
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
        &auth.DeviceRegisterRequest{DeviceID: initResp.DeviceID, DeviceName: "LoginNeg Dev", PublicKeyJWK: jwk, DeviceProof: regProof, Timestamp: ts}, &auth.DeviceRegisterResponse{})
    require.NoError(t, err)

    // Init login
    var lInit loginInitResp
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/login/init", nil,
        map[string]string{"device_id": initResp.DeviceID}, &lInit)
    require.NoError(t, err)

    // Send completely invalid proof
    badSig := "not-base64url"
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/login", nil,
        map[string]any{"device_id": initResp.DeviceID, "timestamp": time.Now().Unix(), "device_proof": badSig}, &loginResp{})
    require.Error(t, err)
}

// buildRegisterChallengeRawSig builds base64url(r||s) for the DEVICE-REGISTER canonical string
func buildRegisterChallengeRawSig(priv *ecdsa.PrivateKey, deviceID, challenge string, ts int64) (string, error) {
    canonical := fmt.Sprintf("DEVICE-REGISTER\n%d\n%s\n%s", ts, deviceID, challenge)
    h := sha256.Sum256([]byte(canonical))
    r, s, err := ecdsa.Sign(rand.Reader, priv, h[:])
    if err != nil {
        return "", err
    }
    rb := r.FillBytes(make([]byte, 32))
    sb := s.FillBytes(make([]byte, 32))
    sig := append(rb, sb...)
    return base64.RawURLEncoding.EncodeToString(sig), nil
}

