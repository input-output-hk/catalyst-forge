package test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/deployments"
	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
)

func TestDeploymentAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := generateTestName("test-project-deploy")

	createdRelease, err := createTestRelease(c, ctx, projectName)
	require.NoError(t, err)

	t.Run("CreateDeployment", func(t *testing.T) {
		deployment, err := c.Deployments().Create(ctx, createdRelease.ID)
		require.NoError(t, err)

		assert.NotEmpty(t, deployment.ID)
		assert.Equal(t, createdRelease.ID, deployment.ReleaseID)
		assert.Equal(t, deployments.DeploymentStatusPending, deployment.Status)
		assert.NotZero(t, deployment.Timestamp)
		assert.Equal(t, 0, deployment.Attempts)

		deploymentID := deployment.ID
		t.Logf("Created deployment with ID: %s", deploymentID)

		t.Run("GetDeployment", func(t *testing.T) {
			fetchedDeployment, err := c.Deployments().Get(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)

			assert.Equal(t, deploymentID, fetchedDeployment.ID)
			assert.Equal(t, createdRelease.ID, fetchedDeployment.ReleaseID)
			assert.Equal(t, deployments.DeploymentStatusPending, fetchedDeployment.Status)
			assert.Equal(t, 0, fetchedDeployment.Attempts)
		})

		t.Run("UpdateDeployment", func(t *testing.T) {
			currentDeployment, err := c.Deployments().Get(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)

			currentDeployment.Status = deployments.DeploymentStatusRunning
			currentDeployment.Reason = "Deployment in progress"

			updatedDeployment, err := c.Deployments().Update(ctx, createdRelease.ID, currentDeployment)
			require.NoError(t, err)
			assert.Equal(t, deployments.DeploymentStatusRunning, updatedDeployment.Status)
			assert.Equal(t, "Deployment in progress", updatedDeployment.Reason)

			fetchedDeployment, err := c.Deployments().Get(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)
			assert.Equal(t, deployments.DeploymentStatusRunning, fetchedDeployment.Status)
			assert.Equal(t, "Deployment in progress", fetchedDeployment.Reason)

			fetchedDeployment.Status = deployments.DeploymentStatusSucceeded
			fetchedDeployment.Reason = "Deployment completed successfully"
			fetchedDeployment.Attempts = 1

			updatedDeployment, err = c.Deployments().Update(ctx, createdRelease.ID, fetchedDeployment)
			require.NoError(t, err)
			assert.Equal(t, deployments.DeploymentStatusSucceeded, updatedDeployment.Status)
			assert.Equal(t, "Deployment completed successfully", updatedDeployment.Reason)
			assert.Equal(t, 1, updatedDeployment.Attempts)
		})

		t.Run("CreateSecondDeployment", func(t *testing.T) {
			deployment2, err := c.Deployments().Create(ctx, createdRelease.ID)
			require.NoError(t, err)

			deployment2.Status = deployments.DeploymentStatusFailed
			deployment2.Reason = "Deployment failed due to resource constraints"
			deployment2.Attempts = 2

			updatedDeployment2, err := c.Deployments().Update(ctx, createdRelease.ID, deployment2)
			require.NoError(t, err)
			assert.Equal(t, deployments.DeploymentStatusFailed, updatedDeployment2.Status)
			assert.Equal(t, 2, updatedDeployment2.Attempts)

			t.Run("ListDeployments", func(t *testing.T) {
				deploymentList, err := c.Deployments().List(ctx, createdRelease.ID)
				require.NoError(t, err)

				assert.GreaterOrEqual(t, len(deploymentList), 2)

				found1, found2 := false, false
				for _, d := range deploymentList {
					if d.ID == deploymentID {
						found1 = true
					}
					if d.ID == deployment2.ID {
						found2 = true
					}
				}

				assert.True(t, found1, "First deployment not found in list")
				assert.True(t, found2, "Second deployment not found in list")
			})
		})
	})
}

