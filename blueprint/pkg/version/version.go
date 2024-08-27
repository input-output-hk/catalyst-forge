package version

import (
	"errors"
	"fmt"

	"cuelang.org/go/cue"
	"github.com/Masterminds/semver/v3"
)

var (
	ErrMajorMismatch = errors.New("major version mismatch")
	ErrMinorMismatch = errors.New("minor version mismatch")
)

// getVersion extracts the version from the given CUE value. If the version is
// not found or invalid, an error is returned.
func GetVersion(v cue.Value) (*semver.Version, error) {
	cueVersion := v.LookupPath(cue.ParsePath("version"))
	if !cueVersion.Exists() || !cueVersion.IsConcrete() {
		return nil, fmt.Errorf("version not found")
	}

	strVersion, err := cueVersion.String()
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	return semver.NewVersion(strVersion)
}

// validateVersion validates the version of the given blueprint against the
// schema. If the blueprint major version is greater than the schema major
// version, an error is returned. If the blueprint minor version is greater than
// the schema minor version, an error is returned.
func ValidateVersions(blueprintVersion *semver.Version, schemaVersion *semver.Version) error {
	if blueprintVersion.Major() != schemaVersion.Major() {
		return ErrMajorMismatch
	} else if blueprintVersion.Minor() > schemaVersion.Minor() {
		return ErrMinorMismatch
	}

	return nil
}
