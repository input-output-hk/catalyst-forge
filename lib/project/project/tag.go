package project

import (
	"errors"
	"strings"
)

var (
	ErrNotAProjectTag = errors.New("git tag is not a project tag")
)

// ProjectTag represents a project tag.
type ProjectTag struct {
	// Full is the full tag.
	Full string

	// Project is the project name.
	Project string

	// Version is the project version.
	Version string
}

// ParseProjectTag parses a project tag from a git tag.
func ParseProjectTag(tag string) (ProjectTag, error) {
	parts := strings.Split(tag, "/")
	if len(parts) != 2 {
		return ProjectTag{}, ErrNotAProjectTag
	}

	return ProjectTag{
		Full:    tag,
		Project: parts[0],
		Version: parts[1],
	}, nil
}
