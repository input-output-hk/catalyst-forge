package loader

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/blueprint"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/injector"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/version"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	cuetools "github.com/input-output-hk/catalyst-forge/tools/pkg/cue"
	"github.com/spf13/afero"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/loader.go . BlueprintLoader

const BlueprintFileName = "blueprint.cue"

var (
	ErrVersionNotFound = errors.New("version not found")
)

// BlueprintLoader is an interface for loading blueprints.
type BlueprintLoader interface {
	// Load loads the blueprint.
	Load(projectPath, gitRootPath string) (blueprint.RawBlueprint, error)
}

// DefaultBlueprintLoader is the default implementation of the BlueprintLoader
type DefaultBlueprintLoader struct {
	fs       afero.Fs
	injector injector.Injector
	logger   *slog.Logger
}

func (b *DefaultBlueprintLoader) Load(projectPath, gitRootPath string) (blueprint.RawBlueprint, error) {
	files := make(map[string][]byte)

	pbPath := filepath.Join(projectPath, BlueprintFileName)
	pb, err := afero.ReadFile(b.fs, pbPath)
	if err != nil {
		if os.IsNotExist(err) {
			b.logger.Warn("No project blueprint file found", "path", pbPath)
		} else {
			b.logger.Error("Failed to read blueprint file", "path", pbPath, "error", err)
			return blueprint.RawBlueprint{}, fmt.Errorf("failed to read blueprint file: %w", err)
		}
	} else {
		files[pbPath] = pb
	}

	if projectPath != gitRootPath {
		rootPath := filepath.Join(gitRootPath, BlueprintFileName)
		rb, err := afero.ReadFile(b.fs, rootPath)
		if err != nil {
			if os.IsNotExist(err) {
				b.logger.Warn("No root blueprint file found", "path", rootPath)
			} else {
				b.logger.Error("Failed to read blueprint file", "path", rootPath, "error", err)
				return blueprint.RawBlueprint{}, fmt.Errorf("failed to read blueprint file: %w", err)
			}
		} else {
			files[rootPath] = rb
		}
	}

	ctx := cuecontext.New()
	schema, err := schema.LoadSchema(ctx)
	if err != nil {
		b.logger.Error("Failed to load schema", "error", err)
		return blueprint.RawBlueprint{}, fmt.Errorf("failed to load schema: %w", err)
	}

	var finalBlueprint cue.Value
	var finalVersion *semver.Version
	var bps blueprint.BlueprintFiles
	if len(files) > 0 {
		for path, data := range files {
			b.logger.Info("Loading blueprint file", "path", path)
			bp, err := blueprint.NewBlueprintFile(ctx, path, data, b.injector)
			if err != nil {
				b.logger.Error("Failed to load blueprint file", "path", path, "error", err)
				return blueprint.RawBlueprint{}, fmt.Errorf("failed to load blueprint file: %w", err)
			}

			bps = append(bps, bp)
		}

		if err := bps.ValidateMajorVersions(); err != nil {
			b.logger.Error("Major version mismatch")
			return blueprint.RawBlueprint{}, err
		}

		userBlueprint, err := bps.Unify(ctx)
		if err != nil {
			b.logger.Error("Failed to unify blueprint files", "error", err)
			return blueprint.RawBlueprint{}, fmt.Errorf("failed to unify blueprint files: %w", err)
		}

		finalVersion = bps.Version()
		userBlueprint = userBlueprint.FillPath(cue.ParsePath("version"), finalVersion.String())
		userBlueprint = b.injector.InjectEnv(userBlueprint)
		finalBlueprint = schema.Unify(userBlueprint)
	} else {
		b.logger.Warn("No blueprint files found, using default values")
		finalVersion = schema.Version
		finalBlueprint = schema.Value.FillPath(cue.ParsePath("version"), finalVersion)
	}

	if err := cuetools.Validate(finalBlueprint, cue.Concrete(true)); err != nil {
		b.logger.Error("Failed to validate full blueprint", "error", err)
		return blueprint.RawBlueprint{}, err
	}

	if err := version.ValidateVersions(finalVersion, schema.Version); err != nil {
		if errors.Is(err, version.ErrMinorMismatch) {
			b.logger.Warn("The minor version of the blueprint is greater than the supported version", "version", finalVersion)
		} else {
			b.logger.Error("The major version of the blueprint is greater than the supported version", "version", finalVersion)
			return blueprint.RawBlueprint{}, fmt.Errorf("the major version of the blueprint (%s) is different than the supported version: cannot continue", finalVersion.String())
		}
	}

	return blueprint.NewRawBlueprint(finalBlueprint), nil
}

// NewDefaultBlueprintLoader creates a new DefaultBlueprintLoader.
func NewDefaultBlueprintLoader(logger *slog.Logger) DefaultBlueprintLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return DefaultBlueprintLoader{
		fs:       afero.NewOsFs(),
		injector: injector.NewDefaultInjector(logger),
		logger:   logger,
	}
}

func NewCustomBlueprintLoader(fs afero.Fs, injector injector.Injector, logger *slog.Logger) DefaultBlueprintLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return DefaultBlueprintLoader{
		fs:       fs,
		injector: injector,
		logger:   logger,
	}
}
