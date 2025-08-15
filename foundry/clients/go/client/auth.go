package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/catalyst-forge/foundry/clients/go/generated"
)

// AuthProvider defines the interface for authentication providers
type AuthProvider interface {
	// Intercept modifies the request to add authentication
	Intercept(ctx context.Context, req *http.Request) error
}

// BearerTokenProvider implements bearer token authentication
type BearerTokenProvider struct {
	token string
}

// NewBearerTokenProvider creates a new bearer token provider
func NewBearerTokenProvider(token string) *BearerTokenProvider {
	return &BearerTokenProvider{token: token}
}

// NewBearerTokenProviderFromEnv creates a bearer token provider from environment variable
func NewBearerTokenProviderFromEnv(envVar string) (*BearerTokenProvider, error) {
	token := os.Getenv(envVar)
	if token == "" {
		return nil, fmt.Errorf("environment variable %s is not set", envVar)
	}
	return NewBearerTokenProvider(token), nil
}

// Intercept adds the bearer token to the request
func (b *BearerTokenProvider) Intercept(ctx context.Context, req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", b.token))
	return nil
}

// APIKeyProvider implements API key authentication
type APIKeyProvider struct {
	key        string
	headerName string
}

// NewAPIKeyProvider creates a new API key provider
func NewAPIKeyProvider(key, headerName string) *APIKeyProvider {
	if headerName == "" {
		headerName = "X-API-Key"
	}
	return &APIKeyProvider{
		key:        key,
		headerName: headerName,
	}
}

// Intercept adds the API key to the request
func (a *APIKeyProvider) Intercept(ctx context.Context, req *http.Request) error {
	req.Header.Set(a.headerName, a.key)
	return nil
}

// BasicAuthProvider implements basic authentication
type BasicAuthProvider struct {
	username string
	password string
}

// NewBasicAuthProvider creates a new basic auth provider
func NewBasicAuthProvider(username, password string) *BasicAuthProvider {
	return &BasicAuthProvider{
		username: username,
		password: password,
	}
}

// Intercept adds basic auth to the request
func (b *BasicAuthProvider) Intercept(ctx context.Context, req *http.Request) error {
	req.SetBasicAuth(b.username, b.password)
	return nil
}

// GitHubActionsProvider implements GitHub Actions OIDC token authentication
type GitHubActionsProvider struct {
	token    string
	audience string
}

// NewGitHubActionsProvider creates a provider for GitHub Actions authentication
func NewGitHubActionsProvider(token, audience string) *GitHubActionsProvider {
	return &GitHubActionsProvider{
		token:    token,
		audience: audience,
	}
}

// NewGitHubActionsProviderFromEnv creates a GitHub Actions provider from environment
func NewGitHubActionsProviderFromEnv() (*GitHubActionsProvider, error) {
	token := os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("not running in GitHub Actions environment")
	}

	audience := os.Getenv("FOUNDRY_AUDIENCE")
	if audience == "" {
		audience = "foundry"
	}

	return NewGitHubActionsProvider(token, audience), nil
}

// Intercept adds the GitHub Actions token to the request
func (g *GitHubActionsProvider) Intercept(ctx context.Context, req *http.Request) error {
	// For GitHub Actions, we typically need to exchange the OIDC token for an API token
	// This is a placeholder - actual implementation would call the auth/github/login endpoint
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.token))
	return nil
}

// ClientConfig holds configuration for creating a Foundry client
type ClientConfig struct {
	BaseURL        string
	AuthProvider   AuthProvider
	HTTPClient     *http.Client
	RequestTimeout time.Duration
}

// NewDefaultConfig creates a default client configuration
func NewDefaultConfig(baseURL string) *ClientConfig {
	return &ClientConfig{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		RequestTimeout: 30 * time.Second,
	}
}

// WithAuth sets the authentication provider
func (c *ClientConfig) WithAuth(provider AuthProvider) *ClientConfig {
	c.AuthProvider = provider
	return c
}

// WithHTTPClient sets a custom HTTP client
func (c *ClientConfig) WithHTTPClient(client *http.Client) *ClientConfig {
	c.HTTPClient = client
	return c
}

// WithTimeout sets the request timeout
func (c *ClientConfig) WithTimeout(timeout time.Duration) *ClientConfig {
	c.RequestTimeout = timeout
	if c.HTTPClient != nil {
		c.HTTPClient.Timeout = timeout
	}
	return c
}

// Build creates a new ClientWithResponses with the configured options
func (c *ClientConfig) Build() (*generated.ClientWithResponses, error) {
	opts := []generated.ClientOption{
		generated.WithHTTPClient(c.HTTPClient),
	}

	if c.AuthProvider != nil {
		opts = append(opts, generated.WithRequestEditorFn(c.AuthProvider.Intercept))
	}

	return generated.NewClientWithResponses(c.BaseURL, opts...)
}
