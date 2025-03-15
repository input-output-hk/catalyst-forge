package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetAbs returns the absolute path of a given path.
func GetAbs(path string) (string, error) {
	var ap string
	var err error
	if !filepath.IsAbs(path) {
		ap, err = filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("could not get absolute path: %w", err)
		}
	} else {
		ap = path
	}

	return ap, nil
}

// Exists checks if a given path exists.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, fmt.Errorf("could not stat path: %w", err)
	}

	return true, nil
}
