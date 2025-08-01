package earthly

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor"
	secretstore "github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
)

// EarthlyExecutorOption is an option for configuring an EarthlyExecutor.
type EarthlyExecutorOption func(e *EarthlyExecutor)

// EarthlySecret represents a secret to be passed to Earthly.
type EarthlySecret struct {
	Id    string
	Value string
}

// earthlyExecutorOptions contains the configuration options for an
// EarthlyExecutor.
type earthlyExecutorOptions struct {
	artifact  string
	ci        bool
	platforms []string
	retries   sc.CIRetries
}

// EarthlyExecutor is an Executor that runs Earthly targets.
type EarthlyExecutor struct {
	logger       *slog.Logger
	opts         earthlyExecutorOptions
	executor     executor.Executor
	earthfile    string
	earthlyArgs  []string
	secrets      []sc.Secret
	secretsStore secretstore.SecretStore
	target       string
	targetArgs   []string
}

// EarthlyExecutionResult contains the results of an Earthly execution.
type EarthlyExecutionResult struct {
	Artifacts map[string]string `json:"artifacts"`
	Images    map[string]string `json:"images"`
}

// Run executes the Earthly target and returns the resulting images and
// artifacts.
func (e EarthlyExecutor) Run() error {
	var (
		err     error
		output  []byte
		secrets []EarthlySecret
	)

	if e.secrets != nil {
		secrets, err = e.buildSecrets()
		if err != nil {
			return err
		}

		if len(secrets) > 0 {
			var secretString []string
			for _, secret := range secrets {
				e.logger.Info("Adding Earthly secret", "earthly_id", secret.Id)
				secretString = append(secretString, fmt.Sprintf("%s=%s", secret.Id, secret.Value))
			}

			if err := os.Setenv("EARTHLY_SECRETS", strings.Join(secretString, ",")); err != nil {
				e.logger.Error("Failed to set secret environment varibles", "envvar", "EARTHLY_SECRETS")
			}
		}
	}

	if e.opts.platforms == nil {
		e.opts.platforms = []string{GetBuildPlatform()}
	}

	var attempts int
	for _, platform := range e.opts.platforms {
		for i := 0; i < int(e.opts.retries.Attempts)+1; i++ {
			attempts++
			arguments := e.buildArguments(platform)

			e.logger.Info("Executing Earthly",
				"attempt", i,
				"attempts", e.opts.retries.Attempts,
				"filters", e.opts.retries.Filters,
				"arguments", arguments,
				"platform", platform,
			)
			output, err = e.executor.Execute("earthly", arguments...)
			if err == nil {
				break
			}

			if len(e.opts.retries.Filters) > 0 {
				found := false
				for _, filter := range e.opts.retries.Filters {
					if strings.Contains(string(output), filter) {
						e.logger.Info("Found filter", "filter", filter)
						found = true
						break
					}
				}

				if !found {
					e.logger.Info("No filter found", "filters", e.opts.retries.Filters)
					break
				}
			}

			e.logger.Error("Failed to run Earthly", "error", err)

			if e.opts.retries.Delay != "" {
				delay, err := time.ParseDuration(e.opts.retries.Delay)
				if err != nil {
					e.logger.Error("Failed to parse delay duration", "error", err)
				} else {
					e.logger.Info("Sleeping for delay", "delay", delay)
					time.Sleep(delay)
				}
			}
		}
	}

	if err != nil {
		if attempts > 1 {
			e.logger.Error(fmt.Sprintf("Failed to run Earthly after %d attempts", attempts), "error", err)
		} else {
			e.logger.Error("Failed to run Earthly", "error", err)
		}
		return fmt.Errorf("failed to run Earthly: %w", err)
	}

	if attempts > 1 {
		e.logger.Info(fmt.Sprintf("Earthly run succeeded after %d attempts", attempts))
	}

	return nil
}

