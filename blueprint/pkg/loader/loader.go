package loader

import (
	"fmt"
	"io"
	"log/slog"
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
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
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
	injector injector.Injector
	logger   *slog.Logger
	walker   walker.ReverseWalker
}

func (b *DefaultBlueprintLoader) Load(projectPath, gitRootPath string) (blueprint.RawBlueprint, error) {
	files, err := b.findBlueprints(projectPath, gitRootPath)
	if err != nil {
		b.logger.Error("Failed to find blueprint files", "error", err)
		return blueprint.RawBlueprint{}, fmt.Errorf("failed to find blueprint files: %w", err)
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

// findBlueprints searches for blueprint files starting from the startPath and
// ending at the endPath. It returns a map of blueprint file paths to their
// contents or an error if the search fails.
func (b *DefaultBlueprintLoader) findBlueprints(startPath, endPath string) (map[string][]byte, error) {
	bps := make(map[string][]byte)

	err := b.walker.Walk(
		startPath,
		endPath,
		func(path string, fileType walker.FileType, openFile func() (walker.FileSeeker, error)) error {
			if fileType == walker.FileTypeFile {
				if filepath.Base(path) == BlueprintFileName {
					reader, err := openFile()
					if err != nil {
						return err
					}

					defer reader.Close()

					data, err := io.ReadAll(reader)
					if err != nil {
						return err
					}

					bps[path] = data
				}
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return bps, nil
}

// NewDefaultBlueprintLoader creates a new DefaultBlueprintLoader.
func NewDefaultBlueprintLoader(logger *slog.Logger) DefaultBlueprintLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	walker := walker.NewDefaultFSReverseWalker(logger)
	return DefaultBlueprintLoader{
		injector: injector.NewDefaultInjector(logger),
		logger:   logger,
		walker:   &walker,
	}
}
