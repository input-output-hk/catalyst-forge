package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/name"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// compile-time check that DefaultAuth implements AuthProvider
var _ AuthProvider = (*DefaultAuth)(nil)

// AuthProvider provides authentication for registry operations
type AuthProvider interface {
	// Authenticator returns an authenticator for the given registry.
	// Returns nil for anonymous access.
	// The returned value should be compatible with both ORAS and ggcr.
	Authenticator(registry string) (any, error)
}

// DefaultAuth provides authentication using Docker config
type DefaultAuth struct {
	// Optional: override the keychain (useful for testing)
	keychain authn.Keychain
}

// Authenticator returns an authenticator using Docker config (~/.docker/config.json)
func (d *DefaultAuth) Authenticator(registry string) (any, error) {
	kc := d.keychain
	if kc == nil {
		kc = authn.DefaultKeychain
	}

	// For ggcr, we return the keychain directly
	// For ORAS, we need to convert to ORAS auth
	// Since we return 'any', the caller will type-assert as needed
	return &multiAuth{
		keychain: kc,
		registry: registry,
	}, nil
}

// multiAuth provides authentication that works with both ORAS and ggcr
type multiAuth struct {
	keychain authn.Keychain
	registry string
}

// GetAuthn returns a ggcr Authenticator for the registry
func (m *multiAuth) GetAuthn() (authn.Authenticator, error) {
	// Parse registry to get the proper resource
	resource, err := name.NewRegistry(m.registry)
	if err != nil {
		// Fallback to anonymous if parsing fails
		return authn.Anonymous, nil
	}

	// Get authenticator from keychain
	auth, err := m.keychain.Resolve(resource)
	if err != nil {
		// Fallback to anonymous on error
		return authn.Anonymous, nil
	}

	return auth, nil
}

// GetCredential returns credentials for ORAS
func (m *multiAuth) GetCredential(ctx context.Context, registry string) (auth.Credential, error) {
	// Get ggcr authenticator
	authenticator, err := m.GetAuthn()
	if err != nil {
		return auth.EmptyCredential, err
	}

	// Get auth config from authenticator
	authConfig, err := authenticator.Authorization()
	if err != nil {
		return auth.EmptyCredential, err
	}

	// Convert to ORAS credential
	if authConfig == nil || authConfig.Username == "" && authConfig.Password == "" && authConfig.Auth == "" && authConfig.IdentityToken == "" {
		return auth.EmptyCredential, nil
	}

	cred := auth.Credential{}

	// Handle different auth types
	if authConfig.IdentityToken != "" {
		// Bearer token auth
		cred.AccessToken = authConfig.IdentityToken
	} else if authConfig.Auth != "" {
		// Base64 encoded username:password
		decoded, err := base64.StdEncoding.DecodeString(authConfig.Auth)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				cred.Username = parts[0]
				cred.Password = parts[1]
			}
		}
	} else {
		// Plain username/password
		cred.Username = authConfig.Username
		cred.Password = authConfig.Password
	}

	// Set refresh token if available
	if authConfig.RegistryToken != "" {
		cred.RefreshToken = authConfig.RegistryToken
	}

	return cred, nil
}

// StaticAuth provides static authentication with username/password
type StaticAuth struct {
	Username string
	Password string
	Token    string // Optional: use token instead of username/password
}

// Authenticator returns a static authenticator
func (s *StaticAuth) Authenticator(registry string) (any, error) {
	if s.Token != "" {
		return &staticTokenAuth{token: s.Token}, nil
	}
	return &staticBasicAuth{
		username: s.Username,
		password: s.Password,
	}, nil
}

// staticBasicAuth provides basic authentication
type staticBasicAuth struct {
	username string
	password string
}

func (s *staticBasicAuth) GetAuthn() (authn.Authenticator, error) {
	return &authn.Basic{
		Username: s.username,
		Password: s.password,
	}, nil
}

