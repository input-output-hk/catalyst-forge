package deployment

import (
	"cuelang.org/go/cue"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/manifest.go . ManifestGenerator

// ManifestGenerator generates deployment manifests.
type ManifestGenerator interface {
	// Generate generates a deployment manifest for the given module.
	Generate(mod sp.Module, raw cue.Value, env string) ([]byte, error)
}
