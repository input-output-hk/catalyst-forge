package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"gopkg.in/square/go-jose.v2"
)

// GitHubOIDC handles everything related to verifying GitHub Actions ID tokens.
type GitHubOIDC struct {
	client *http.Client
	jwks   *jose.JSONWebKeySet
}

// fetchJWKS fetches the JWKS from the GitHub Actions endpoint.
func (g *GitHubOIDC) fetchJWKS(ctx context.Context) (*jose.JSONWebKeySet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubJWKSURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return nil, fmt.Errorf("JWKS contained no keys")
	}
	return &jwks, nil
}

// NewGitHubOIDC creates a new GitHubOIDC instance.
func NewGitHubOIDC(ctx context.Context) (*GitHubOIDC, error) {
	// You can customize the transport here (proxy, keep‑alives, etc.).
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}

	return &GitHubOIDC{
		client: httpClient,
	}, nil
}
