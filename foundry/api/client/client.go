package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/input-output-hk/catalyst-forge/foundry/api/client/auth"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/deployments"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/gha"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/releases"
	"github.com/input-output-hk/catalyst-forge/foundry/api/client/users"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/client.go . Client

// APIError represents an error returned by the API server
type APIError struct {
	StatusCode    int    `json:"status_code"`
	StatusText    string `json:"status_text"`
	ErrorMessage  string `json:"error"`
	Message       string `json:"message,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	Path          string `json:"path,omitempty"`
	Method        string `json:"method,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error %d (%s): %s - %s", e.StatusCode, e.StatusText, e.ErrorMessage, e.Message)
	}
	return fmt.Sprintf("API error %d (%s): %s", e.StatusCode, e.StatusText, e.ErrorMessage)
}

// Unwrap returns the underlying error message
func (e *APIError) Unwrap() error {
	return errors.New(e.ErrorMessage)
}

// IsConflict returns true if the error is a 409 Conflict
func (e *APIError) IsConflict() bool {
	return e.StatusCode == http.StatusConflict
}

// IsNotFound returns true if the error is a 404 Not Found
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401 Unauthorized
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if the error is a 403 Forbidden
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsBadRequest returns true if the error is a 400 Bad Request
func (e *APIError) IsBadRequest() bool {
	return e.StatusCode == http.StatusBadRequest
}

// IsServerError returns true if the error is a 5xx server error
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// Client interface defines the operations that can be performed against the API
type Client interface {
	// Category accessors
	Users() users.UsersClientInterface
	Roles() users.RolesClientInterface
	Keys() users.KeysClientInterface
	Auth() auth.AuthClientInterface
	GHA() gha.GHAClientInterface
	Releases() releases.ReleasesClientInterface
	Aliases() releases.AliasesClientInterface
	Deployments() deployments.DeploymentsClientInterface
	Events() deployments.EventsClientInterface
}

// HTTPClient is an implementation of the Client interface that uses HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
	token      string

	// Category clients
	users       users.UsersClientInterface
	roles       users.RolesClientInterface
	keys        users.KeysClientInterface
	auth        auth.AuthClientInterface
	gha         gha.GHAClientInterface
	releases    releases.ReleasesClientInterface
	aliases     releases.AliasesClientInterface
	deployments deployments.DeploymentsClientInterface
	events      deployments.EventsClientInterface
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

// WithToken sets the JWT token for authentication
func WithToken(token string) ClientOption {
	return func(c *HTTPClient) {
		c.token = token
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

	// Initialize category clients
	client.users = users.NewUsersClient(client.do)
	client.roles = users.NewRolesClient(client.do)
	client.keys = users.NewKeysClient(client.do)
	client.auth = auth.NewAuthClient(client.do)
	client.gha = gha.NewGHAClient(client.do)
	client.releases = releases.NewReleasesClient(client.do)
	client.aliases = releases.NewAliasesClient(client.do)
	client.deployments = deployments.NewDeploymentsClient(client.do)
	client.events = deployments.NewEventsClient(client.do)

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

	// Add JWT token to Authorization header if present
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

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
		// Parse the simple error response format from the server
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(respBodyBytes, &errResp); err != nil {
			// If we can't parse JSON at all, create a basic error
			apiErr := APIError{
				StatusCode:   resp.StatusCode,
				StatusText:   resp.Status,
				ErrorMessage: "Unknown error",
				Message:      string(respBodyBytes),
			}
			return &apiErr
		} else {
			apiErr := APIError{
				StatusCode:   resp.StatusCode,
				StatusText:   resp.Status,
				ErrorMessage: errResp.Error,
				Path:         path,
				Method:       method,
			}
			return &apiErr
		}
	}

	if respBody != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.Unmarshal(respBodyBytes, respBody); err != nil {
			return fmt.Errorf("error unmarshaling response: %w", err)
		}
	}

	return nil
}

// Category accessors
func (c *HTTPClient) Users() users.UsersClientInterface {
	return c.users
}

func (c *HTTPClient) Roles() users.RolesClientInterface {
	return c.roles
}

func (c *HTTPClient) Keys() users.KeysClientInterface {
	return c.keys
}

func (c *HTTPClient) Auth() auth.AuthClientInterface {
	return c.auth
}

func (c *HTTPClient) GHA() gha.GHAClientInterface {
	return c.gha
}

func (c *HTTPClient) Releases() releases.ReleasesClientInterface {
	return c.releases
}

func (c *HTTPClient) Aliases() releases.AliasesClientInterface {
	return c.aliases
}

func (c *HTTPClient) Deployments() deployments.DeploymentsClientInterface {
	return c.deployments
}

func (c *HTTPClient) Events() deployments.EventsClientInterface {
	return c.events
}
