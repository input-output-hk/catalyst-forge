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

    apiclient "github.com/input-output-hk/catalyst-forge/lib/foundry/client"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/auth"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/buildsessions"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/invites"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/users"
    "github.com/stretchr/testify/require"
    "github.com/input-output-hk/catalyst-forge/foundry/api/test/testutil"
)

// Unauthenticated requests to admin-only endpoints should be rejected.
func TestAuthz_UnauthenticatedUsersList(t *testing.T) {
    env := NewTestEnv(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    pub := apiclient.NewClient(env.BaseURL())
    _, err := pub.Users().List(ctx)
    require.Error(t, err)
}

// Limited-role user should receive 403 for admin-only actions.
func TestAuthz_ForbiddenForLimitedRole(t *testing.T) {
    env := NewTestEnv(t)
    admin := env.AdminClient()
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()

    // Create role with no user:read permission
    roleName := generateTestName("limited-role")
    _, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{Name: roleName, Permissions: []string{"read"}})
    require.NoError(t, err)

    // Create invite for a new user with the limited role
    email := generateTestEmail()
    inv, err := admin.Invites().Create(ctx, &invites.CreateInviteRequest{Email: email, Roles: []string{roleName}, TTL: "24h"})
    require.NoError(t, err)

    // Use raw HTTP client with cookies for refresh flow
    jar, err := cookiejar.New(nil)
    require.NoError(t, err)
    httpClient := &http.Client{Jar: jar}

    // Device registration init
    var initResp auth.DeviceRegistrationInitResponse
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/init", nil,
        &auth.DeviceRegistrationInitRequest{Token: inv.Token, InviteID: int(inv.ID)}, &initResp)
    require.NoError(t, err)

    // Generate key, build proof, and register
    priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    require.NoError(t, err)
    ts := time.Now().Unix()
    proof, err := buildDeviceRegisterProof(priv, initResp.DeviceID, initResp.Challenge, ts)
    require.NoError(t, err)
    jwk := makeECJWK(&priv.PublicKey)
    var regResp auth.DeviceRegisterResponse
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/devices/register", nil,
        &auth.DeviceRegisterRequest{DeviceID: initResp.DeviceID, DeviceName: "Limited", PublicKeyJWK: jwk, DeviceProof: proof, Timestamp: ts}, &regResp)
    require.NoError(t, err)

    // Use the access token for API client
    userClient := apiclient.NewClient(env.BaseURL(), apiclient.WithToken(regResp.AccessToken))

    // Attempt admin-only action
    _, err = userClient.Users().List(ctx)
    require.Error(t, err) // expected 403

    // Also attempt direct refresh/logout proof to ensure auth flow is intact
    u, _ := url.Parse(env.BaseURL())
    now := time.Now().Unix()
    refreshProof, err := buildRefreshProof(priv, initResp.DeviceID, u.Host, "POST", "/auth/refresh", now)
    require.NoError(t, err)
    _, err = testutil.DoJSON(httpClient, "POST", env.BaseURL()+"/auth/refresh", map[string]string{"X-Device-Id": initResp.DeviceID, "X-Device-Proof": refreshProof}, map[string]string{}, &auth.DeviceRefreshResponse{})
    require.NoError(t, err)
}

// Test comprehensive permission enforcement on multiple endpoints
func TestAuthz_PermissionEnforcementOnRepresentativeRoutes(t *testing.T) {
    env := NewTestEnv(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Test without token (expect 401)
    publicClient := apiclient.NewClient(env.BaseURL())
    
    testCases := []struct {
        name     string
        testFunc func() error
    }{
        {"Users List", func() error { _, err := publicClient.Users().List(ctx); return err }},
        {"Roles List", func() error { _, err := publicClient.Roles().List(ctx); return err }},
        {"Releases Create", func() error { 
            _, err := publicClient.Releases().Create(ctx, &releases.Release{
                Project: "test", 
                SourceRepo: "test/repo",
                SourceCommit: "abc123",
            }, false)
            return err 
        }},
    }

    for _, tc := range testCases {
        t.Run(tc.name+" without token", func(t *testing.T) {
            err := tc.testFunc()
            require.Error(t, err, "Should fail without authentication")
        })
    }
}

// Test that admin endpoints reject requests without proper tokens
func TestAuthz_AdminEndpointsRequireAuth(t *testing.T) {
    env := NewTestEnv(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    publicClient := apiclient.NewClient(env.BaseURL())
    
    // Test various admin endpoints
    t.Run("Users endpoints", func(t *testing.T) {
        _, err := publicClient.Users().List(ctx)
        require.Error(t, err, "Users list should require auth")
        
        _, err = publicClient.Users().Get(ctx, 123) // Use uint ID
        require.Error(t, err, "Users get should require auth")
        
        _, err = publicClient.Users().Create(ctx, &users.CreateUserRequest{
            Email: "test@example.com",
        })
        require.Error(t, err, "Users create should require auth")
    })
    
    t.Run("Roles endpoints", func(t *testing.T) {
        _, err := publicClient.Roles().List(ctx)
        require.Error(t, err, "Roles list should require auth")
        
        _, err = publicClient.Roles().Create(ctx, &users.CreateRoleRequest{
            Name:        "test-role",
            Permissions: []string{"read"},
        })
        require.Error(t, err, "Roles create should require auth")
    })
    
    t.Run("Build sessions", func(t *testing.T) {
        _, err := publicClient.BuildSessions().Create(ctx, &buildsessions.CreateRequest{
            OwnerType: "repo",
            OwnerID:   "test/repo",
            TTL:       "10m",
        })
        require.Error(t, err, "Build sessions create should require auth")
    })
}

// Test specific permission granularity
func TestAuthz_GranularPermissions(t *testing.T) {
    env := NewTestEnv(t)
    admin := env.AdminClient()
    ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
    defer cancel()
    
    // Create various roles with specific permissions
    testCases := []struct {
        roleName    string
        permissions []string
        canRead     bool
        canWrite    bool
        endpoint    string
    }{
        {
            roleName:    "user-reader",
            permissions: []string{"user:read"},
            canRead:     true,
            canWrite:    false,
            endpoint:    "users",
        },
        {
            roleName:    "deployment-writer",
            permissions: []string{"deployment:write", "deployment:read"},
            canRead:     true,
            canWrite:    true,
            endpoint:    "deployments",
        },
        {
            roleName:    "release-reader",
            permissions: []string{"release:read"},
            canRead:     true,
            canWrite:    false,
            endpoint:    "releases",
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.roleName, func(t *testing.T) {
            // Create role
            _, err := admin.Roles().Create(ctx, &users.CreateRoleRequest{
                Name:        generateTestName(tc.roleName),
                Permissions: tc.permissions,
            })
            require.NoError(t, err)
            
            // TODO: Create user with role and test access
            // This requires setting up a full user with the role
        })
    }
}

