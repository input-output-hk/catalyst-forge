package github

import (
	"log/slog"
	"os"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
)

type DefaultGithubClientOption func(*DefaultGithubClient)
type DefaultGithubClientOptions struct {
	Creds *common.Secret
	Token string
}

// WithCreds sets the credentials for the Github client.
func WithCreds(creds *common.Secret) DefaultGithubClientOption {
	return func(c *DefaultGithubClient) {
		c.opts.Creds = creds
	}
}

// WithCredsOrEnv sets the credentials for the Github client.
// If a GITHUB_TOKEN environment variable is set, it will use that token.
// Otherwise, it will use the given secret.
func WithCredsOrEnv(creds *common.Secret) DefaultGithubClientOption {
	token, exists := os.LookupEnv("GITHUB_TOKEN")
	if exists {
		return func(c *DefaultGithubClient) {
			c.opts.Token = token
		}
	}

	return func(c *DefaultGithubClient) {
		c.opts.Creds = creds
	}
}

// WithFs sets the filesystem for the Github client.
func WithFs(fs fs.Filesystem) DefaultGithubClientOption {
	return func(c *DefaultGithubClient) {
		c.fs = fs
	}
}

// WithGithubEnv sets the Github environment for the Github client.
func WithGithubEnv(env GithubEnv) DefaultGithubClientOption {
	return func(c *DefaultGithubClient) {
		c.env = env
	}
}

// WithGithubClient sets the native Github client for the Github client.
func WithGithubClient(client *github.Client) DefaultGithubClientOption {
	return func(c *DefaultGithubClient) {
		c.client = client
	}
}

// WithLogger sets the logger for the Github client.
func WithLogger(logger *slog.Logger) DefaultGithubClientOption {
	return func(c *DefaultGithubClient) {
		c.logger = logger
	}
}

// WithSecretStore sets the secret store for the Github client.
func WithSecretStore(ss *secrets.SecretStore) DefaultGithubClientOption {
	return func(c *DefaultGithubClient) {
		c.secretStore = ss
	}
}
