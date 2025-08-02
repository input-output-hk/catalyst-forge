package deployments

import (
	"context"
	"fmt"
	"net/http"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/events.go . EventsClientInterface

// EventsClientInterface defines the interface for deployment event operations
type EventsClientInterface interface {
	Add(ctx context.Context, releaseID string, deployID string, name string, message string) (*ReleaseDeployment, error)
	Get(ctx context.Context, releaseID string, deployID string) ([]DeploymentEvent, error)
}

// EventsClient handles deployment event-related operations
type EventsClient struct {
	do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error
}

// Ensure EventsClient implements EventsClientInterface
var _ EventsClientInterface = (*EventsClient)(nil)

// NewEventsClient creates a new events client
func NewEventsClient(do func(ctx context.Context, method, path string, reqBody, respBody interface{}) error) *EventsClient {
	return &EventsClient{do: do}
}

// Add adds an event to a deployment
func (c *EventsClient) Add(ctx context.Context, releaseID string, deployID string, name string, message string) (*ReleaseDeployment, error) {
	path := fmt.Sprintf("/release/%s/deploy/%s/events", releaseID, deployID)

	payload := struct {
		Name    string `json:"name"`
		Message string `json:"message"`
	}{
		Name:    name,
		Message: message,
	}

	var resp ReleaseDeployment
	err := c.do(ctx, http.MethodPost, path, payload, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// Get retrieves all events for a deployment
func (c *EventsClient) Get(ctx context.Context, releaseID string, deployID string) ([]DeploymentEvent, error) {
	path := fmt.Sprintf("/release/%s/deploy/%s/events", releaseID, deployID)

	var resp []DeploymentEvent
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
