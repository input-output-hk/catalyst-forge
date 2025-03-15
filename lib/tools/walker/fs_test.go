package walker

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestFileSystemWalkerWalk(t *testing.T) {
	tests := []struct {
		name          string
		callbackErr   error
		path          string
		files         map[string]string
		expectedFiles map[string]string
		expectErr     bool
		expectedErr   string
	}{
		{
			name:        "single directory",
			callbackErr: nil,
			path:        "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
				"/test1/file2": "file2",
			},
			expectedFiles: map[string]string{
				"/test1/file1": "file1",
				"/test1/file2": "file2",
			},
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "nested directories",
			callbackErr: nil,
			path:        "/test1",
			files: map[string]string{
				"/test1/file1":           "file1",
				"/test1/dir1/file2":      "file2",
				"/test1/dir1/dir2/file3": "file3",
			},
			expectedFiles: map[string]string{
				"/test1/file1":           "file1",
				"/test1/dir1/file2":      "file2",
				"/test1/dir1/dir2/file3": "file3",
			},
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "callback error",
			callbackErr: fmt.Errorf("callback error"),
			path:        "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     true,
			expectedErr:   "callback error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			walker := FSWalker{
				fs:     fs,
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			testutils.SetupFS(t, fs, tt.files)

			callbackFiles := make(map[string]string)
			err := walker.Walk(tt.path, func(path string, fileType FileType, openFile func() (FileSeeker, error)) error {
				if tt.callbackErr != nil {
					return tt.callbackErr
				}

				if fileType == FileTypeDir {
					return nil
				}

				file, err := openFile()
				if err != nil {
					return err
				}
				defer file.Close()

				content, err := io.ReadAll(file)
				if err != nil {
					return err
				}

				callbackFiles[path] = string(content)
				return nil
			})

			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}

			assert.Equal(t, tt.expectedFiles, callbackFiles)
		})
	}
}