func TestCreateReleaseWithDeployment(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := generateTestName("test-project-deploy-with-release")

	bundleStr := base64.StdEncoding.EncodeToString([]byte("sample code for deployment testing"))
	release := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.Releases().Create(ctx, release, true)
	require.NoError(t, err)

	assert.NotEmpty(t, createdRelease.ID)
	assert.Equal(t, projectName, createdRelease.Project)

	// Verify that a deployment was created automatically
	deploymentList, err := c.Deployments().List(ctx, createdRelease.ID)
	require.NoError(t, err)
	assert.Len(t, deploymentList, 1)

	deployment := deploymentList[0]
	assert.NotEmpty(t, deployment.ID)
	assert.Equal(t, createdRelease.ID, deployment.ReleaseID)
	assert.Equal(t, deployments.DeploymentStatusPending, deployment.Status)
	assert.NotZero(t, deployment.Timestamp)
	assert.Equal(t, 0, deployment.Attempts)
}

func TestIncrementDeploymentAttemptsOnly(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := generateTestName("test-project-increment")

	createdRelease, err := createTestRelease(c, ctx, projectName)
	require.NoError(t, err)

	deployment, err := c.Deployments().Create(ctx, createdRelease.ID)
	require.NoError(t, err)

	// Verify initial attempts count
	assert.Equal(t, 0, deployment.Attempts)

	// Increment attempts using Update method (since IncrementAttempts doesn't exist)
	deployment.Attempts = 1
	updatedDeployment, err := c.Deployments().Update(ctx, createdRelease.ID, deployment)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedDeployment.Attempts)

	// Verify the increment persisted
	fetchedDeployment, err := c.Deployments().Get(ctx, createdRelease.ID, deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, fetchedDeployment.Attempts)

	// Increment again
	fetchedDeployment.Attempts = 2
	updatedDeployment, err = c.Deployments().Update(ctx, createdRelease.ID, fetchedDeployment)
	require.NoError(t, err)
	assert.Equal(t, 2, updatedDeployment.Attempts)

	// Verify the second increment persisted
	fetchedDeployment, err = c.Deployments().Get(ctx, createdRelease.ID, deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, fetchedDeployment.Attempts)
}

func TestDeploymentEvents(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := fmt.Sprintf("test-project-events-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("sample code for deployment testing"))
	release := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.Releases().Create(ctx, release, false)
	require.NoError(t, err)

	deployment, err := c.Deployments().Create(ctx, createdRelease.ID)
	require.NoError(t, err)

	t.Run("AddEvent", func(t *testing.T) {
		eventName := "deployment_started"
		eventMessage := "Deployment process initiated"

		updatedDeployment, err := c.Events().Add(ctx, createdRelease.ID, deployment.ID, eventName, eventMessage)
		require.NoError(t, err)

		assert.NotEmpty(t, updatedDeployment.ID)
		assert.Equal(t, createdRelease.ID, updatedDeployment.ReleaseID)

		t.Run("GetEvents", func(t *testing.T) {
			events, err := c.Events().Get(ctx, createdRelease.ID, deployment.ID)
			require.NoError(t, err)

			assert.Len(t, events, 1)
			event := events[0]

			assert.NotZero(t, event.ID)
			assert.Equal(t, deployment.ID, event.DeploymentID)
			assert.Equal(t, eventName, event.Name)
			assert.Equal(t, eventMessage, event.Message)
			assert.NotZero(t, event.Timestamp)
		})

		t.Run("AddMultipleEvents", func(t *testing.T) {
			// Add a second event
			_, err := c.Events().Add(ctx, createdRelease.ID, deployment.ID, "deployment_progress", "Deployment is 50% complete")
			require.NoError(t, err)

			// Add a third event
			_, err = c.Events().Add(ctx, createdRelease.ID, deployment.ID, "deployment_completed", "Deployment finished successfully")
			require.NoError(t, err)

			events, err := c.Events().Get(ctx, createdRelease.ID, deployment.ID)
			require.NoError(t, err)

			assert.Len(t, events, 3)

			// Verify all events have the correct deployment ID
			for _, event := range events {
				assert.Equal(t, deployment.ID, event.DeploymentID)
				assert.NotZero(t, event.ID)
				assert.NotZero(t, event.Timestamp)
			}
		})
	})
}
