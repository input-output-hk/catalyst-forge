package blueprint

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"
	"github.com/Masterminds/semver/v3"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/injector"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/version"
	cuetools "github.com/input-output-hk/catalyst-forge/tools/pkg/cue"
)

// BlueprintFile represents a single blueprint file.
type BlueprintFile struct {
	Path    string
	Value   cue.Value
	Version *semver.Version
}

// BlueprintFiles represents a collection of blueprint files.
type BlueprintFiles []BlueprintFile

// Unify unifies the blueprints into a single CUE value. If the unification
// fails, an error is returned.
func (b BlueprintFiles) Unify(ctx *cue.Context) (cue.Value, error) {
	v := ctx.CompileString("{}")
	for _, bp := range b {
		v = v.Unify(bp.Value)
	}

	if err := cuetools.Validate(v, cue.Concrete(true)); err != nil {
		return cue.Value{}, err
	}

	return v, nil
}

// validateMajors validates the major versions of the blueprints. If the
// blueprints have different major versions, an error is returned.
func (b BlueprintFiles) ValidateMajorVersions() error {
	var last *semver.Version
	for _, bp := range b {
		if last == nil {
			last = bp.Version
			continue
		}

		if bp.Version.Major() != last.Major() {
			return fmt.Errorf("blueprints have different major versions")
		}
	}

	return nil
}

// Version returns the highest version number from the blueprints.
// If there are no blueprints, nil is returned.
func (b BlueprintFiles) Version() *semver.Version {
	if len(b) == 0 {
		return nil
	}

	var versions []*semver.Version
	for _, bp := range b {
		versions = append(versions, bp.Version)
	}

	sort.Sort(semver.Collection(versions))
	return versions[len(versions)-1]
}

// NewBlueprintFile creates a new BlueprintFile from the given CUE context,
// path, and contents. The contents are compiled and validated, including
// injecting any necessary environment variables. Additionally, the version is
// extracted from the CUE value. If the version is not found or invalid, or the
// final CUE value is invalid, an error is returned.
func NewBlueprintFile(ctx *cue.Context, path string, contents []byte, inj injector.Injector) (BlueprintFile, error) {
	v, err := cuetools.Compile(ctx, contents)
	if err != nil {
		return BlueprintFile{}, fmt.Errorf("failed to compile CUE file: %w", err)
	}

	version, err := version.GetVersion(v)
	if err != nil {
		return BlueprintFile{}, fmt.Errorf("failed to get version: %w", err)
	}

	// Delete the version to avoid conflicts when merging blueprints.
	// This is safe as we have already extracted the version.
	v, err = cuetools.Delete(ctx, v, "version")
	if err != nil {
		return BlueprintFile{}, fmt.Errorf("failed to delete version from blueprint file: %w", err)
	}

	v = inj.InjectEnv(v)

	if err := cuetools.Validate(v, cue.Concrete(true)); err != nil {
		return BlueprintFile{}, err
	}

	return BlueprintFile{
		Path:    path,
		Value:   v,
		Version: version,
	}, nil
}
