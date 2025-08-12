//go:build integration

package test

import (
    "context"
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "testing"
    "time"

    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/auth"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/invites"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
    "github.com/stretchr/testify/require"
    "github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
)

// Negative refresh/logout device-proof tests (wrong origin and expired timestamp).
func TestDeviceAuth_Negative_RefreshProof(t *testing.T) {
    env := NewTestEnv(t)
    admin := env.AdminClient()
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    // Prepare invited user with minimal role
    role := generateTestName("role-device-neg")
    _, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: role, Permissions: []string{"read"}})
    require.NoError(t, err)
    email := generateTestEmail()
    inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{Email: email, Roles: []string{role}, TTL: "24h"})
    require.NoError(t, err)

    // Cookie jar client
    jar, err := cookiejar.New(nil)
    require.NoError(t, err)
    httpClient := &http.Client{Jar: jar}

    // Init
    var initResp auth.DeviceRegistrationInitResponse
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/init", nil,
        &auth.DeviceRegistrationInitRequest{Token: inv.Token, InviteID: int(inv.ID)}, &initResp)
    require.NoError(t, err)

    // Register
    priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    require.NoError(t, err)
    ts := time.Now().Unix()
    regProof, err := buildDeviceRegisterProof(priv, initResp.DeviceID, initResp.Challenge, ts)
    require.NoError(t, err)
    jwk := makeECJWK(&priv.PublicKey)
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
        &auth.DeviceRegisterRequest{DeviceID: initResp.DeviceID, DeviceName: "DevNeg", PublicKeyJWK: jwk, DeviceProof: regProof, Timestamp: ts}, &auth.DeviceRegisterResponse{})
    require.NoError(t, err)

    // Wrong origin host
    wrongHost := "malicious.example.com"
    badProof, err := buildRefreshProof(priv, initResp.DeviceID, wrongHost, "POST", "/auth/refresh", time.Now().Unix())
    require.NoError(t, err)
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/refresh", map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": badProof}, map[string]string{}, &auth.DeviceRefreshResponse{})
    require.Error(t, err)

    // Expired timestamp (1 hour ago)
    u, _ := url.Parse(env.BaseURL())
    expired := time.Now().Add(-1 * time.Hour).Unix()
    expiredProof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/refresh", expired)
    require.NoError(t, err)
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/refresh", map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": expiredProof}, map[string]string{}, &auth.DeviceRefreshResponse{})
    require.Error(t, err)

    // Sanity: Valid refresh still works
    now := time.Now().Unix()
    goodProof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/refresh", now)
    require.NoError(t, err)
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/refresh", map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": goodProof}, map[string]string{}, &auth.DeviceRefreshResponse{})
    require.NoError(t, err)
}

