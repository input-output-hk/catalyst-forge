package kcl

import (
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"github.com/BurntSushi/toml"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/kcl/client"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

// KCLModule represents a KCL module.
type KCLModule struct {
	Package KCLModulePackage `toml:"package"`
}

// KCLModulePackage represents a KCL module package.
type KCLModulePackage struct {
	Name    string `toml:"name"`
	Edition string `toml:"edition"`
	Version string `toml:"version"`
}

// KCLManifestGenerator is a ManifestGenerator that uses KCL.
type KCLManifestGenerator struct {
	client client.KCLClient
	fs     fs.Filesystem
	logger *slog.Logger
}

func (g *KCLManifestGenerator) Generate(mod sp.Module, raw cue.Value, env string) ([]byte, error) {
	var conf client.KCLModuleConfig
	var path string
	if mod.Path != "" {
		g.logger.Info("Parsing local KCL module", "path", mod.Path)
		kmod, err := g.parseModule(mod.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse KCL module: %w", err)
		}

		path = mod.Path
		conf = client.KCLModuleConfig{
			Env:       env,
			Instance:  mod.Instance,
			Name:      kmod.Package.Name,
			Namespace: mod.Namespace,
			Values:    mod.Values,
			Version:   kmod.Package.Version,
		}
	} else {
		path = fmt.Sprintf("oci://%s/%s?tag=%s", strings.TrimSuffix(mod.Registry, "/"), mod.Name, mod.Version)
		conf = client.KCLModuleConfig{
			Env:       env,
			Instance:  mod.Instance,
			Name:      mod.Name,
			Namespace: mod.Namespace,
			Values:    mod.Values,
			Version:   mod.Version,
		}
	}

	out, err := g.client.Run(path, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to run KCL module: %w", err)
	}

	return []byte(out), nil
}

// parseModule parses a KCL module from the given path.
func (g *KCLManifestGenerator) parseModule(path string) (KCLModule, error) {
	modPath := filepath.Join(path, "kcl.mod")
	exists, err := g.fs.Exists(modPath)
	if err != nil {
		return KCLModule{}, fmt.Errorf("failed to check if KCL module exists: %w", err)
	} else if !exists {
		return KCLModule{}, fmt.Errorf("KCL module not found")
	}

	src, err := g.fs.ReadFile(modPath)
	if err != nil {
		return KCLModule{}, fmt.Errorf("failed to read KCL module: %w", err)
	}

	var mod KCLModule
	_, err = toml.Decode(string(src), &mod)
	if err != nil {
		return KCLModule{}, fmt.Errorf("failed to decode KCL module: %w", err)
	}

	return mod, nil
}

// NewKCLManifestGenerator creates a new KCL manifest generator.
func NewKCLManifestGenerator(logger *slog.Logger) *KCLManifestGenerator {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &KCLManifestGenerator{
		client: client.KPMClient{},
		fs:     billy.NewBaseOsFS(),
		logger: logger,
	}
}
