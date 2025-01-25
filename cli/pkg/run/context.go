package run

import (
	"log/slog"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/generator"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

// RunContext represents the context in which a CLI run is happening.
type RunContext struct {
	// CI is true if the run is happening in a CI environment.
	CI bool

	// DeploymentGenerator is the deployment generator to use for generating deployments.
	DeploymentGenerator generator.Generator

	// FSWalker is the walker to use for walking the filesystem.
	FSWalker walker.FSWalker

	// Local is true if the run is happening in a local environment.
	Local bool

	// Logger is the logger to use for logging.
	Logger *slog.Logger

	// ProjectLoader is the project loader to use for loading projects.
	ProjectLoader project.ProjectLoader

	// SecretStore is the secret store to use for fetching secrets.
	SecretStore secrets.SecretStore

	// Verbose is the verbosity level of the run.
	Verbose int
}
