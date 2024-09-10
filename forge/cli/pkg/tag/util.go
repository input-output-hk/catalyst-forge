package tag

import (
	"fmt"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GetTag returns the tag of the given reference if it exists. If multiple tags
// point to the same commit, the first one found is returned.
func GetTag(repo *gg.Repository, ref *plumbing.Reference) (string, error) {
	tags, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	var tag string
	err = tags.ForEach(func(t *plumbing.Reference) error {
		// Only process annotated tags
		tobj, err := repo.TagObject(t.Hash())
		if err != nil {
			return nil
		}

		fmt.Printf("Processing tag: %s\n", tobj.Name)

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
