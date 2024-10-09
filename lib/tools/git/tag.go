package git

import (
	"fmt"
	"os"
	"strings"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	ErrTagNotFound = fmt.Errorf("tag not found")
)

// MonoTag represents a monorepo git tag.
type MonoTag struct {
	// Full is the full tag string.
	Full string

	// Project is the project path.
	Project string

	// Tag is the tag name.
	Tag string
}

// Tag represents a git tag.
type Tag string

// IsMono returns true if the tag is a monorepo tag.
func (t Tag) IsMono() bool {
	parts := strings.Split(string(t), "/")
	if len(parts) < 2 {
		return false
	} else {
		return true
	}
}

// ToMono parses a monorepo tag into its project and tag components.
func (t Tag) ToMono() MonoTag {
	parts := strings.Split(string(t), "/")
	return MonoTag{
		Full:    string(t),
		Project: strings.Join(parts[:len(parts)-1], "/"),
		Tag:     parts[len(parts)-1],
	}
}

// GetTag returns the tag of the current HEAD commit.
func GetTag(repo *gg.Repository) (Tag, error) {
	var tag string
	var err error
	if InCI() {
		tag, err = getCITag()
		if err != nil {
			return "", err
		}
	} else {
		tag, err = getLocalTag(repo)
		if err != nil {
			return "", err
		}
	}

	return Tag(tag), nil
}

// getCITag returns the tag from the CI environment if it exists.
func getCITag() (string, error) {
	tag, exists := os.LookupEnv("GITHUB_REF")
	if exists && strings.HasPrefix(tag, "refs/tags/") {
		return strings.TrimPrefix(tag, "refs/tags/"), nil
	} else {
		return "", ErrTagNotFound
	}
}

// getLocalTag returns the tag of the current HEAD commit if it exists.
func getLocalTag(repo *gg.Repository) (string, error) {
	tags, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	ref, err := repo.Head()
	if err != nil {
		return "", err
	}

	var tag string
	err = tags.ForEach(func(t *plumbing.Reference) error {
		// Only process annotated tags
		tobj, err := repo.TagObject(t.Hash())
		if err != nil {
			return nil
		}

		if tobj.Target == ref.Hash() {
			tag = tobj.Name
			return nil
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to iterate over tags: %w", err)
	}

	if tag == "" {
		return "", ErrTagNotFound
	}

	return tag, nil
}
