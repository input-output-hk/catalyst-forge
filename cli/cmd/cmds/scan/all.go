package scan

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

type AllCmd struct {
	Absolute       bool     `short:"a" help:"Output absolute paths."`
	FilePattern    []string `short:"f" help:"Regular expression patterns to filter filenames."`
	ContentPattern []string `short:"c" help:"Regular expression patterns to filter file contents."`
	Pretty         bool     `short:"p" help:"Pretty print JSON output."`
	RootPath       string   `kong:"arg,predictor=path" help:"Root path to scan for files."`
}

func (c *AllCmd) Run(ctx run.RunContext) error {
	var err error
	rootPath := c.RootPath

	if c.Absolute {
		rootPath, err = getAbsolutePath(rootPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	// Compile file patterns
	var filePatterns []*regexp.Regexp
	for _, pattern := range c.FilePattern {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid file pattern %q: %w", pattern, err)
		}
		filePatterns = append(filePatterns, re)
	}

	// Compile content patterns
	var contentPatterns []*regexp.Regexp
	for _, pattern := range c.ContentPattern {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid content pattern %q: %w", pattern, err)
		}
		contentPatterns = append(contentPatterns, re)
	}

	// Scan files
	results := make([]string, 0)
	err = ctx.FSWalker.Walk(rootPath, func(path string, fileType walker.FileType, openFile func() (walker.FileSeeker, error)) error {
		if fileType != walker.FileTypeFile {
			return nil
		}

		// Check if filename matches all file patterns
		filename := filepath.Base(path)
		if len(filePatterns) > 0 {
			matches := true
			for _, pattern := range filePatterns {
				if !pattern.MatchString(filename) {
					matches = false
					break
				}
			}
			if !matches {
				return nil
			}
		}

		// Check if content matches all content patterns
		if len(contentPatterns) > 0 {
			file, err := openFile()
			if err != nil {
				ctx.Logger.Debug("Failed to open file", "path", path, "error", err)
				return nil // Skip files we can't read
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				ctx.Logger.Debug("Failed to read file", "path", path, "error", err)
				return nil // Skip files we can't read
			}

			matches := true
			for _, pattern := range contentPatterns {
				if !pattern.Match(content) {
					matches = false
					break
				}
			}
			if !matches {
				return nil
			}
		}

		// Normalize path for output
		outputPath := path
		if !c.Absolute && !strings.HasPrefix(rootPath, "/") && path != "." {
			outputPath = fmt.Sprintf("./%s", path)
		}

		results = append(results, outputPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	utils.PrintJson(results, c.Pretty)
	return nil
}
