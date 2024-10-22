package command

import (
	"fmt"
	"os"
	"path/filepath"
)

// Finds all `Developer.md` files within the specified `rootPath`.
// If `rootPath` is nil, it defaults to the current working directory.
func DiscoverMarkdownFiles(rootPath *string) ([]string, error) {
	// Default to the current directory if rootPath is not provided
	var searchPath string
	if rootPath != nil && *rootPath != "" {
		searchPath = *rootPath
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		searchPath = cwd
	}

	var result []string

	err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %v", path, err)
		}

		if !info.IsDir() && info.Name() == "Developer.md" {
			result = append(result, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking through the directory: %v", err)
	}

	return result, nil
}
