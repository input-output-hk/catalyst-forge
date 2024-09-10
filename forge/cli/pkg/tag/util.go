package tag

import (
	"fmt"
	"strings"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

var (
	ErrNotMonoTag = fmt.Errorf("tag is not a monorepo tag")
)

type MonoTag struct {
	Project string
	Tag     string
}

// GetTag returns the tag of the current HEAD commit if it exists.
func GetTag(repo *gg.Repository) (string, error) {
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

	return tag, nil
}

func IsMonoTag(tag string) bool {
	_, err := ParseMonoTag(tag)
	return err == nil
}

func MatchMonoTag(project string, tag MonoTag) bool {
	return project == tag.Project
}

// ParseMonoTag parses a monorepo tag into its project and tag components.
func ParseMonoTag(tag string) (MonoTag, error) {
	parts := strings.Split(tag, "/")
	if len(parts) < 2 {
		return MonoTag{}, ErrNotMonoTag
	} else {
		return MonoTag{
			Project: strings.Join(parts[:len(parts)-1], "/"),
			Tag:     parts[len(parts)-1],
		}, nil
	}
}
