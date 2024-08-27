package loader

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"testing"

	"github.com/input-output-hk/catalyst-forge/blueprint/internal/testutils"
	"github.com/spf13/afero"
)

type wrapfs struct {
	afero.Fs

	attempts  int
	failAfter int
	trigger   error
}

func (w *wrapfs) Open(name string) (afero.File, error) {
	w.attempts++
	if w.attempts == w.failAfter {
		return nil, w.trigger
	}
	return afero.Fs.Open(w.Fs, name)
}

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
		expectedErr   error
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
			expectedErr: nil,
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
			expectedErr: nil,
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
			expectedErr: nil,
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
			expectedErr:   fmt.Errorf("failed to read directory: failed"),
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
			expectedErr:   fmt.Errorf("failed to open file: failed"),
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
			expectedErr:   fmt.Errorf("callback error"),
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
			expectedErr:   nil,
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
			expectedErr:   fmt.Errorf("start path is not a subdirectory of end path"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := FSReverseWalker{
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
