package deployment

import (
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure -pkg mocks -out mocks/manifest.go . ManifestGenerator

// ManifestGenerator generates deployment manifests.
type ManifestGenerator interface {
	// Generate generates a deployment manifest for the given module.
	Generate(mod schema.DeploymentModule, instance, registry string) ([]byte, error)
}
