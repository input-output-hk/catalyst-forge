package earthly

import (
	"strings"

	sc "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/common"
)

// WithArtifact is an option for configuring an EarthlyExecutor to output all
// artifacts contained within the given target to the given path.
func WithArtifact(path string) EarthlyExecutorOption {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	return func(e *EarthlyExecutor) {
		e.opts.artifact = path
	}
}

// WithCI is an option for configuring an EarthlyExecutor to run the CI
func WithCI() EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.opts.ci = true
	}
}

// WithConfig is an option for configuring an EarthlyExecutor to use the given
// Earthly config file.
func WithConfig(config string) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.earthlyArgs = append(e.earthlyArgs, "--config", config)
	}
}

// WithPlatforms is an option for configuring an EarthlyExecutor to run the
// Earthly target against the given platforms.
func WithPlatforms(platforms ...string) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.opts.platforms = append(e.opts.platforms, platforms...)
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

// WithSecrets is an option for configuring an EarthlyExecutor with secrets to
// be passed to the Earthly target.
func WithSecrets(secrets []sc.Secret) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.secrets = append(e.secrets, secrets...)
	}
}

// WithSkipOutput is an option for configuring an EarthlyExecutor to skip
// outputting any images or artifacts.
func WithSkipOutput() EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.earthlyArgs = append(e.earthlyArgs, "--no-output")
	}
}

// WithEarthlyArgs is an option for configuring an EarthlyExecutor with
// additional arguments that will be passed to the Earthly target.
func WithTargetArgs(args ...string) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.targetArgs = append(e.targetArgs, args...)
	}
}
