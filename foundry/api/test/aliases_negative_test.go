//go:build integration

package test

import (
    "testing"

    "github.com/stretchr/testify/require"
)

func TestAliases_Negative_CreateForNonexistentRelease(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    err := c.Aliases().Create(ctx, generateTestName("alias"), "nonexistent-release-id")
    require.Error(t, err)
}

func TestAliases_Negative_CreateWithEmptyName(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    // Attempt to create alias with empty name (should fail if server validates)
    err := c.Aliases().Create(ctx, "", "nonexistent-release-id")
    require.Error(t, err)
}

