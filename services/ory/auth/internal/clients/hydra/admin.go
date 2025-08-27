// Package hydra provides a minimal client for the Ory Hydra Admin API
// endpoints used by the auth orchestrator service.
package hydra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// AcceptConsentBody describes the payload for accepting consent.
type AcceptConsentBody struct {
	GrantScope               []string       `json:"grant_scope"`
	GrantAccessTokenAudience []string       `json:"grant_access_token_audience"`
	Remember                 bool           `json:"remember"`
	RememberFor              int            `json:"remember_for"`
	Session                  map[string]any `json:"session,omitempty"`
}

// ConsentRequest is a subset of Hydra's consent request payload.
type ConsentRequest struct {
	Skip                         bool     `json:"skip"`
	Subject                      string   `json:"subject"`
	RequestedScope               []string `json:"requested_scope"`
	RequestedAccessTokenAudience []string `json:"requested_access_token_audience"`
}

// LoginRequest is a subset of Hydra's login request payload.
type LoginRequest struct {
	Skip    bool   `json:"skip"`
	Subject string `json:"subject"`
}

type acceptConsentResponse struct {
	RedirectTo string `json:"redirect_to"`
}

type acceptLoginBody struct {
	Subject     string `json:"subject"`
	Remember    bool   `json:"remember"`
	RememberFor int    `json:"remember_for"`
}

type acceptLoginResponse struct {
	RedirectTo string `json:"redirect_to"`
}

// AdminClient is a small wrapper around Hydra's Admin HTTP API.
type AdminClient struct {
	// BaseURL is the Hydra Admin base address, e.g. http://hydra-admin:4445
	BaseURL string
	// HTTPClient is the HTTP client used for requests. It may be http.DefaultClient.
	HTTPClient *http.Client
}

// NewAdminClient constructs an AdminClient with a normalized base URL.
func NewAdminClient(baseURL string, httpClient *http.Client) *AdminClient {
	return &AdminClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: httpClient,
	}
}

// GetLoginRequest fetches the login request details for a given challenge.
func (c *AdminClient) GetLoginRequest(challenge string) (*LoginRequest, error) {
	endpoint := fmt.Sprintf(
		"%s/oauth2/auth/requests/login?login_challenge=%s",
		c.BaseURL,
		url.QueryEscape(challenge),
	)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hydra get login request: status %d", resp.StatusCode)
	}

	var parsed LoginRequest
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}
	return &parsed, nil
}

// AcceptLoginRequest finalizes the login step and returns Hydra's redirect URL.
func (c *AdminClient) AcceptLoginRequest(challenge, subject string, remember bool, rememberFor int) (string, error) {
	endpoint := fmt.Sprintf(
		"%s/oauth2/auth/requests/login/accept?login_challenge=%s",
		c.BaseURL,
		url.QueryEscape(challenge),
	)

	payload := acceptLoginBody{
		Subject:     subject,
		Remember:    remember,
		RememberFor: rememberFor,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hydra accept login: status %d", resp.StatusCode)
	}

	var parsed acceptLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode body: %w", err)
	}
	return parsed.RedirectTo, nil
}

// GetConsentRequest fetches the consent request details for a given challenge.
func (c *AdminClient) GetConsentRequest(challenge string) (*ConsentRequest, error) {
	endpoint := fmt.Sprintf(
		"%s/oauth2/auth/requests/consent?consent_challenge=%s",
		c.BaseURL,
		url.QueryEscape(challenge),
	)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hydra get consent request: status %d", resp.StatusCode)
	}

	var parsed ConsentRequest
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}
	return &parsed, nil
}

// AcceptConsentRequest finalizes the consent step and returns Hydra's redirect URL.
func (c *AdminClient) AcceptConsentRequest(challenge string, body AcceptConsentBody) (string, error) {
	endpoint := fmt.Sprintf(
		"%s/oauth2/auth/requests/consent/accept?consent_challenge=%s",
		c.BaseURL,
		url.QueryEscape(challenge),
	)

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hydra accept consent: status %d", resp.StatusCode)
	}

	var parsed acceptConsentResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decode body: %w", err)
	}
	return parsed.RedirectTo, nil
}
