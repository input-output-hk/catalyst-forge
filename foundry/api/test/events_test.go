package test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/input-output-hk/catalyst-forge/lib/foundry/client/releases"
)

func TestEventsAPI(t *testing.T) {
	c := newTestClient()
	ctx, cancel := newTestContext()
	defer cancel()

	projectName := fmt.Sprintf("test-project-events-%d", time.Now().Unix())

	bundleStr := base64.StdEncoding.EncodeToString([]byte("sample code for events testing"))
	release := &releases.Release{
		SourceRepo:   "github.com/example/repo",
		SourceCommit: "abcdef123456",
		Project:      projectName,
		ProjectPath:  "services/api",
		Bundle:       bundleStr,
	}

	createdRelease, err := c.Releases().Create(ctx, release, false)
	require.NoError(t, err)
	require.NotEmpty(t, createdRelease.ID)

	// Create a deployment for testing events
	deployment, err := c.Deployments().Create(ctx, createdRelease.ID)
	require.NoError(t, err)
	require.NotEmpty(t, deployment.ID)

	t.Logf("Created deployment with ID: %s for testing events", deployment.ID)

	t.Run("AddEvent", func(t *testing.T) {
		eventName := "deployment_started"
		eventMessage := "Deployment process initiated"

		updatedDeployment, err := c.Events().Add(ctx, createdRelease.ID, deployment.ID, eventName, eventMessage)
		require.NoError(t, err)

		assert.NotEmpty(t, updatedDeployment.ID)
		assert.Equal(t, createdRelease.ID, updatedDeployment.ReleaseID)
		assert.NotZero(t, updatedDeployment.Timestamp)
		assert.NotZero(t, updatedDeployment.CreatedAt)
		assert.NotZero(t, updatedDeployment.UpdatedAt)

		// Verify the deployment was updated
		assert.Equal(t, deployment.ID, updatedDeployment.ID)
		assert.Equal(t, createdRelease.ID, updatedDeployment.ReleaseID)

		t.Run("GetEvents", func(t *testing.T) {
			events, err := c.Events().Get(ctx, createdRelease.ID, deployment.ID)
			require.NoError(t, err)

			// Should have at least one event (the one we just added)
			assert.NotEmpty(t, events)

			// Find our specific event
			found := false
			for _, event := range events {
				if event.Name == eventName && event.Message == eventMessage {
					found = true
					assert.NotZero(t, event.ID)
					assert.Equal(t, deployment.ID, event.DeploymentID)
					assert.Equal(t, eventName, event.Name)
					assert.Equal(t, eventMessage, event.Message)
					assert.NotZero(t, event.Timestamp)
					assert.NotZero(t, event.CreatedAt)
					assert.NotZero(t, event.UpdatedAt)
				}
			}

			assert.True(t, found, "Added event not found in events list")
		})
	})

	t.Run("MultipleEvents", func(t *testing.T) {
		// Add multiple events to test event accumulation
		events := []struct {
			name    string
			message string
		}{
			{"deployment_running", "Deployment is now running"},
			{"deployment_progress", "Deployment at 50% completion"},
			{"deployment_complete", "Deployment completed successfully"},
		}

		for _, event := range events {
			_, err := c.Events().Add(ctx, createdRelease.ID, deployment.ID, event.name, event.message)
			require.NoError(t, err)
		}

		t.Run("GetAllEvents", func(t *testing.T) {
			allEvents, err := c.Events().Get(ctx, createdRelease.ID, deployment.ID)
			require.NoError(t, err)

			// Should have at least the number of events we added (plus any from previous tests)
			assert.GreaterOrEqual(t, len(allEvents), len(events))

			// Verify our specific events are present
			foundEvents := make(map[string]bool)
			for _, expectedEvent := range events {
				foundEvents[expectedEvent.name] = false
			}

			for _, event := range allEvents {
				for _, expectedEvent := range events {
					if event.Name == expectedEvent.name && event.Message == expectedEvent.message {
						foundEvents[expectedEvent.name] = true
						assert.Equal(t, deployment.ID, event.DeploymentID)
						assert.NotZero(t, event.ID)
						assert.NotZero(t, event.Timestamp)
					}
				}
			}

			// Verify all expected events were found
			for eventName, found := range foundEvents {
				assert.True(t, found, "Event '%s' not found in events list", eventName)
			}
		})
	})

	t.Run("EventWithSpecialCharacters", func(t *testing.T) {
		// Test events with special characters and longer messages
		eventName := "deployment_error"
		eventMessage := "Deployment failed with error: Connection timeout after 30 seconds. Retrying..."

		_, err := c.Events().Add(ctx, createdRelease.ID, deployment.ID, eventName, eventMessage)
		require.NoError(t, err)

		events, err := c.Events().Get(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err)

		found := false
		for _, event := range events {
			if event.Name == eventName && event.Message == eventMessage {
				found = true
				assert.Equal(t, eventName, event.Name)
				assert.Equal(t, eventMessage, event.Message)
			}
		}

		assert.True(t, found, "Event with special characters not found")
	})

	t.Run("EventTimestamps", func(t *testing.T) {
		// Test that events have proper timestamps
		eventName := "timestamp_test"
		eventMessage := "Testing event timestamps"

		beforeAdd := time.Now()
		_, err := c.Events().Add(ctx, createdRelease.ID, deployment.ID, eventName, eventMessage)
		require.NoError(t, err)
		afterAdd := time.Now()

		events, err := c.Events().Get(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err)

		found := false
		for _, event := range events {
			if event.Name == eventName && event.Message == eventMessage {
				found = true
				// Verify timestamp is within reasonable bounds
				assert.True(t, event.Timestamp.After(beforeAdd) || event.Timestamp.Equal(beforeAdd))
				assert.True(t, event.Timestamp.Before(afterAdd) || event.Timestamp.Equal(afterAdd))
				assert.True(t, event.CreatedAt.After(beforeAdd) || event.CreatedAt.Equal(beforeAdd))
				assert.True(t, event.CreatedAt.Before(afterAdd) || event.CreatedAt.Equal(afterAdd))
			}
		}

		assert.True(t, found, "Timestamp test event not found")
	})

	t.Run("EventsForDifferentDeployments", func(t *testing.T) {
		// Create a second deployment to test event isolation
		deployment2, err := c.Deployments().Create(ctx, createdRelease.ID)
		require.NoError(t, err)
		require.NotEmpty(t, deployment2.ID)

		// Add event to second deployment
		eventName := "second_deployment_event"
		eventMessage := "Event for second deployment"

		_, err = c.Events().Add(ctx, createdRelease.ID, deployment2.ID, eventName, eventMessage)
		require.NoError(t, err)

		// Verify events are isolated between deployments
		events1, err := c.Events().Get(ctx, createdRelease.ID, deployment.ID)
		require.NoError(t, err)

		events2, err := c.Events().Get(ctx, createdRelease.ID, deployment2.ID)
		require.NoError(t, err)

		// Check that the second deployment's event is not in the first deployment's events
		foundInDeployment1 := false
		for _, event := range events1 {
			if event.Name == eventName && event.Message == eventMessage {
				foundInDeployment1 = true
			}
		}
		assert.False(t, foundInDeployment1, "Event from deployment 2 should not appear in deployment 1")

		// Check that the second deployment's event is in its own events
		foundInDeployment2 := false
		for _, event := range events2 {
			if event.Name == eventName && event.Message == eventMessage {
				foundInDeployment2 = true
				assert.Equal(t, deployment2.ID, event.DeploymentID)
			}
		}
		assert.True(t, foundInDeployment2, "Event should be found in its own deployment")

		// Clean up second deployment
		// Note: We don't delete the deployment as it might be used by other tests
		// The test environment should handle cleanup
	})

	t.Run("EmptyEventsList", func(t *testing.T) {
		// Create a new deployment with no events
		deployment3, err := c.Deployments().Create(ctx, createdRelease.ID)
		require.NoError(t, err)
		require.NotEmpty(t, deployment3.ID)

		// Get events for the new deployment (should be empty or minimal)
		events, err := c.Events().Get(ctx, createdRelease.ID, deployment3.ID)
		require.NoError(t, err)

		// The list should be empty or contain only system-generated events
		// This depends on the API implementation
		assert.NotNil(t, events, "Events list should not be nil")
	})

	t.Run("InvalidDeploymentID", func(t *testing.T) {
		// Test with invalid deployment ID
		invalidDeploymentID := "invalid-deployment-id"
		eventName := "test_invalid"
		eventMessage := "This should fail"

		_, err := c.Events().Add(ctx, createdRelease.ID, invalidDeploymentID, eventName, eventMessage)
		assert.Error(t, err, "Expected error when adding event to invalid deployment")

		_, err = c.Events().Get(ctx, createdRelease.ID, invalidDeploymentID)
		assert.Error(t, err, "Expected error when getting events for invalid deployment")
	})

	t.Run("InvalidReleaseID", func(t *testing.T) {
		// Test with invalid release ID
		invalidReleaseID := "invalid-release-id"
		eventName := "test_invalid_release"
		eventMessage := "This should fail"

		_, err := c.Events().Add(ctx, invalidReleaseID, deployment.ID, eventName, eventMessage)
		assert.Error(t, err, "Expected error when adding event to invalid release")

		_, err = c.Events().Get(ctx, invalidReleaseID, deployment.ID)
		assert.Error(t, err, "Expected error when getting events for invalid release")
	})
}
