// Package kratos provides a minimal client for Ory Kratos public endpoints
// used by the auth orchestrator service.
package kratos

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PublicClient is a thin client for Kratos public endpoints.
type PublicClient struct {
	// BaseURL is the Kratos Public base address, e.g. http://kratos-public:4433
	BaseURL string
	// HTTPClient is the HTTP client used for requests.
	HTTPClient *http.Client
}

// WhoAmIResponse models the subset of the Kratos whoami response we need.
type WhoAmIResponse struct {
	Identity struct {
		ID     string         `json:"id"`
		Traits map[string]any `json:"traits"`
		// Other fields can be added when needed
	} `json:"identity"`
}

// NewPublicClient constructs a PublicClient with a normalized base URL.
func NewPublicClient(baseURL string, httpClient *http.Client) *PublicClient {
	return &PublicClient{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: httpClient}
}

// WhoAmI checks for an active session using the provided cookies.
func (c *PublicClient) WhoAmI(cookies []*http.Cookie) (*http.Response, error) {
	endpoint := fmt.Sprintf("%s/sessions/whoami", c.BaseURL)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	return c.HTTPClient.Do(req)
}

// WhoAmIJSON performs whoami and decodes the JSON response when status is 200.
// It returns the parsed response and the HTTP status code. If status is not 200,
// the parsed response will be nil and err will be nil.
func (c *PublicClient) WhoAmIJSON(cookies []*http.Cookie) (*WhoAmIResponse, int, error) {
	resp, err := c.WhoAmI(cookies)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, nil
	}
	var parsed WhoAmIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, resp.StatusCode, err
	}
	return &parsed, resp.StatusCode, nil
}

// LoginRedirect builds a Kratos login UI URL with a return_to parameter.
func (c *PublicClient) LoginRedirect(returnTo string) string {
	v := url.Values{"return_to": {returnTo}}
	return fmt.Sprintf("%s/self-service/login/browser?%s", c.BaseURL, v.Encode())
}