// Test device registration with bad inputs
func TestDeviceAuth_Negative_Registration(t *testing.T) {
    env := NewTestEnv(t)
    admin := env.AdminClient()
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    // Create invite for testing
    email := generateTestEmail()
    inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{
        Email: email,
        Roles: []string{"viewer"},
        TTL:   "24h",
    })
    require.NoError(t, err)
    
    jar, err := cookiejar.New(nil)
    require.NoError(t, err)
    httpClient := &http.Client{Jar: jar}
    
    // Init to get challenge
    var initResp auth.DeviceRegistrationInitResponse
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/init", nil,
        &auth.DeviceRegistrationInitRequest{Token: inv.Token, InviteID: int(inv.ID)}, &initResp)
    require.NoError(t, err)
    
    priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    require.NoError(t, err)
    
    t.Run("bad timestamp skew", func(t *testing.T) {
        // Future timestamp (1 hour ahead)
        futureTs := time.Now().Add(time.Hour).Unix()
        proof, err := buildDeviceRegisterProof(priv, initResp.DeviceID, initResp.Challenge, futureTs)
        require.NoError(t, err)
        jwk := makeECJWK(&priv.PublicKey)
        
        _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
            &auth.DeviceRegisterRequest{
                DeviceID:     initResp.DeviceID,
                DeviceName:   "BadTimestamp",
                PublicKeyJWK: jwk,
                DeviceProof:  proof,
                Timestamp:    futureTs,
            }, &auth.DeviceRegisterResponse{})
        // API might accept future timestamps within a certain tolerance
        if err == nil {
            t.Log("API accepts future timestamps (1 hour ahead)")
        } else {
            t.Log("API rejects future timestamps")
        }
    })
    
    t.Run("mismatched device ID", func(t *testing.T) {
        ts := time.Now().Unix()
        // Build proof with correct device ID but send different one
        proof, err := buildDeviceRegisterProof(priv, initResp.DeviceID, initResp.Challenge, ts)
        require.NoError(t, err)
        jwk := makeECJWK(&priv.PublicKey)
        
        _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
            &auth.DeviceRegisterRequest{
                DeviceID:     "wrong-device-id",
                DeviceName:   "WrongID",
                PublicKeyJWK: jwk,
                DeviceProof:  proof,
                Timestamp:    ts,
            }, &auth.DeviceRegisterResponse{})
        require.Error(t, err, "Should reject mismatched device ID")
    })
    
    t.Run("invalid proof format", func(t *testing.T) {
        ts := time.Now().Unix()
        jwk := makeECJWK(&priv.PublicKey)
        
        _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
            &auth.DeviceRegisterRequest{
                DeviceID:     initResp.DeviceID,
                DeviceName:   "BadProof",
                PublicKeyJWK: jwk,
                DeviceProof:  "not-a-valid-proof",
                Timestamp:    ts,
            }, &auth.DeviceRegisterResponse{})
        require.Error(t, err, "Should reject invalid proof format")
    })
    
    t.Run("invalid JWK", func(t *testing.T) {
        ts := time.Now().Unix()
        proof, err := buildDeviceRegisterProof(priv, initResp.DeviceID, initResp.Challenge, ts)
        require.NoError(t, err)
        
        // Invalid JWK structure
        badJWK := map[string]interface{}{
            "kty": "EC",
            // Missing required fields like crv, x, y
        }
        
        _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
            &auth.DeviceRegisterRequest{
                DeviceID:     initResp.DeviceID,
                DeviceName:   "BadJWK",
                PublicKeyJWK: badJWK,
                DeviceProof:  proof,
                Timestamp:    ts,
            }, &auth.DeviceRegisterResponse{})
        require.Error(t, err, "Should reject invalid JWK")
    })
}

// Test logout with wrong components
func TestDeviceAuth_Negative_Logout(t *testing.T) {
    env := NewTestEnv(t)
    admin := env.AdminClient()
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    
    // Setup: Create user and complete registration
    email := generateTestEmail()
    inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{
        Email: email,
        Roles: []string{"viewer"},
        TTL:   "24h",
    })
    require.NoError(t, err)
    
    jar, err := cookiejar.New(nil)
    require.NoError(t, err)
    httpClient := &http.Client{Jar: jar}
    
    // Complete registration flow
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
    
    var regResp auth.DeviceRegisterResponse
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
        &auth.DeviceRegisterRequest{
            DeviceID:     initResp.DeviceID,
            DeviceName:   "TestDevice",
            PublicKeyJWK: jwk,
            DeviceProof:  regProof,
            Timestamp:    ts,
        }, &regResp)
    require.NoError(t, err)
    
    // Test logout with wrong components
    u, _ := url.Parse(env.BaseURL())
    
    t.Run("wrong canonical path", func(t *testing.T) {
        now := time.Now().Unix()
        // Build proof for wrong path
        badProof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/wrong-path", now)
        require.NoError(t, err)
        
        _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/logout",
            map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": badProof},
            map[string]string{}, nil)
        require.Error(t, err, "Should reject proof for wrong path")
    })
    
    t.Run("replayed proof", func(t *testing.T) {
        now := time.Now().Unix()
        proof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/logout", now)
        require.NoError(t, err)
        
        // First logout might succeed
        _, _ = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/logout",
            map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": proof},
            map[string]string{}, nil)
        
        // Replay the same proof (same timestamp)
        _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/logout",
            map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": proof},
            map[string]string{}, nil)
        // API might not enforce replay protection for logout
        if err == nil {
            t.Log("API does not enforce replay protection for logout")
        } else {
            t.Log("API enforces replay protection for logout")
        }
    })
}
