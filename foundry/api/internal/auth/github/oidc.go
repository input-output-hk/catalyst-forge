package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"gopkg.in/square/go-jose.v2/jwt"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks --out ./mocks/oidc.go . GithubActionsOIDCClient

// GithubActionsOIDCClient is an interface that provides a way to verify GitHub Actions ID tokens.
type GithubActionsOIDCClient interface {
	Verify(token, audience string) (*TokenInfo, error)
	StartCache() error
	StopCache()
}

// DefaultGithubActionsOIDCClient is the default implementation of the GithubActionsOIDCClient interface.
type DefaultGithubActionsOIDCClient struct {
	cacher GitHubJWKSCacher
	client *http.Client
	ctx    context.Context
}

// TokenInfo contains the information about a GitHub Actions ID token.
type TokenInfo struct {
	Subject string
	Issuer  string
	Aud     []string
	Issued  time.Time
	Expiry  time.Time

	Repository        string
	RepositoryID      string
	RepositoryOwner   string
	RepositoryOwnerID string
	Ref               string
	SHA               string
	Workflow          string
	JobWorkflowRef    string
	RunID             string
	RunnerEnvironment string
	Environment       string
}

// ghaTokenClaims is the claims of a GitHub Actions ID token.
type ghaTokenClaims struct {
	jwt.Claims

	Repository        string `json:"repository"`
	RepositoryID      string `json:"repository_id,omitempty"`
	RepositoryOwner   string `json:"repository_owner,omitempty"`
	RepositoryOwnerID string `json:"repository_owner_id,omitempty"`
	Ref               string `json:"ref,omitempty"`
	SHA               string `json:"sha,omitempty"`
	Workflow          string `json:"workflow,omitempty"`
	JobWorkflowRef    string `json:"job_workflow_ref,omitempty"`
	RunID             string `json:"run_id,omitempty"`
	RunnerEnvironment string `json:"runner_environment,omitempty"`
	Environment       string `json:"environment,omitempty"`
}

func (g *DefaultGithubActionsOIDCClient) Verify(token, audience string) (*TokenInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("empty token string")
	}

	parsed, err := jwt.ParseSigned(token)
	if err != nil {
		return nil, fmt.Errorf("parse signed token: %w", err)
	}

	ks := g.cacher.JWKS()
	if ks == nil || len(ks.Keys) == 0 {
		return nil, fmt.Errorf("jwks cache is empty")
	}

	kid := parsed.Headers[0].KeyID
	if kid == "" {
		return nil, fmt.Errorf("token missing kid header")
	}

	matched := ks.Key(kid)
	if len(matched) == 0 {
		return nil, fmt.Errorf("kid %q not found in JWKS cache", kid)
	}

	var claims ghaTokenClaims
	if err := parsed.Claims(matched[0].Key, &claims); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	if claims.Issuer != "https://token.actions.githubusercontent.com" {
		return nil, fmt.Errorf("unexpected issuer: %s", claims.Issuer)
	}

	expected := jwt.Expected{
		Time: time.Now(),
	}
	if audience != "" {
		expected.Audience = jwt.Audience{audience}
	}

	if err := claims.ValidateWithLeeway(expected, 60*time.Second); err != nil {
		return nil, err
	}

	return &TokenInfo{
		Subject:           claims.Subject,
		Issuer:            claims.Issuer,
		Aud:               claims.Audience,
		Issued:            claims.IssuedAt.Time(),
		Expiry:            claims.Expiry.Time(),
		Repository:        claims.Repository,
		RepositoryID:      claims.RepositoryID,
		RepositoryOwner:   claims.RepositoryOwner,
		RepositoryOwnerID: claims.RepositoryOwnerID,
		Ref:               claims.Ref,
		SHA:               claims.SHA,
		Workflow:          claims.Workflow,
		JobWorkflowRef:    claims.JobWorkflowRef,
		RunID:             claims.RunID,
		RunnerEnvironment: claims.RunnerEnvironment,
		Environment:       claims.Environment,
	}, nil
}

// StartCache starts the cache for the DefaultGithubActionsOIDCClient.
func (g *DefaultGithubActionsOIDCClient) StartCache() error {
	return g.cacher.Start(g.ctx)
}

// StopCache stops the cache for the DefaultGithubActionsOIDCClient.
func (g *DefaultGithubActionsOIDCClient) StopCache() {
	g.cacher.Stop()
}

// NewDefaultGithubActionsOIDCClient creates a new DefaultGithubActionsOIDCClient instance.
func NewDefaultGithubActionsOIDCClient(ctx context.Context, cachePath string) (*DefaultGithubActionsOIDCClient, error) {
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

	cacher := NewDefaultGitHubJWKSCacher(ctx, cachePath)

	return &DefaultGithubActionsOIDCClient{
		client: httpClient,
		cacher: cacher,
		ctx:    ctx,
	}, nil
}