func (s *staticBasicAuth) GetCredential(ctx context.Context, registry string) (auth.Credential, error) {
	return auth.Credential{
		Username: s.username,
		Password: s.password,
	}, nil
}

// staticTokenAuth provides token authentication
type staticTokenAuth struct {
	token string
}

func (s *staticTokenAuth) GetAuthn() (authn.Authenticator, error) {
	return &authn.Bearer{Token: s.token}, nil
}

func (s *staticTokenAuth) GetCredential(ctx context.Context, registry string) (auth.Credential, error) {
	return auth.Credential{
		AccessToken: s.token,
	}, nil
}

// GitHubAuth provides authentication for GitHub Container Registry
type GitHubAuth struct {
	Token string // GitHub personal access token or GITHUB_TOKEN
}

// Authenticator returns a GitHub authenticator
func (g *GitHubAuth) Authenticator(registry string) (any, error) {
	// Only use GitHub auth for ghcr.io
	if !strings.Contains(registry, "ghcr.io") {
		return &DefaultAuth{}, nil
	}

	token := g.Token
	if token == "" {
		// Try to get from GitHub keychain (uses GITHUB_TOKEN env var)
		return &multiAuth{
			keychain: github.Keychain,
			registry: registry,
		}, nil
	}

	return &staticTokenAuth{token: token}, nil
}

// ECRAuth provides authentication for AWS Elastic Container Registry
// Note: This is a simplified version. Production usage should integrate with AWS SDK
type ECRAuth struct {
	// In production, you'd have AWS session, region, etc.
	// This is a placeholder for the pattern
}

// Authenticator returns an ECR authenticator
func (e *ECRAuth) Authenticator(registry string) (any, error) {
	// Check if this is an ECR registry
	if !strings.Contains(registry, ".ecr.") || !strings.Contains(registry, ".amazonaws.com") {
		// Not ECR, fallback to default
		return (&DefaultAuth{}).Authenticator(registry)
	}

	// In production, you would:
	// 1. Use AWS SDK to get ECR auth token
	// 2. Return appropriate authenticator
	// For now, we'll return an error indicating ECR auth needs implementation
	return nil, fmt.Errorf("ECR authentication not yet implemented - use docker login or AWS CLI")
}

// ChainAuth tries multiple auth providers in sequence
type ChainAuth struct {
	Providers []AuthProvider
}

// Authenticator tries each provider until one succeeds
func (c *ChainAuth) Authenticator(registry string) (any, error) {
	var lastErr error

	for _, provider := range c.Providers {
		auth, err := provider.Authenticator(registry)
		if err == nil && auth != nil {
			return auth, nil
		}
		if err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	// Fallback to anonymous
	return nil, nil
}

// ToGGCRAuth converts an auth object to a ggcr authenticator
func ToGGCRAuth(authObj any) (authn.Authenticator, error) {
	if authObj == nil {
		return authn.Anonymous, nil
	}
	// Direct ggcr authenticator
	if a, ok := authObj.(authn.Authenticator); ok {
		return a, nil
	}
	// Provider exposing GetAuthn
	if provider, ok := authObj.(interface {
		GetAuthn() (authn.Authenticator, error)
	}); ok {
		return provider.GetAuthn()
	}
	return authn.Anonymous, nil
}

// ToORASAuth converts an auth object to an ORAS credential
func ToORASAuth(ctx context.Context, authObj any, registry string) (auth.Credential, error) {
	if authObj == nil {
		return auth.EmptyCredential, nil
	}
	// Direct ORAS credential
	if cred, ok := authObj.(*auth.Credential); ok {
		return *cred, nil
	}
	// Provider exposing GetCredential
	if provider, ok := authObj.(interface {
		GetCredential(context.Context, string) (auth.Credential, error)
	}); ok {
		return provider.GetCredential(ctx, registry)
	}
	return auth.EmptyCredential, nil
}
