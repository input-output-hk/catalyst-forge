package walker

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/input-output-hk/catalyst-forge/tools/pkg/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestFSReverseWalkerWalk(t *testing.T) {
	tests := []struct {
		name          string
		fs            afero.Fs
		callbackErr   error
		startPath     string
		endPath       string
		files         map[string]string
		expectedFiles map[string]string
		expectErr     bool
		expectedErr   string
	}{
		{
			name:        "single directory",
			fs:          afero.NewMemMapFs(),
			callbackErr: nil,
			startPath:   "/test1",
			endPath:     "/test1",
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
			name:        "multiple directories",
			fs:          afero.NewMemMapFs(),
			callbackErr: nil,
			startPath:   "/test1/test2",
			endPath:     "/test1",
			files: map[string]string{
				"/test1/file1":       "file1",
				"/test1/file2":       "file2",
				"/test1/test2/file3": "file3",
			},
			expectedFiles: map[string]string{
				"/test1/file1":       "file1",
				"/test1/file2":       "file2",
				"/test1/test2/file3": "file3",
			},
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "multiple scoped directories",
			fs:          afero.NewMemMapFs(),
			callbackErr: nil,
			startPath:   "/test1/test2",
			endPath:     "/",
			files: map[string]string{
				"/file0":             "file0",
				"/test0/file0":       "file0",
				"/test1/file1":       "file1",
				"/test1/file2":       "file2",
				"/test1/test2/file3": "file3",
				"/test1/test3/file4": "file4",
			},
			expectedFiles: map[string]string{
				"/file0":             "file0",
				"/test1/file1":       "file1",
				"/test1/file2":       "file2",
				"/test1/test2/file3": "file3",
			},
			expectErr:   false,
			expectedErr: "",
		},
		{
			name:        "error reading directory",
			fs:          &wrapfs{Fs: afero.NewMemMapFs(), failAfter: 1, trigger: fmt.Errorf("failed")},
			callbackErr: nil,
			startPath:   "/test1",
			endPath:     "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     true,
			expectedErr:   "failed to read directory: failed",
		},
		{
			name:        "error reading file",
			fs:          &wrapfs{Fs: afero.NewMemMapFs(), failAfter: 2, trigger: fmt.Errorf("failed")},
			callbackErr: nil,
			startPath:   "/test1",
			endPath:     "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     true,
			expectedErr:   "failed to open file: failed",
		},
		{
			name:        "callback error",
			fs:          afero.NewMemMapFs(),
			callbackErr: fmt.Errorf("callback error"),
			startPath:   "/test1",
			endPath:     "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     true,
			expectedErr:   "callback error",
		},
		{
			name:        "callback EOF",
			fs:          afero.NewMemMapFs(),
			callbackErr: io.EOF,
			startPath:   "/test1",
			endPath:     "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     false,
			expectedErr:   "",
		},
		{
			name:        "start path is not a subdirectory of end path",
			fs:          afero.NewMemMapFs(),
			callbackErr: nil,
			startPath:   "/test1",
			endPath:     "/test2",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     true,
			expectedErr:   "start path is not a subdirectory of end path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := FSReverseWalker{
				fs:     tt.fs,
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			testutils.SetupFS(t, tt.fs, tt.files)

			callbackFiles := make(map[string]string)
			err := walker.Walk(tt.startPath, tt.endPath, func(path string, fileType FileType, openFile func() (FileSeeker, error)) error {
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
