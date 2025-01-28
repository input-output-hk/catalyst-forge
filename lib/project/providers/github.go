package providers

// import (
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"os"

// 	"github.com/google/go-github/v66/github"
// 	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
// 	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
// 	"github.com/spf13/afero"
// )

// var (
// 	ErrNoEventFound = fmt.Errorf("no GitHub event data found")
// )

// type GithubProvider struct {
// 	fs     afero.Fs
// 	logger *slog.Logger
// 	store  *secrets.SecretStore
// }

// // GetEventType returns the GitHub event type.
// func (g *GithubProvider) GetEventType() string {
// 	return os.Getenv("GITHUB_EVENT_NAME")
// }

// // GetEventPayload returns the GitHub event payload.
// func (g *GithubProvider) GetEventPayload() (any, error) {
// 	path, pathExists := os.LookupEnv("GITHUB_EVENT_PATH")
// 	name, nameExists := os.LookupEnv("GITHUB_EVENT_NAME")

// 	if !pathExists || !nameExists {
// 		return nil, ErrNoEventFound
// 	}

// 	g.logger.Debug("Reading GitHub event data", "path", path, "name", name)
// 	payload, err := afero.ReadFile(g.fs, path)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read GitHub event data: %w", err)
// 	}

// 	event, err := github.ParseWebHook(name, payload)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse GitHub event data: %w", err)
// 	}

// 	return event, nil
// }

// // HasEvent returns whether a GitHub event payload exists.
// func (g *GithubProvider) HasEvent() bool {
// 	_, pathExists := os.LookupEnv("GITHUB_EVENT_PATH")
// 	_, nameExists := os.LookupEnv("GITHUB_EVENT_NAME")
// 	return pathExists && nameExists
// }

// // NewClient returns a new GitHub client.
// func (g *GithubProvider) NewClient(secret *schema.Secret) *github.Client {
// 	token, exists := os.LookupEnv("GITHUB_TOKEN")
// 	if exists {
// 		return github.NewClient(nil).WithAuthToken(token)
// 	} else if secret != nil {
// 		if secret.Maps != nil {
// 			secretMap, err := secrets.GetSecretMap(secret, g.store, g.logger)
// 			if err != nil {
// 				g.logger.Error("Failed to get Github secret", "error", err)
// 			} else {
// 				token, exists := secretMap["token"]
// 				if exists {
// 					return github.NewClient(nil).WithAuthToken(token)
// 				} else {
// 					g.logger.Error("Github secret map does not contain token")
// 				}
// 			}
// 		} else {
// 			secret, err := secrets.GetSecret(secret, g.store, g.logger)
// 			if err != nil {
// 				g.logger.Error("Failed to get Github secret", "error", err)
// 			} else {
// 				return github.NewClient(nil).WithAuthToken(secret)
// 			}
// 		}
// 	}

// 	g.logger.Warn("No Github token found, using unauthenticated client")
// 	return github.NewClient(nil)
// }

// // NewDefaultGithubProvider returns a new default GitHub provider.
// func NewDefaultGithubProvider(logger *slog.Logger) GithubProvider {
// 	if logger == nil {
// 		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
// 	}

// 	fs := afero.NewOsFs()
// 	store := secrets.NewDefaultSecretStore()
// 	return GithubProvider{
// 		fs:     fs,
// 		logger: logger,
// 		store:  &store,
// 	}
// }

// // NewGithubProvider returns a new GitHub provider.
// func NewGithubProvider(fs afero.Fs, logger *slog.Logger, store *secrets.SecretStore) GithubProvider {
// 	if logger == nil {
// 		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
// 	}

// 	return GithubProvider{
// 		fs:     fs,
// 		logger: logger,
// 		store:  store,
// 	}
// }
