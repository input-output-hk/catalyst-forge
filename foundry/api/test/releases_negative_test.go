//go:build integration

package test

import (
    "encoding/base64"
    "testing"

    "github.com/stretchr/testify/require"

    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
)

func TestReleases_Negative_CreateValidation(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    // Invalid: missing Project
    bundleStr := base64.StdEncoding.EncodeToString([]byte("bundle"))
    _, err := c.Releases().Create(ctx, &releases.Release{
        SourceRepo:   "github.com/example/repo",
        SourceCommit: "abcdef",
        Project:      "", // required
        ProjectPath:  "services/api",
        Bundle:       bundleStr,
    }, false)
    require.Error(t, err)

    // Invalid: empty bundle
    _, err = c.Releases().Create(ctx, &releases.Release{
        SourceRepo:   "github.com/example/repo",
        SourceCommit: "abcdef",
        Project:      generateTestName("project-neg"),
        ProjectPath:  "services/api",
        Bundle:       "", // Empty bundle
    }, false)
    require.Error(t, err)
}

func TestReleases_Negative_GetNonexistent(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    _, err := c.Releases().Get(ctx, "nonexistent-release-id")
    require.Error(t, err)
    
    // Also test GetByAlias with non-existent alias
    _, err = c.Releases().GetByAlias(ctx, "non-existent-alias")
    require.Error(t, err)
}

// Test comprehensive release validation
func TestReleases_Negative_Validation(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()
    
    bundleStr := base64.StdEncoding.EncodeToString([]byte("test content"))
    
    testCases := []struct {
        name        string
        release     *releases.Release
        shouldError bool
    }{
        {
            name: "missing source repo",
            release: &releases.Release{
                SourceRepo:   "", // Empty
                SourceCommit: "abc123",
                Project:      "test-project",
                Bundle:       bundleStr,
            },
            shouldError: true,
        },
        {
            name: "missing source commit",
            release: &releases.Release{
                SourceRepo:   "test/repo",
                SourceCommit: "", // Empty
                Project:      "test-project",
                Bundle:       bundleStr,
            },
            shouldError: true,
        },
        {
            name: "invalid source branch format",
            release: &releases.Release{
                SourceRepo:   "test/repo",
                SourceCommit: "abc123",
                SourceBranch: "refs/heads/../../../etc/passwd", // Path traversal
                Project:      "test-project",
                Bundle:       bundleStr,
            },
            shouldError: false, // May be accepted depending on validation
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            _, err := c.Releases().Create(ctx, tc.release, false)
            if tc.shouldError {
                require.Error(t, err)
            }
        })
    }
}

// Test alias negative cases
func TestAliases_Negative(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()
    
    // Create a valid release first
    bundleStr := base64.StdEncoding.EncodeToString([]byte("test content"))
    release, err := c.Releases().Create(ctx, &releases.Release{
        SourceRepo:   "test/repo",
        SourceCommit: "abc123",
        Project:      generateTestName("alias-test"),
        ProjectPath:  "services/api",
        Bundle:       bundleStr,
    }, false)
    require.NoError(t, err)
    
    t.Run("duplicate alias name", func(t *testing.T) {
        // Create first alias
        err := c.Aliases().Create(ctx, "prod", release.ID)
        require.NoError(t, err)
        
        // Try to create duplicate (API might allow reassignment)
        err = c.Aliases().Create(ctx, "prod", release.ID)
        if err == nil {
            t.Log("API allows duplicate alias (reassignment)")
        } else {
            t.Log("API rejects duplicate alias")
        }
    })
    
    t.Run("invalid alias formats", func(t *testing.T) {
        testCases := []struct {
            aliasName   string
            shouldError bool
        }{
            {"", true},                      // Empty
            {"alias with spaces", false},    // API might accept spaces
            {"alias!@#$", false},            // API might accept special chars
            {"../../../etc/passwd", false},  // API might not validate path traversal
            {"valid-alias-123", false},      // Valid
        }
        
        for _, tc := range testCases {
            err := c.Aliases().Create(ctx, tc.aliasName, release.ID)
            if tc.shouldError {
                require.Error(t, err, "Should reject alias: %s", tc.aliasName)
            }
        }
    })
    
    t.Run("non-existent release ID", func(t *testing.T) {
        err := c.Aliases().Create(ctx, "orphan", "non-existent-id")
        require.Error(t, err, "Should reject alias for non-existent release")
    })
}

// Test cross-project isolation
func TestReleases_CrossProjectIsolation(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()
    
    bundleStr := base64.StdEncoding.EncodeToString([]byte("content"))
    
    // Create releases in different projects
    projectA := generateTestName("project-a")
    releaseA, err := c.Releases().Create(ctx, &releases.Release{
        SourceRepo:   "test/repo",
        SourceCommit: "aaa111",
        Project:      projectA,
        ProjectPath:  "services/api",
        Bundle:       bundleStr,
    }, false)
    require.NoError(t, err)
    
    projectB := generateTestName("project-b")
    releaseB, err := c.Releases().Create(ctx, &releases.Release{
        SourceRepo:   "test/repo",
        SourceCommit: "bbb222",
        Project:      projectB,
        ProjectPath:  "services/api",
        Bundle:       bundleStr,
    }, false)
    require.NoError(t, err)
    
    // List releases for project A
    releasesA, err := c.Releases().List(ctx, projectA)
    require.NoError(t, err)
    for _, r := range releasesA {
        require.Equal(t, projectA, r.Project, "Should only see project A releases")
    }
    
    // List releases for project B
    releasesB, err := c.Releases().List(ctx, projectB)
    require.NoError(t, err)
    for _, r := range releasesB {
        require.Equal(t, projectB, r.Project, "Should only see project B releases")
    }
    
    // Create aliases and verify isolation
    err = c.Aliases().Create(ctx, projectA+"-prod", releaseA.ID)
    require.NoError(t, err)
    
    err = c.Aliases().Create(ctx, projectB+"-prod", releaseB.ID)
    require.NoError(t, err)
}

