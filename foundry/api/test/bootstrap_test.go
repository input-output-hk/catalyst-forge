//go:build integration

package test

import (
    "testing"
    "time"
    "context"

    "github.com/stretchr/testify/require"
)

// TestBootstrapAdmin_TC validates that we can bootstrap an admin account and use it.
func TestBootstrapAdmin_TC(t *testing.T) {
    env := NewTestEnv(t)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Verify the admin user exists and is retrievable by email
    admin := env.AdminClient()
    u, err := admin.Users().GetByEmail(ctx, env.AdminEmail)
    require.NoError(t, err)
    require.Equal(t, env.AdminEmail, u.Email)

    // Verify admin has functional permissions by performing a simple privileged action
    // List roles and ensure the built-in "admin" role exists
    r, err := admin.Roles().GetByName(ctx, "admin")
    require.NoError(t, err)
    require.Equal(t, "admin", r.Name)
}

