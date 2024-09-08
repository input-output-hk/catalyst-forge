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

const BlueprintFileName = "blueprint.cue"

var (
	ErrGitRootNotFound = errors.New("git root not found")
	ErrVersionNotFound = errors.New("version not found")
)

// BlueprintLoader is an interface for loading blueprints.
type BlueprintLoader interface {

	// Load loads the blueprint.
	Load() blueprint.RawBlueprint
}

// DefaultBlueprintLoader is the default implementation of the BlueprintLoader
type DefaultBlueprintLoader struct {
	injector injector.Injector
	logger   *slog.Logger
	rootPath string
	walker   walker.ReverseWalker
}

func (b *DefaultBlueprintLoader) Load() (blueprint.RawBlueprint, error) {
	b.logger.Info("Searching for git root", "rootPath", b.rootPath)
	gitRoot, err := b.findGitRoot(b.rootPath)
	if err != nil && !errors.Is(err, ErrGitRootNotFound) {
		b.logger.Error("Failed to find git root", "error", err)
		return blueprint.RawBlueprint{}, fmt.Errorf("failed to find git root: %w", err)
	}

	var files map[string][]byte
	if errors.Is(err, ErrGitRootNotFound) {
		b.logger.Warn("Git root not found, searching for blueprint files in root path", "rootPath", b.rootPath)
		files, err = b.findBlueprints(b.rootPath, b.rootPath)
		if err != nil {
			b.logger.Error("Failed to find blueprint files", "error", err)
			return blueprint.RawBlueprint{}, fmt.Errorf("failed to find blueprint files: %w", err)
		}
	} else {
		b.logger.Info("Git root found, searching for blueprint files up to git root", "gitRoot", gitRoot)
		files, err = b.findBlueprints(b.rootPath, gitRoot)
		if err != nil {
			b.logger.Error("Failed to find blueprint files", "error", err)
			return blueprint.RawBlueprint{}, fmt.Errorf("failed to find blueprint files: %w", err)
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

// findGitRoot finds the root of a Git repository starting from the given
// path. It returns the path to the root of the Git repository or an error if
// the root is not found.
func (b *DefaultBlueprintLoader) findGitRoot(startPath string) (string, error) {
	var gitRoot string
	err := b.walker.Walk(
		startPath,
		"/",
		func(path string, fileType walker.FileType, openFile func() (walker.FileSeeker, error)) error {
			if fileType == walker.FileTypeDir {
				if filepath.Base(path) == ".git" {
					gitRoot = filepath.Dir(path)
					return io.EOF
				}
			}

			return nil
		},
	)

	if err != nil {
		return "", err
	}

	if gitRoot == "" {
		return "", ErrGitRootNotFound
	}

	return gitRoot, nil
}

// NewDefaultBlueprintLoader creates a new DefaultBlueprintLoader.
func NewDefaultBlueprintLoader(rootPath string,
	logger *slog.Logger,
) DefaultBlueprintLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	walker := walker.NewDefaultFSReverseWalker(logger)
	return DefaultBlueprintLoader{
		injector: injector.NewDefaultInjector(logger),
		logger:   logger,
		rootPath: rootPath,
		walker:   &walker,
	}
}
