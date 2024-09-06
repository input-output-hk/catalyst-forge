package earthly

import (
	"strings"

	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
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

func WithCI() EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.opts.ci = true
		e.earthlyArgs = append(e.earthlyArgs, "--ci")
	}
}

// WithPlatform is an option for configuring an EarthlyExecutor to run the
// Earthly target with the given platform.
func WithPlatform(platform string) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.earthlyArgs = append(e.earthlyArgs, "--platform", platform)
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

// WithSecrets is an option for configuring an EarthlyExecutor with secrets to
// be passed to the Earthly target.
func WithSecrets(secrets []schema.Secret) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.secrets = secrets
	}
}

// WithEarthlyArgs is an option for configuring an EarthlyExecutor with
// additional arguments that will be passed to the Earthly target.
func WithTargetArgs(args ...string) EarthlyExecutorOption {
	return func(e *EarthlyExecutor) {
		e.targetArgs = args
	}
}
