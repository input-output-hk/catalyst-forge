package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client"
)

func TestDeploymentAPI(t *testing.T) {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	c := client.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectName := fmt.Sprintf("test-project-deploy-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("sample code for deployment testing"))
	release := &client.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.CreateRelease(ctx, release, false)
	require.NoError(t, err)

	t.Run("CreateDeployment", func(t *testing.T) {
		deployment, err := c.CreateDeployment(ctx, createdRelease.ID)
		require.NoError(t, err)

		assert.NotEmpty(t, deployment.ID)
		assert.Equal(t, createdRelease.ID, deployment.ReleaseID)
		assert.Equal(t, client.DeploymentStatusPending, deployment.Status)
		assert.NotZero(t, deployment.Timestamp)

		deploymentID := deployment.ID
		t.Logf("Created deployment with ID: %s", deploymentID)

		t.Run("GetDeployment", func(t *testing.T) {
			fetchedDeployment, err := c.GetDeployment(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)

			assert.Equal(t, deploymentID, fetchedDeployment.ID)
			assert.Equal(t, createdRelease.ID, fetchedDeployment.ReleaseID)
			assert.Equal(t, client.DeploymentStatusPending, fetchedDeployment.Status)
		})

		t.Run("UpdateDeploymentStatus", func(t *testing.T) {
			err := c.UpdateDeploymentStatus(ctx, createdRelease.ID, deploymentID, client.DeploymentStatusRunning, "Deployment in progress")
			require.NoError(t, err)

			deployment, err := c.GetDeployment(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)
			assert.Equal(t, client.DeploymentStatusRunning, deployment.Status)
			assert.Equal(t, "Deployment in progress", deployment.Reason)

			err = c.UpdateDeploymentStatus(ctx, createdRelease.ID, deploymentID, client.DeploymentStatusSucceeded, "Deployment completed successfully")
			require.NoError(t, err)

			deployment, err = c.GetDeployment(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)
			assert.Equal(t, client.DeploymentStatusSucceeded, deployment.Status)
			assert.Equal(t, "Deployment completed successfully", deployment.Reason)
		})

		t.Run("CreateSecondDeployment", func(t *testing.T) {
			deployment2, err := c.CreateDeployment(ctx, createdRelease.ID)
			require.NoError(t, err)

			err = c.UpdateDeploymentStatus(ctx, createdRelease.ID, deployment2.ID, client.DeploymentStatusFailed, "Deployment failed due to resource constraints")
			require.NoError(t, err)

			t.Run("ListDeployments", func(t *testing.T) {
				deployments, err := c.ListDeployments(ctx, createdRelease.ID)
				require.NoError(t, err)

				assert.Equal(t, 2, len(deployments))

				found1, found2 := false, false
				for _, d := range deployments {
					if d.ID == deploymentID {
						found1 = true
						assert.Equal(t, client.DeploymentStatusSucceeded, d.Status)
					}
					if d.ID == deployment2.ID {
						found2 = true
						assert.Equal(t, client.DeploymentStatusFailed, d.Status)
					}
				}

				assert.True(t, found1, "First deployment not found in list")
				assert.True(t, found2, "Second deployment not found in list")
			})

			t.Run("GetLatestDeployment", func(t *testing.T) {
				latest, err := c.GetLatestDeployment(ctx, createdRelease.ID)
				require.NoError(t, err)

				assert.Equal(t, deployment2.ID, latest.ID)
				assert.Equal(t, client.DeploymentStatusFailed, latest.Status)
			})
		})
	})
}

func TestCreateReleaseWithDeployment(t *testing.T) {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	c := client.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectName := fmt.Sprintf("test-project-auto-deploy-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("sample code for auto-deployment testing"))
	release := &client.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "deploy-commit",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.CreateRelease(ctx, release, true)
	require.NoError(t, err)
	assert.NotEmpty(t, createdRelease.ID)

	assert.GreaterOrEqual(t, len(createdRelease.Deployments), 1, "Expected at least one deployment to be attached to the release")
	assert.Equal(t, createdRelease.ID, createdRelease.Deployments[0].ReleaseID)
	assert.Equal(t, client.DeploymentStatusPending, createdRelease.Deployments[0].Status)

	latest, err := c.GetLatestDeployment(ctx, createdRelease.ID)
	require.NoError(t, err)
	assert.Equal(t, createdRelease.ID, latest.ReleaseID)
	assert.Equal(t, createdRelease.Deployments[0].ID, latest.ID)
}
