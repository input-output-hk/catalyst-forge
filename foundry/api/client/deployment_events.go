package client

import (
	"context"
	"fmt"
	"net/http"
)

// AddDeploymentEvent adds an event to a deployment
func (c *HTTPClient) AddDeploymentEvent(ctx context.Context, releaseID string, deployID string, name string, message string) (*ReleaseDeployment, error) {
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

// GetDeploymentEvents retrieves all events for a deployment
func (c *HTTPClient) GetDeploymentEvents(ctx context.Context, releaseID string, deployID string) ([]DeploymentEvent, error) {
	path := fmt.Sprintf("/release/%s/deploy/%s/events", releaseID, deployID)

	var resp []DeploymentEvent
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
