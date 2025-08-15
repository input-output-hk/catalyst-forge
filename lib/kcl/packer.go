package kcl

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/input-output-hk/catalyst-forge/lib/kcl/internal"
)

// PackOptions configures tar packing behavior.
type PackOptions struct {
	// ExcludePatterns lists patterns of files to exclude
	ExcludePatterns []string
	// IncludeHidden includes hidden files (starting with .)
	IncludeHidden bool
	// FollowSymlinks follows symbolic links
	FollowSymlinks bool
	// Timestamp to use for all files (for reproducibility)
	Timestamp time.Time
}

// DefaultPackOptions returns default packing options.
func DefaultPackOptions() PackOptions {
	return PackOptions{
		ExcludePatterns: []string{
			".git",
			".gitignore",
			"*.pyc",
			"__pycache__",
			".DS_Store",
			"*.swp",
			"*.swo",
			"*~",
			".vscode",
			".idea",
		},
		IncludeHidden:  false,
		FollowSymlinks: false,
		Timestamp:      time.Unix(0, 0), // Epoch for reproducibility
	}
}

// PackModule creates a deterministic tar archive from a module directory.
// Returns the tar bytes and the SHA256 checksum.
func PackModule(moduleRoot string, opts PackOptions) ([]byte, string, error) {
	// Verify module root exists and contains kcl.mod
	if !internal.DirExists(moduleRoot) {
		return nil, "", fmt.Errorf("module root does not exist: %s", moduleRoot)
	}

	kclModPath := filepath.Join(moduleRoot, "kcl.mod")
	if !internal.FileExists(kclModPath) {
		return nil, "", fmt.Errorf("kcl.mod not found in module root: %s", moduleRoot)
	}

	// Collect files to pack
	files, err := collectFiles(moduleRoot, opts)
	if err != nil {
		return nil, "", fmt.Errorf("failed to collect files: %w", err)
	}

	// Sort files for deterministic ordering
	sort.Strings(files)

	// Create tar archive
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	defer func() { _ = tw.Close() }()

	for _, file := range files {
		fullPath := filepath.Join(moduleRoot, file)

		// Get file info
		info, err := os.Lstat(fullPath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to stat %s: %w", file, err)
		}

		// Handle symlinks
		var link string
		if info.Mode()&os.ModeSymlink != 0 {
			if !opts.FollowSymlinks {
				link, err = os.Readlink(fullPath)
				if err != nil {
					return nil, "", fmt.Errorf("failed to read symlink %s: %w", file, err)
				}
			} else {
				// Follow the symlink
				info, err = os.Stat(fullPath)
				if err != nil {
					return nil, "", fmt.Errorf("failed to follow symlink %s: %w", file, err)
				}
			}
		}

		// Create tar header with deterministic values
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create header for %s: %w", file, err)
		}

		// Normalize header for reproducibility
		header.Name = file // Use relative path
		header.ModTime = opts.Timestamp
		header.AccessTime = opts.Timestamp
		header.ChangeTime = opts.Timestamp
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""

		// Clear system-specific fields
		header.Devmajor = 0
		header.Devminor = 0

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return nil, "", fmt.Errorf("failed to write header for %s: %w", file, err)
		}

		// Write file content (if regular file)
		if info.Mode().IsRegular() {
			content, err := os.ReadFile(fullPath)
			if err != nil {
				return nil, "", fmt.Errorf("failed to read %s: %w", file, err)
			}

			if _, err := tw.Write(content); err != nil {
				return nil, "", fmt.Errorf("failed to write content for %s: %w", file, err)
			}
		}
	}

	// Flush the tar writer
	if err := tw.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Calculate checksum
	tarBytes := buf.Bytes()
	checksum := internal.HashBytes(tarBytes)

	return tarBytes, checksum, nil
}

// collectFiles collects all files to be included in the tar archive.
func collectFiles(root string, opts PackOptions) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip root directory itself
		if relPath == "." {
			return nil
		}

		// Normalize path separators
		relPath = filepath.ToSlash(relPath)

		// Check exclusions
		if shouldExclude(relPath, info, opts) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Add directories and files
		if info.IsDir() {
			// Add trailing slash for directories
			files = append(files, relPath+"/")
		} else {
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}

// shouldExclude checks if a file should be excluded from the archive.
func shouldExclude(path string, _ os.FileInfo, opts PackOptions) bool {
	base := filepath.Base(path)

	// Check hidden files
	if !opts.IncludeHidden && strings.HasPrefix(base, ".") && base != "." {
		return true
	}

	// Check exclude patterns
	for _, pattern := range opts.ExcludePatterns {
		// Try exact match first
		if base == pattern {
			return true
		}

		// Try glob match
		matched, err := filepath.Match(pattern, base)
		if err == nil && matched {
			return true
		}

		// Check if any part of the path matches
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if part == pattern {
				return true
			}
			matched, err := filepath.Match(pattern, part)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// UnpackModule extracts a tar archive to a directory.
func UnpackModule(tarReader io.Reader, destDir string) error {
	return internal.ExtractSafe(tarReader, destDir)
}

// ValidateModuleStructure checks if a directory has valid KCL module structure.
func ValidateModuleStructure(moduleRoot string) error {
	// Check for kcl.mod
	kclModPath := filepath.Join(moduleRoot, "kcl.mod")
	if !internal.FileExists(kclModPath) {
		return fmt.Errorf("kcl.mod not found")
	}

	// Parse and validate kcl.mod content
	meta, err := readModuleMetadata(moduleRoot)
	if err != nil {
		return fmt.Errorf("invalid kcl.mod: %w", err)
	}
	if meta.Name == "" {
		return fmt.Errorf("invalid kcl.mod: name is required")
	}
	if meta.Version == "" {
		return fmt.Errorf("invalid kcl.mod: version is required")
	}

	return nil
}

// GetModuleEntry determines the entry point for a KCL module.
func GetModuleEntry(moduleRoot string, meta *ModuleMeta) (string, error) {
	// If meta specifies entry, use it
	if meta != nil && meta.Entry != "" {
		entryPath := filepath.Join(moduleRoot, meta.Entry)
		if internal.FileExists(entryPath) {
			return meta.Entry, nil
		}
		return "", fmt.Errorf("specified entry file not found: %s", meta.Entry)
	}

	// Common entry point patterns
	commonEntries := []string{
		"main.k",
		"index.k",
		"lib.k",
		"mod.k",
	}

	for _, entry := range commonEntries {
		entryPath := filepath.Join(moduleRoot, entry)
		if internal.FileExists(entryPath) {
			return entry, nil
		}
	}

	// Look for any .k file in root
	files, err := os.ReadDir(moduleRoot)
	if err != nil {
		return "", fmt.Errorf("failed to read module directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".k") {
			return file.Name(), nil
		}
	}

	return "", fmt.Errorf("no entry point found in module")
}
