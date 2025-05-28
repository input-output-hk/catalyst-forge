package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/client.go . Client

// Client interface defines the operations that can be performed against the API
type Client interface {
	// Release operations
	CreateRelease(ctx context.Context, release *Release, deploy bool) (*Release, error)
	GetRelease(ctx context.Context, id string) (*Release, error)
	UpdateRelease(ctx context.Context, release *Release) (*Release, error)
	ListReleases(ctx context.Context, projectName string) ([]Release, error)
	GetReleaseByAlias(ctx context.Context, aliasName string) (*Release, error)

	// Release alias operations
	CreateAlias(ctx context.Context, aliasName string, releaseID string) error
	DeleteAlias(ctx context.Context, aliasName string) error
	ListAliases(ctx context.Context, releaseID string) ([]ReleaseAlias, error)

	// Deployment operations
	CreateDeployment(ctx context.Context, releaseID string) (*ReleaseDeployment, error)
	GetDeployment(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error)
	UpdateDeployment(ctx context.Context, releaseID string, deployment *ReleaseDeployment) (*ReleaseDeployment, error)
	IncrementDeploymentAttempts(ctx context.Context, releaseID string, deployID string) (*ReleaseDeployment, error)
	ListDeployments(ctx context.Context, releaseID string) ([]ReleaseDeployment, error)
	GetLatestDeployment(ctx context.Context, releaseID string) (*ReleaseDeployment, error)

	// Deployment event operations
	AddDeploymentEvent(ctx context.Context, releaseID string, deployID string, name string, message string) (*ReleaseDeployment, error)
	GetDeploymentEvents(ctx context.Context, releaseID string, deployID string) ([]DeploymentEvent, error)
}

// HTTPClient is an implementation of the Client interface that uses HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// ClientOption is a function type for client configuration
type ClientOption func(*HTTPClient)

// WithTimeout sets the timeout for the HTTP client
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Timeout = timeout
	}
}

// WithTransport sets a custom transport for the HTTP client
func WithTransport(transport http.RoundTripper) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Transport = transport
	}
}

// NewClient creates a new API client
func NewClient(baseURL string, options ...ClientOption) Client {
	client := &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// do performs an HTTP request and processes the response
func (c *HTTPClient) do(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var reqBodyReader io.Reader
	if reqBody != nil {
		reqBodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("error marshaling request body: %w", err)
		}
		reqBodyReader = bytes.NewReader(reqBodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBodyReader)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error performing request: %w", err)
	}
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(respBodyBytes, &errResp); err != nil {
			return fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(respBodyBytes))
		}
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, errResp.Error)
	}

	if respBody != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(respBodyBytes, respBody); err != nil {
			return fmt.Errorf("error unmarshaling response: %w", err)
		}
	}

	return nil
}
