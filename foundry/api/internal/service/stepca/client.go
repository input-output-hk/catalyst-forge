package stepca

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StepCAClient interface for interacting with step-ca
type StepCAClient interface {
	SignCertificate(token string, csr []byte, ttl time.Duration) (*SignResponse, error)
	GetRootCertificate() ([]byte, error)
	GetSignEndpointURL() string
}

// Client implements the StepCAClient interface for interacting with step-ca
type Client struct {
	baseURL    string
	httpClient *http.Client
	rootCA     []byte
}

// Config holds configuration for the step-ca client
type Config struct {
	// BaseURL is the step-ca server URL (e.g., "https://step-ca:9000")
	BaseURL string

	// RootCA is the PEM-encoded root certificate for TLS verification
	RootCA []byte

	// InsecureSkipVerify skips TLS verification (only for testing!)
	InsecureSkipVerify bool

	// Timeout for HTTP requests
	Timeout time.Duration
}

// NewClient creates a new step-ca client
func NewClient(config Config) (*Client, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("step-ca base URL is required")
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	// Add root CA to TLS config if provided
	if len(config.RootCA) > 0 {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(config.RootCA) {
			return nil, fmt.Errorf("failed to parse root CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return &Client{
		baseURL:    config.BaseURL,
		httpClient: httpClient,
		rootCA:     config.RootCA,
	}, nil
}

// SignRequest represents the request to sign a certificate
type SignRequest struct {
	CSR       string `json:"csr"`
	OTT       string `json:"ott"`                 // One-time token (JWT)
	NotAfter  string `json:"notAfter,omitempty"`  // Certificate expiry time/duration
	NotBefore string `json:"notBefore,omitempty"` // Certificate validity start time/duration
}

// SignResponse represents the response from signing a certificate
type SignResponse struct {
	CRT       string   `json:"crt"`       // Signed certificate
	CA        string   `json:"ca"`        // CA certificate
	TLS       bool     `json:"tls"`       // Whether this is a TLS certificate
	X509Chain []string `json:"x509Chain"` // Certificate chain
}

// SignCertificate sends a CSR to step-ca for signing
func (c *Client) SignCertificate(token string, csr []byte, ttl time.Duration) (*SignResponse, error) {
	// step-ca expects CSR as PEM string directly, not base64 encoded
	csrPEM := string(csr)

	// Create sign request
	signReq := SignRequest{
		CSR: csrPEM,
		OTT: token,
	}

	// If TTL is specified, set notAfter to control certificate validity
	if ttl > 0 {
		// step-ca accepts duration strings like "5m", "24h", etc.
		signReq.NotAfter = ttl.String()
	}

	reqBody, err := json.Marshal(signReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sign request: %w", err)
	}

	// Send request to step-ca
	url := fmt.Sprintf("%s/1.0/sign", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to step-ca: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// Try to parse error response
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
			return nil, fmt.Errorf("step-ca error: %s", errorResp.Error)
		}
		return nil, fmt.Errorf("step-ca returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var signResp SignResponse
	if err := json.Unmarshal(body, &signResp); err != nil {
		return nil, fmt.Errorf("failed to parse sign response: %w", err)
	}

	// step-ca returns certificate as PEM string directly

	return &signResp, nil
}

// GetRootCertificate retrieves the root certificate from step-ca
func (c *Client) GetRootCertificate() ([]byte, error) {
	// If we have it cached, return it
	if len(c.rootCA) > 0 {
		return c.rootCA, nil
	}

	// Fetch from step-ca
	url := fmt.Sprintf("%s/roots.pem", c.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch root certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get root certificate, status %d: %s", resp.StatusCode, string(body))
	}

	rootCA, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read root certificate: %w", err)
	}

	// Cache it
	c.rootCA = rootCA

	return rootCA, nil
}

// GetSignEndpointURL returns the full URL for the step-ca sign endpoint
func (c *Client) GetSignEndpointURL() string {
	return fmt.Sprintf("%s/1.0/sign", c.baseURL)
}

// HealthCheck checks if step-ca is healthy
func (c *Client) HealthCheck() error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("step-ca health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("step-ca unhealthy, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
