package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"

	"github.com/catalyst-forge/services/clients/go/generated"
)

// FoundryClient provides a high-level interface to the Foundry API
type FoundryClient struct {
	*generated.ClientWithResponses
	config *ClientConfig
	tokens *TokenStore
}

// NewFoundryClient creates a new high-level Foundry client
func NewFoundryClient(config *ClientConfig) (*FoundryClient, error) {
	// Ensure cookie jar
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{}
	}
	if config.HTTPClient.Jar == nil {
		jar, _ := cookiejar.New(nil)
		config.HTTPClient.Jar = jar
	}
	// Wrap transport
	base := config.HTTPClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	ts := NewTokenStore()
	aat := NewAutoAuthTransport(base, ts, config.BaseURL)
	aat.Jar = config.HTTPClient.Jar
	config.HTTPClient.Transport = aat

	client, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build client: %w", err)
	}

	return &FoundryClient{
		ClientWithResponses: client,
		config:              config,
		tokens:              ts,
	}, nil
}

// NewFoundryClientSimple creates a client with just a base URL and token
func NewFoundryClientSimple(baseURL, token string) (*FoundryClient, error) {
	config := NewDefaultConfig(baseURL).
		WithAuth(NewBearerTokenProvider(token))

	return NewFoundryClient(config)
}

// NewFoundryClientFromEnv creates a client from environment variables
func NewFoundryClientFromEnv() (*FoundryClient, error) {
	baseURL := getEnvOrDefault("FOUNDRY_API_URL", "https://api.foundry.example.com")

	// Try different auth methods in order of preference
	var authProvider AuthProvider

	// 1. Try bearer token
	if token := getEnvVar("FOUNDRY_API_TOKEN"); token != "" {
		authProvider = NewBearerTokenProvider(token)
	} else if token := getEnvVar("FOUNDRY_TOKEN"); token != "" {
		authProvider = NewBearerTokenProvider(token)
	} else if apiKey := getEnvVar("FOUNDRY_API_KEY"); apiKey != "" {
		// 2. Try API key
		authProvider = NewAPIKeyProvider(apiKey, "")
	} else if ghProvider, err := NewGitHubActionsProviderFromEnv(); err == nil {
		// 3. Try GitHub Actions
		authProvider = ghProvider
	}

	if authProvider == nil {
		return nil, fmt.Errorf("no authentication credentials found in environment")
	}

	config := NewDefaultConfig(baseURL).WithAuth(authProvider)
	return NewFoundryClient(config)
}

// HealthCheck performs a health check on the API
func (c *FoundryClient) HealthCheck(ctx context.Context) error {
	resp, err := c.GetHealthz(ctx, generated.GetHealthzJSONRequestBody{})
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// GetJWKS retrieves the JSON Web Key Set
func (c *FoundryClient) GetJWKS(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.GetWellKnownJwksJsonWithResponse(ctx, generated.GetWellKnownJwksJsonJSONRequestBody{})
	if err != nil {
		return nil, fmt.Errorf("failed to get JWKS: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("JWKS request failed with status: %d", resp.StatusCode())
	}

	// Type assertion for the response
	if resp.JSON200 != nil {
		return *resp.JSON200, nil
	}

	return nil, fmt.Errorf("empty JWKS response")
}

// SetAccessToken sets the current access token used by the auto-auth transport.
func (c *FoundryClient) SetAccessToken(token string) {
	if c.tokens != nil {
		c.tokens.Set(token)
	}
}

// GetAccessToken returns the current access token.
func (c *FoundryClient) GetAccessToken() string {
	if c.tokens != nil {
		return c.tokens.Get()
	}
	return ""
}

// Helper functions

func getEnvVar(key string) string {
	return os.Getenv(key)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := getEnvVar(key); value != "" {
		return value
	}
	return defaultValue
}

// Error types for better error handling

// APIError represents an error response from the API
type APIError struct {
	StatusCode int
	Message    string
	Body       []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound checks if an error is a 404 Not Found error
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// IsUnauthorized checks if an error is a 401 Unauthorized error
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusUnauthorized
	}
	return false
}

// IsForbidden checks if an error is a 403 Forbidden error
func IsForbidden(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusForbidden
	}
	return false
}
