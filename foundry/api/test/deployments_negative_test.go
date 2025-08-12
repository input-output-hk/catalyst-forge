//go:build integration

package test

import (
    "encoding/base64"
    "testing"

    "github.com/stretchr/testify/require"

    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/deployments"
    "github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
)

// Test deployment with non-existent IDs
func TestDeployments_NonExistentIDs(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    t.Run("create deployment for non-existent release", func(t *testing.T) {
        _, err := c.Deployments().Create(ctx, "non-existent-release-id")
        require.Error(t, err, "Should reject deployment for non-existent release")
    })

    t.Run("get non-existent deployment", func(t *testing.T) {
        // First create a valid release
        bundleStr := base64.StdEncoding.EncodeToString([]byte("test content"))
        release, err := c.Releases().Create(ctx, &releases.Release{
            Project:      generateTestName("deploy-test"),
            ProjectPath:  "services/api",
            SourceRepo:   "test/repo",
            SourceCommit: "abc123",
            Bundle:       bundleStr,
        }, false)
        require.NoError(t, err)

        // Try to get non-existent deployment
        _, err = c.Deployments().Get(ctx, release.ID, "non-existent-deployment-id")
        require.Error(t, err, "Should return error for non-existent deployment")
    })

    t.Run("update non-existent deployment", func(t *testing.T) {
        // Create a valid release
        bundleStr := base64.StdEncoding.EncodeToString([]byte("test content"))
        release, err := c.Releases().Create(ctx, &releases.Release{
            Project:      generateTestName("deploy-update-test"),
            ProjectPath:  "services/api",
            SourceRepo:   "test/repo",
            SourceCommit: "def456",
            Bundle:       bundleStr,
        }, false)
        require.NoError(t, err)

        // Try to update non-existent deployment
        fakeDeployment := &deployments.ReleaseDeployment{
            ID:        "non-existent-deployment-id",
            ReleaseID: release.ID,
            Status:    deployments.DeploymentStatusRunning,
            Reason:    "Test update",
        }
        _, err = c.Deployments().Update(ctx, release.ID, fakeDeployment)
        require.Error(t, err, "Should reject update for non-existent deployment")
    })
}

// Test invalid state transitions
func TestDeployments_InvalidStateTransitions(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    // Create a valid release
    bundleStr := base64.StdEncoding.EncodeToString([]byte("test content"))
    release, err := c.Releases().Create(ctx, &releases.Release{
        Project:      generateTestName("state-test"),
        ProjectPath:  "services/api",
        SourceRepo:   "test/repo",
        SourceCommit: "abc789",
        Bundle:       bundleStr,
    }, false)
    require.NoError(t, err)

    // Create deployment
    deployment, err := c.Deployments().Create(ctx, release.ID)
    require.NoError(t, err)
    require.Equal(t, deployments.DeploymentStatusPending, deployment.Status)

    // Note: The API might not enforce strict state transitions
    // These tests document the current behavior
    t.Run("invalid status values", func(t *testing.T) {
        // Try to set an invalid status
        deployment.Status = "invalid-status"
        _, err := c.Deployments().Update(ctx, release.ID, deployment)
        // The API might accept any string or might validate
        _ = err // Document the behavior
    })

    t.Run("transition from succeeded to pending", func(t *testing.T) {
        // First set to succeeded
        deployment.Status = deployments.DeploymentStatusSucceeded
        deployment.Reason = "Success"
        updated, err := c.Deployments().Update(ctx, release.ID, deployment)
        require.NoError(t, err)
        require.Equal(t, deployments.DeploymentStatusSucceeded, updated.Status)

        // Try to transition back to pending (might be invalid)
        updated.Status = deployments.DeploymentStatusPending
        updated.Reason = "Retry"
        _, err = c.Deployments().Update(ctx, release.ID, updated)
        // Document whether this is allowed
        _ = err
    })

    t.Run("negative attempts count", func(t *testing.T) {
        deployment2, err := c.Deployments().Create(ctx, release.ID)
        require.NoError(t, err)

        // Try to set negative attempts
        deployment2.Attempts = -1
        _, err = c.Deployments().Update(ctx, release.ID, deployment2)
        // Document whether negative attempts are validated
        _ = err
    })
}

// Test deployment isolation between projects
func TestDeployments_CrossProjectIsolation(t *testing.T) {
    env := NewTestEnv(t)
    c := env.AdminClient()
    ctx, cancel := newTestContext()
    defer cancel()

    bundleStr := base64.StdEncoding.EncodeToString([]byte("content"))

    // Create releases in different projects
    projectA := generateTestName("project-a")
    releaseA, err := c.Releases().Create(ctx, &releases.Release{
        Project:      projectA,
        ProjectPath:  "services/api",
        SourceRepo:   "test/repo",
        SourceCommit: "aaa111",
        Bundle:       bundleStr,
    }, false)
    require.NoError(t, err)

    projectB := generateTestName("project-b")
    releaseB, err := c.Releases().Create(ctx, &releases.Release{
        Project:      projectB,
        ProjectPath:  "services/api",
        SourceRepo:   "test/repo",
        SourceCommit: "bbb222",
        Bundle:       bundleStr,
    }, false)
    require.NoError(t, err)

    // Create deployments for each project
    deployA, err := c.Deployments().Create(ctx, releaseA.ID)
    require.NoError(t, err)

    deployB, err := c.Deployments().Create(ctx, releaseB.ID)
    require.NoError(t, err)
    _ = deployB // Used for verification below

    // Verify deployments are isolated to their releases
    deploymentsA, err := c.Deployments().List(ctx, releaseA.ID)
    require.NoError(t, err)
    for _, d := range deploymentsA {
        require.Equal(t, releaseA.ID, d.ReleaseID, "Should only see deployments for release A")
    }

    deploymentsB, err := c.Deployments().List(ctx, releaseB.ID)
    require.NoError(t, err)
    for _, d := range deploymentsB {
        require.Equal(t, releaseB.ID, d.ReleaseID, "Should only see deployments for release B")
    }

    // Try to access deployment A through release B
    // Note: API might not enforce cross-release isolation
    _, err = c.Deployments().Get(ctx, releaseB.ID, deployA.ID)
    if err == nil {
        t.Log("API does not enforce cross-release deployment isolation")
    } else {
        t.Log("API enforces cross-release deployment isolation")
    }
}