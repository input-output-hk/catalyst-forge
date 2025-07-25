package run

import (
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

// RunContext represents the context in which a CLI run is happening.
type RunContext struct {
	// CI is true if the run is happening in a CI environment.
	CI bool

	// CueCtx is the CUE context to use for CUE operations.
	CueCtx *cue.Context

	// FSWalker is the walker to use for walking the filesystem.
	FSWalker walker.FSWalker

	// Local is true if the run is happening in a local environment.
	Local bool

	// Logger is the logger to use for logging.
	Logger *slog.Logger

	// ManifestGeneratorStore is the manifest generator store to use for storing manifest generators.
	ManifestGeneratorStore deployment.ManifestGeneratorStore

	// ProjectLoader is the project loader to use for loading projects.
	ProjectLoader project.ProjectLoader

	// SecretStore is the secret store to use for fetching secrets.
	SecretStore secrets.SecretStore

	// Verbose is the verbosity level of the run.
	Verbose int
}
