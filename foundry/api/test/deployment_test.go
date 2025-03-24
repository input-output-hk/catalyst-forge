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
		assert.Equal(t, 0, deployment.Attempts)

		deploymentID := deployment.ID
		t.Logf("Created deployment with ID: %s", deploymentID)

		t.Run("GetDeployment", func(t *testing.T) {
			fetchedDeployment, err := c.GetDeployment(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)

			assert.Equal(t, deploymentID, fetchedDeployment.ID)
			assert.Equal(t, createdRelease.ID, fetchedDeployment.ReleaseID)
			assert.Equal(t, client.DeploymentStatusPending, fetchedDeployment.Status)
			assert.Equal(t, 0, fetchedDeployment.Attempts)
		})

		t.Run("UpdateDeployment", func(t *testing.T) {
			currentDeployment, err := c.GetDeployment(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)

			currentDeployment.Status = client.DeploymentStatusRunning
			currentDeployment.Reason = "Deployment in progress"

			updatedDeployment, err := c.UpdateDeployment(ctx, createdRelease.ID, currentDeployment)
			require.NoError(t, err)
			assert.Equal(t, client.DeploymentStatusRunning, updatedDeployment.Status)
			assert.Equal(t, "Deployment in progress", updatedDeployment.Reason)

			fetchedDeployment, err := c.GetDeployment(ctx, createdRelease.ID, deploymentID)
			require.NoError(t, err)
			assert.Equal(t, client.DeploymentStatusRunning, fetchedDeployment.Status)
			assert.Equal(t, "Deployment in progress", fetchedDeployment.Reason)

			fetchedDeployment.Status = client.DeploymentStatusSucceeded
			fetchedDeployment.Reason = "Deployment completed successfully"
			fetchedDeployment.Attempts = 1

			updatedDeployment, err = c.UpdateDeployment(ctx, createdRelease.ID, fetchedDeployment)
			require.NoError(t, err)
			assert.Equal(t, client.DeploymentStatusSucceeded, updatedDeployment.Status)
			assert.Equal(t, "Deployment completed successfully", updatedDeployment.Reason)
			assert.Equal(t, 1, updatedDeployment.Attempts)
		})

		t.Run("CreateSecondDeployment", func(t *testing.T) {
			deployment2, err := c.CreateDeployment(ctx, createdRelease.ID)
			require.NoError(t, err)

			deployment2.Status = client.DeploymentStatusFailed
			deployment2.Reason = "Deployment failed due to resource constraints"
			deployment2.Attempts = 2

			updatedDeployment2, err := c.UpdateDeployment(ctx, createdRelease.ID, deployment2)
			require.NoError(t, err)
			assert.Equal(t, client.DeploymentStatusFailed, updatedDeployment2.Status)
			assert.Equal(t, 2, updatedDeployment2.Attempts)

			t.Run("ListDeployments", func(t *testing.T) {
				deployments, err := c.ListDeployments(ctx, createdRelease.ID)
				require.NoError(t, err)

				assert.Equal(t, 2, len(deployments))

				found1, found2 := false, false
				for _, d := range deployments {
					if d.ID == deploymentID {
						found1 = true
						assert.Equal(t, client.DeploymentStatusSucceeded, d.Status)
						assert.Equal(t, 1, d.Attempts)
					}
					if d.ID == deployment2.ID {
						found2 = true
						assert.Equal(t, client.DeploymentStatusFailed, d.Status)
						assert.Equal(t, 2, d.Attempts)
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
				assert.Equal(t, 2, latest.Attempts)
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
	assert.Equal(t, 0, createdRelease.Deployments[0].Attempts)

	latest, err := c.GetLatestDeployment(ctx, createdRelease.ID)
	require.NoError(t, err)
	assert.Equal(t, createdRelease.ID, latest.ReleaseID)
	assert.Equal(t, createdRelease.Deployments[0].ID, latest.ID)
	assert.Equal(t, 0, latest.Attempts)
}

func TestIncrementDeploymentAttemptsOnly(t *testing.T) {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	c := client.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectName := fmt.Sprintf("test-increment-only-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("increment attempts test"))
	release := &client.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.CreateRelease(ctx, release, false)
	require.NoError(t, err)
	require.NotEmpty(t, createdRelease.ID)

	deployment, err := c.CreateDeployment(ctx, createdRelease.ID)
	require.NoError(t, err)
	require.NotEmpty(t, deployment.ID)

	assert.Equal(t, 0, deployment.Attempts, "New deployment should start with 0 attempts")
	assert.Equal(t, client.DeploymentStatusPending, deployment.Status, "New deployment should have pending status")

	t.Run("SingleIncrement", func(t *testing.T) {
		incrementedDeployment, err := c.IncrementDeploymentAttempts(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err, "Increment operation should succeed")
		assert.Equal(t, 1, incrementedDeployment.Attempts, "Attempts should be incremented to 1")

		fetchedDeployment, err := c.GetDeployment(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err, "Should be able to fetch the deployment")
		assert.Equal(t, 1, fetchedDeployment.Attempts, "Fetched deployment should show incremented attempts")
		assert.Equal(t, client.DeploymentStatusPending, fetchedDeployment.Status, "Status should remain unchanged")
	})

	t.Run("MultipleIncrements", func(t *testing.T) {
		incrementedDeployment, err := c.IncrementDeploymentAttempts(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err, "Second increment operation should succeed")
		assert.Equal(t, 2, incrementedDeployment.Attempts, "Attempts should be incremented to 2")

		incrementedDeployment, err = c.IncrementDeploymentAttempts(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err, "Third increment operation should succeed")
		assert.Equal(t, 3, incrementedDeployment.Attempts, "Attempts should be incremented to 3")
	})
}

func TestDeploymentEvents(t *testing.T) {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	c := client.NewClient(apiURL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectName := fmt.Sprintf("test-project-events-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("sample code for events testing"))
	release := &client.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.CreateRelease(ctx, release, false)
	require.NoError(t, err)

	deployment, err := c.CreateDeployment(ctx, createdRelease.ID)
	require.NoError(t, err)

	eventData := []struct {
		name    string
		message string
	}{
		{"DeploymentStarted", "Beginning deployment process"},
		{"ConfigValidation", "Validating deployment configuration"},
		{"ResourceCreation", "Creating Kubernetes resources"},
		{"DeploymentReady", "Deployment resources are ready"},
	}

	t.Run("AddSingleEvent", func(t *testing.T) {
		event := eventData[0]
		updatedDeployment, err := c.AddDeploymentEvent(ctx, createdRelease.ID, deployment.ID, event.name, event.message)
		require.NoError(t, err)

		require.NotEmpty(t, updatedDeployment.Events, "Deployment should have events")
		require.Len(t, updatedDeployment.Events, 1, "Deployment should have exactly one event")
		assert.Equal(t, event.name, updatedDeployment.Events[0].Name)
		assert.Equal(t, event.message, updatedDeployment.Events[0].Message)
		assert.NotZero(t, updatedDeployment.Events[0].Timestamp, "Event timestamp should be set")

		fetchedDeployment, err := c.GetDeployment(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err)
		require.NotEmpty(t, fetchedDeployment.Events, "Fetched deployment should have events")
		require.Len(t, fetchedDeployment.Events, 1, "Fetched deployment should have exactly one event")
		assert.Equal(t, event.name, fetchedDeployment.Events[0].Name)
	})

	t.Run("AddMultipleEvents", func(t *testing.T) {
		for i := 1; i < len(eventData); i++ {
			event := eventData[i]
			_, err := c.AddDeploymentEvent(ctx, createdRelease.ID, deployment.ID, event.name, event.message)
			require.NoError(t, err)
		}

		fetchedDeployment, err := c.GetDeployment(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err)

		require.Len(t, fetchedDeployment.Events, len(eventData),
			"Deployment should have all events")

		// Events should be in reverse chronological order (newest first)
		for i, event := range fetchedDeployment.Events {
			expectedEvent := eventData[len(eventData)-1-i]
			assert.Equal(t, expectedEvent.name, event.Name)
			assert.Equal(t, expectedEvent.message, event.Message)
		}
	})

	t.Run("GetDeploymentEvents", func(t *testing.T) {
		events, err := c.GetDeploymentEvents(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err)

		require.Len(t, events, len(eventData),
			"Should return all events")

		for i, event := range events {
			expectedEvent := eventData[len(eventData)-1-i]
			assert.Equal(t, expectedEvent.name, event.Name)
			assert.Equal(t, expectedEvent.message, event.Message)
		}
	})
}
