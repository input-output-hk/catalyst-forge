package internal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SemVer represents a semantic version.
type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Metadata   string
}

var semverRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-\.]+))?(?:\+([0-9A-Za-z\-\.]+))?$`)

// ParseSemVer parses a semantic version string.
func ParseSemVer(version string) (*SemVer, error) {
	matches := semverRegex.FindStringSubmatch(version)
	if matches == nil {
		return nil, fmt.Errorf("invalid semver: %s", version)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &SemVer{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: matches[4],
		Metadata:   matches[5],
	}, nil
}

// String returns the string representation of the version.
func (v *SemVer) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	
	if v.Metadata != "" {
		s += "+" + v.Metadata
	}
	
	return s
}

// Compare compares two semantic versions.
// Returns -1 if v < other, 0 if v == other, 1 if v > other.
func (v *SemVer) Compare(other *SemVer) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Handle prerelease versions
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1 // Release version is greater than prerelease
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1 // Prerelease is less than release
	}
	if v.Prerelease != "" && other.Prerelease != "" {
		return strings.Compare(v.Prerelease, other.Prerelease)
	}

	return 0
}

// IsCompatible checks if this version is compatible with a constraint.
func (v *SemVer) IsCompatible(constraint string) (bool, error) {
	// Simple constraint parsing (can be extended)
	constraint = strings.TrimSpace(constraint)

	// Exact match
	if !strings.ContainsAny(constraint, "^~><=") {
		other, err := ParseSemVer(constraint)
		if err != nil {
			return false, err
		}
		return v.Compare(other) == 0, nil
	}

	// Caret constraint (^1.2.3 means >=1.2.3 <2.0.0)
	if strings.HasPrefix(constraint, "^") {
		base, err := ParseSemVer(constraint[1:])
		if err != nil {
			return false, err
		}

		if v.Major != base.Major {
			return v.Major > base.Major && base.Major == 0, nil
		}

		if v.Minor < base.Minor {
			return false, nil
		}

		if v.Minor == base.Minor && v.Patch < base.Patch {
			return false, nil
		}

		return true, nil
	}

	// Tilde constraint (~1.2.3 means >=1.2.3 <1.3.0)
	if strings.HasPrefix(constraint, "~") {
		base, err := ParseSemVer(constraint[1:])
		if err != nil {
			return false, err
		}

		if v.Major != base.Major || v.Minor != base.Minor {
			return false, nil
		}

		return v.Patch >= base.Patch, nil
	}

	// Range constraints (can be extended with more complex parsing)
	return false, fmt.Errorf("unsupported constraint: %s", constraint)
}

// Bump increases the version based on the bump type.
func (v *SemVer) Bump(bumpType string) *SemVer {
	newVer := &SemVer{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}

	switch bumpType {
	case "major":
		newVer.Major++
		newVer.Minor = 0
		newVer.Patch = 0
	case "minor":
		newVer.Minor++
		newVer.Patch = 0
	case "patch":
		newVer.Patch++
	}

	return newVer
}

// IsPrerelease returns true if this is a prerelease version.
func (v *SemVer) IsPrerelease() bool {
	return v.Prerelease != ""
}

// IsStable returns true if this is a stable release (not prerelease).
func (v *SemVer) IsStable() bool {
	return v.Prerelease == ""
}

// ValidateSemVer checks if a string is a valid semantic version.
func ValidateSemVer(version string) error {
	_, err := ParseSemVer(version)
	return err
}

// NormalizeSemVer normalizes a version string (adds/removes 'v' prefix as needed).
func NormalizeSemVer(version string, includeV bool) string {
	version = strings.TrimSpace(version)
	
	if includeV && !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	
	if !includeV && strings.HasPrefix(version, "v") {
		return version[1:]
	}
	
	return version
}

// ExtractSemVer attempts to extract a semver from a string.
func ExtractSemVer(text string) (*SemVer, error) {
	matches := semverRegex.FindStringSubmatch(text)
	if matches == nil {
		// Try to find semver pattern anywhere in the text
		pattern := regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-\.]+))?(?:\+([0-9A-Za-z\-\.]+))?`)
		matches = pattern.FindStringSubmatch(text)
		if matches == nil {
			return nil, fmt.Errorf("no semver found in text")
		}
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &SemVer{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: matches[4],
		Metadata:   matches[5],
	}, nil
}