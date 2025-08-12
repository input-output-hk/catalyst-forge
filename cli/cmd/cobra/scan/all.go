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
	"github.com/spf13/cobra"
)

// AllOptions holds the flags for the all scan command.
type AllOptions struct {
	Absolute       bool
	FilePattern    []string
	ContentPattern []string
	Pretty         bool
}

// NewAllCommand creates the scan all subcommand.
func NewAllCommand() *cobra.Command {
	opts := &AllOptions{}

	cmd := &cobra.Command{
		Use:   "all ROOTPATH",
		Short: "Scan for files matching filename and content patterns",
		Long:  `Scan filesystem for files matching specific patterns in names and content.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := run.MustFromContext(cmd.Context())
			return allExecute(ctx, args[0], opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Absolute, "absolute", "a", false, "Output absolute paths")
	cmd.Flags().StringSliceVarP(&opts.FilePattern, "file-pattern", "f", nil, "Regular expression patterns to filter filenames")
	cmd.Flags().StringSliceVarP(&opts.ContentPattern, "content-pattern", "c", nil, "Regular expression patterns to filter file contents")
	cmd.Flags().BoolVarP(&opts.Pretty, "pretty", "p", false, "Pretty print JSON output")

	return cmd
}

// allExecute executes the scan all command logic.
func allExecute(ctx run.RunContext, rootPath string, opts *AllOptions) error {
	var err error

	if opts.Absolute {
		rootPath, err = getAbsolutePath(rootPath)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
	}

	// Compile file patterns
	var filePatterns []*regexp.Regexp
	for _, pattern := range opts.FilePattern {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid file pattern %q: %w", pattern, err)
		}
		filePatterns = append(filePatterns, re)
	}

	// Compile content patterns
	var contentPatterns []*regexp.Regexp
	for _, pattern := range opts.ContentPattern {
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
		if !opts.Absolute && !strings.HasPrefix(rootPath, "/") && path != "." {
			outputPath = fmt.Sprintf("./%s", path)
		}

		results = append(results, outputPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	utils.PrintJson(results, opts.Pretty)
	return nil
}

// getAbsolutePath returns the absolute path of the given path.
func getAbsolutePath(path string) (string, error) {
	if path == "" {
		path = "."
	}
	return filepath.Abs(path)
}
