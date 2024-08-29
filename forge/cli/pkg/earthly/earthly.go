package earthly

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	secretstore "github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
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
	artifact bool
	retries  int
}

// EarthlyExecutor is an Executor that runs Earthly targets.
type EarthlyExecutor struct {
	logger       *slog.Logger
	opts         earthlyExecutorOptions
	executor     executor.Executor
	earthfile    string
	earthlyArgs  []string
	secrets      []schema.Secret
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
func (e EarthlyExecutor) Run() (EarthlyExecutionResult, error) {
	var (
		err     error
		secrets []EarthlySecret
	)

	if e.secrets != nil {
		secrets, err = e.buildSecrets()
		if err != nil {
			return EarthlyExecutionResult{}, err
		}

		var secretString []string
		for _, secret := range secrets {
			e.logger.Info("Adding Earthly secret", "earthly_id", secret.Id, "value", secret.Value)
			secretString = append(secretString, fmt.Sprintf("%s=%s", secret.Id, secret.Value))
		}

		if err := os.Setenv("EARTHLY_SECRETS", strings.Join(secretString, ",")); err != nil {
			e.logger.Error("Failed to set secret environment varibles", "envvar", "EARTHLY_SECRETS")
		}
	}

	var output []byte
	arguments := e.buildArguments()
	for i := 0; i < e.opts.retries+1; i++ {
		e.logger.Info("Executing Earthly", "attempt", i, "retries", e.opts.retries, "arguments", arguments)
		output, err = e.executor.Execute("earthly", arguments)
		if err == nil {
			break
		}

		e.logger.Error("Failed to run Earthly", "error", err)
	}

	if err != nil {
		e.logger.Error("Failed to run Earthly", "error", err)
		return EarthlyExecutionResult{}, fmt.Errorf("failed to run Earthly: %w", err)
	}

	return parseResult(string(output)), nil
}

// buildArguments constructs the arguments to pass to the Earthly target.
func (e *EarthlyExecutor) buildArguments() []string {
	var earthlyArgs []string

	earthlyArgs = append(earthlyArgs, e.earthlyArgs...)

	if e.opts.artifact {
		earthlyArgs = append(earthlyArgs, "--artifact", fmt.Sprintf("%s+%s/*", e.earthfile, e.target))
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
		secretClient, err := e.secretsStore.NewClient(e.logger, secretstore.Provider(secret.Provider))
		if err != nil {
			e.logger.Error("Unable to create new secret client", "provider", secret.Provider, "error", err)
			return secrets, fmt.Errorf("unable to create new secret client: %w", err)
		}

		s, err := secretClient.Get(secret.Path)
		if err != nil {
			e.logger.Error("Unable to get secret", "provider", secret.Provider, "path", secret.Path, "error", err)
			return secrets, fmt.Errorf("unable to get secret %s from provider: %s", secret.Path, secret.Provider)
		}

		var secretValues map[string]interface{}

		if err := json.Unmarshal([]byte(s), &secretValues); err != nil {
			e.logger.Error("Unable to unmarshal secret value", "provider", secret.Provider, "path", secret.Path, "error", err)
			return secrets, fmt.Errorf("unable to unmarshal secret value: %w", err)
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

// WithArtifact is an option for configuring an EarthlyExecutor to output all
// artifacts contained within the given target into the local working directory.
func WithArtifact() EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.opts.artifact = true
	}
}

// WithPrivileged is an option for configuring an EarthlyExecutor to run the
// Earthly target with elevated privileges.
func WithPrivileged() EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.earthlyArgs = append(e.earthlyArgs, "--allow-privileged")
	}
}

// WithRetries is an option for configuring an EarthlyExecutor with the number
// of retries to attempt if the Earthly target fails.
func WithRetries(retries int) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.opts.retries = retries
	}
}

// WithSatellite is an option for configuring an EarthlyExecutor with the
// remote satellite to use.
func WithSatellite(s string) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.earthlyArgs = append(e.earthlyArgs, "--sat", s)
	}
}

// WithEarthlyArgs is an option for configuring an EarthlyExecutor with
// additional arguments that will be passed to the Earthly target.
func WithTargetArgs(args ...string) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.targetArgs = args
	}
}

func WithSecrets(secrets []schema.Secret) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.secrets = secrets
	}
}

// parseResult parses the output of an Earthly execution and returns the
// resulting images and artifacts.
func parseResult(output string) EarthlyExecutionResult {
	images := make(map[string]string)
	artifacts := make(map[string]string)
	imageExpr := regexp.MustCompile(`^Image (.*?) output as (.*?)$`)
	artifactExpr := regexp.MustCompile(`Artifact (.*?) output as (.*?)$`)

	for _, line := range strings.Split(string(output), "\n") {
		if matches := imageExpr.FindStringSubmatch(line); matches != nil {
			images[matches[1]] = matches[2]
		}

		if matches := artifactExpr.FindStringSubmatch(line); matches != nil {
			artifacts[matches[1]] = matches[2]
		}
	}

	return EarthlyExecutionResult{
		Artifacts: artifacts,
		Images:    images,
	}
}
