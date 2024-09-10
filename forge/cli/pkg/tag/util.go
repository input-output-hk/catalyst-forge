package tag

import (
	"fmt"

	gg "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GetTag returns the tag of the given reference if it exists. If multiple tags
// point to the same commit, the first one found is returned.
func GetTag(repo *gg.Repository, ref *plumbing.Reference) (string, error) {
	fmt.Println("Getting tags")
	// Check if the HEAD is detached
	if ref.Name() == plumbing.HEAD {
		fmt.Println("Detached HEAD state")

		tobj, err := repo.TagObject(ref.Hash())
		if err != nil {
			fmt.Println("Failed to get tag object")
			return "", fmt.Errorf("failed to get tag object: %w", err)
		}

		return tobj.Name, nil
	}

	tags, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to get tags: %w", err)
	}

	var tag string
	err = tags.ForEach(func(t *plumbing.Reference) error {
		fmt.Printf("Processing tag: %s\n", t.Hash().String())
		// Only process annotated tags
		tobj, err := repo.TagObject(t.Hash())
		if err != nil {
			return nil
		}

		fmt.Printf("Processing annotated tag: %s\n", tobj.Name)

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
