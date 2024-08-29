package walker

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"testing"

	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
	"github.com/spf13/afero"
)

type wrapfs struct {
	afero.Fs
	trigger error
}

func (w *wrapfs) Open(name string) (afero.File, error) {
	return nil, w.trigger
}

func TestFileSystemWalkerWalk(t *testing.T) {
	tests := []struct {
		name          string
		fs            afero.Fs
		callbackErr   error
		path          string
		files         map[string]string
		expectedFiles map[string]string
		expectErr     bool
		expectedErr   error
	}{
		{
			name:        "single directory",
			fs:          afero.NewMemMapFs(),
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
			expectedErr: nil,
		},
		{
			name:        "nested directories",
			fs:          afero.NewMemMapFs(),
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
			expectedErr: nil,
		},
		{
			name: "error opening file",
			fs: &wrapfs{
				Fs:      afero.NewMemMapFs(),
				trigger: fmt.Errorf("fail"),
			},
			callbackErr: nil,
			path:        "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     true,
			expectedErr:   fmt.Errorf("fail"),
		},
		{
			name:        "callback error",
			fs:          afero.NewMemMapFs(),
			callbackErr: fmt.Errorf("callback error"),
			path:        "/test1",
			files: map[string]string{
				"/test1/file1": "file1",
			},
			expectedFiles: map[string]string{},
			expectErr:     true,
			expectedErr:   fmt.Errorf("callback error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := FilesystemWalker{
				fs:     tt.fs,
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			for path, content := range tt.files {
				dir := filepath.Dir(path)
				if err := tt.fs.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("failed to create directory %s: %v", dir, err)
				}

				if err := afero.WriteFile(tt.fs, path, []byte(content), 0644); err != nil {
					t.Fatalf("failed to write file %s: %v", path, err)
				}
			}

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

			ret, err := testutils.CheckError(t, err, tt.expectErr, tt.expectedErr)
			if err != nil {
				t.Fatal(err)
			} else if ret {
				return
			}

			if !maps.Equal(callbackFiles, tt.expectedFiles) {
				t.Fatalf("expected: %v, got: %v", tt.expectedFiles, callbackFiles)
			}
		})
	}
}
