package blueprint

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint/defaults"
	s "github.com/input-output-hk/catalyst-forge/lib/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
)

//go:generate go run github.com/matryer/moq@latest --pkg mocks --out ./mocks/loader.go . BlueprintLoader

const BlueprintFileName = "blueprint.cue"

var (
	ErrVersionNotFound = errors.New("version not found")
)

// BlueprintLoader is an interface for loading blueprints.
type BlueprintLoader interface {
	// Load loads the blueprint.
	Load(projectPath, gitRootPath string) (RawBlueprint, error)
}

// DefaultBlueprintLoader is the default implementation of the BlueprintLoader
type DefaultBlueprintLoader struct {
	ctx    *cue.Context
	fs     fs.Filesystem
	logger *slog.Logger
}

func (b *DefaultBlueprintLoader) Load(projectPath, gitRootPath string) (RawBlueprint, error) {
	bpFiles := make(map[string][]byte)

	// Try to load the project blueprint file
	pbPath := filepath.Join(projectPath, BlueprintFileName)
	pb, err := b.fs.ReadFile(pbPath)
	if err != nil {
		if os.IsNotExist(err) {
			b.logger.Warn("No project blueprint file found", "path", pbPath)
		} else {
			b.logger.Error("Failed to read blueprint file", "path", pbPath, "error", err)
			return RawBlueprint{}, fmt.Errorf("failed to read blueprint file: %w", err)
		}
	} else {
		bpFiles[pbPath] = pb
	}

	// Try to load the root blueprint file
	rbPath := filepath.Join(gitRootPath, BlueprintFileName)
	if rbPath != pbPath {
		rb, err := b.fs.ReadFile(rbPath)
		if err != nil {
			if os.IsNotExist(err) {
				b.logger.Warn("No root blueprint file found", "path", rbPath)
			} else {
				b.logger.Error("Failed to read blueprint file", "path", rbPath, "error", err)
				return RawBlueprint{}, fmt.Errorf("failed to read blueprint file: %w", err)
			}
		} else {
			bpFiles[rbPath] = rb
		}
	}

	// If we have any files, unify them
	v := b.ctx.CompileString("{}")
	for path, data := range bpFiles {
		b.logger.Debug("Loading blueprint file", "path", path)
		bv := b.ctx.CompileBytes(data)
		if err := bv.Err(); err != nil {
			b.logger.Error("Failed to compile blueprint file", "path", path, "error", err)
			return RawBlueprint{}, fmt.Errorf("failed to compile blueprint file: %w", err)
		}

		v = v.Unify(bv)
	}

	if err := v.Err(); err != nil {
		b.logger.Error("Failed to unify blueprint files", "error", err)
		return RawBlueprint{}, fmt.Errorf("failed to unify blueprint files: %w", err)
	}

	// Unify the schema with the user-defined blueprint
	schema, err := s.LoadSchema(b.ctx)
	if err != nil {
		b.logger.Error("Failed to load schema", "error", err)
		return RawBlueprint{}, fmt.Errorf("failed to load schema: %w", err)
	}
	finalBlueprint := schema.Unify(v)

	// Apply default setters
	defaultSetters := defaults.GetDefaultSetters()
	for _, setter := range defaultSetters {
		var err error
		finalBlueprint, err = setter.SetDefault(finalBlueprint)
		if err != nil {
			b.logger.Error("Failed to set default values", "error", err)
			return RawBlueprint{}, fmt.Errorf("failed to set default values: %w", err)
		}
	}

	return NewRawBlueprint(finalBlueprint), nil
}

// NewDefaultBlueprintLoader creates a new DefaultBlueprintLoader.
func NewDefaultBlueprintLoader(ctx *cue.Context, logger *slog.Logger) DefaultBlueprintLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return DefaultBlueprintLoader{
		ctx:    ctx,
		fs:     billy.NewBaseOsFS(),
		logger: logger,
	}
}

// NewCustomBlueprintLoader creates a new DefaultBlueprintLoader with custom
// dependencies.
func NewCustomBlueprintLoader(
	ctx *cue.Context,
	fs fs.Filesystem,
	logger *slog.Logger,
) DefaultBlueprintLoader {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return DefaultBlueprintLoader{
		ctx:    ctx,
		fs:     fs,
		logger: logger,
	}
}
