package kcl

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/kcl/client"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
)

// KCLManifestGenerator is a ManifestGenerator that uses KCL.
type KCLManifestGenerator struct {
	client client.KCLClient
	logger *slog.Logger
}

func (g *KCLManifestGenerator) Generate(mod schema.DeploymentModule, instance, registry string) ([]byte, error) {
	container := fmt.Sprintf("oci://%s/%s?tag=%s", strings.TrimSuffix(registry, "/"), mod.Name, mod.Version)
	conf := client.KCLModuleConfig{
		InstanceName: instance,
		Namespace:    mod.Namespace,
		Values:       mod.Values,
	}

	out, err := g.client.Run(container, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to run KCL module: %w", err)
	}

	return []byte(out), nil
}

// NewKCLManifestGenerator creates a new KCL manifest generator.
func NewKCLManifestGenerator(logger *slog.Logger) *KCLManifestGenerator {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &KCLManifestGenerator{
		client: client.KPMClient{},
		logger: logger,
	}
}