// buildArguments constructs the arguments to pass to the Earthly target.
func (e *EarthlyExecutor) buildArguments(platform string) []string {
	var earthlyArgs []string

	earthlyArgs = append(earthlyArgs, "--platform", platform)
	earthlyArgs = append(earthlyArgs, e.earthlyArgs...)

	// If we have an artifact path and multiple platforms, we need to append the platform to the artifact path to avoid conflicts.
	if e.opts.artifact != "" {
		earthlyArgs = append(earthlyArgs, "--artifact", fmt.Sprintf("%s+%s/*", e.earthfile, e.target), path.Join(e.opts.artifact, platform)+"/")
	} else if e.opts.artifact != "" && len(e.opts.platforms) <= 1 {
		earthlyArgs = append(earthlyArgs, "--artifact", fmt.Sprintf("%s+%s/*", e.earthfile, e.target), e.opts.artifact)
	} else {
		earthlyArgs = append(earthlyArgs, fmt.Sprintf("%s+%s", e.earthfile, e.target))
	}

	earthlyArgs = append(earthlyArgs, e.targetArgs...)

	return earthlyArgs
}

// buildSecrets constructs the secrets to pass to Earthly.
func (e *EarthlyExecutor) buildSecrets() ([]EarthlySecret, error) {
	var secrets []EarthlySecret

	for _, secret := range e.secrets {
		if secret.Name != "" && len(secret.Maps) > 0 {
			e.logger.Error("Secret contains both name and maps", "name", secret.Name, "maps", secret.Maps)
			return nil, fmt.Errorf("secret contains both name and maps: %s", secret.Name)
		}

		secretClient, err := e.secretsStore.NewClient(e.logger, secretstore.Provider(secret.Provider))
		if err != nil {
			e.logger.Error("Unable to create new secret client", "provider", secret.Provider, "error", err)
			return secrets, fmt.Errorf("unable to create new secret client: %w", err)
		}

		s, err := secretClient.Get(secret.Path)
		if err != nil {
			if secret.Optional {
				e.logger.Warn("Secret is optional and not found", "provider", secret.Provider, "path", secret.Path)
				continue
			}

			e.logger.Error("Unable to get secret", "provider", secret.Provider, "path", secret.Path, "error", err)
			return secrets, fmt.Errorf("unable to get secret %s from provider: %s", secret.Path, secret.Provider)
		}

		if len(secret.Maps) == 0 {
			if secret.Name == "" {
				e.logger.Error("Secret does not contain name or maps", "provider", secret.Provider, "path", secret.Path)
				return nil, fmt.Errorf("secret does not contain name or maps: %s", secret.Path)
			}

			secrets = append(secrets, EarthlySecret{
				Id:    secret.Name,
				Value: s,
			})
		} else {
			var secretValues map[string]interface{}
			if err := json.Unmarshal([]byte(s), &secretValues); err != nil {
				e.logger.Error("Failed to unmarshal secret values", "provider", secret.Provider, "path", secret.Path, "error", err)
				return nil, fmt.Errorf("failed to unmarshal secret values from provider %s: %w", secret.Provider, err)
			}

			for sk, eid := range secret.Maps {
				if _, ok := secretValues[sk]; !ok {
					e.logger.Error("Secret key not found in secret values", "key", sk, "provider", secret.Provider, "path", secret.Path)
					return nil, fmt.Errorf("secret key not found in secret values: %s", sk)
				}

				s := EarthlySecret{
					Id: eid,
				}

				switch t := secretValues[sk].(type) {
				case bool:
					s.Value = strconv.FormatBool(t)
				case int:
					s.Value = strconv.FormatInt(int64(t), 10)
				default:
					s.Value = t.(string)
				}

				secrets = append(secrets, s)
			}
		}
	}

	return secrets, nil
}

// NewEarthlyExecutor creates a new EarthlyExecutor.
func NewEarthlyExecutor(
	earthfile, target string,
	executor executor.Executor,
	store secretstore.SecretStore,
	logger *slog.Logger,
	opts ...EarthlyExecutorOption,
) EarthlyExecutor {
	e := EarthlyExecutor{
		earthfile:    earthfile,
		executor:     executor,
		logger:       logger,
		secretsStore: store,
		target:       target,
		opts:         earthlyExecutorOptions{},
	}

	for _, opt := range opts {
		opt(&e)
	}

	return e
}
