package loader

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/injector"
	imocks "github.com/input-output-hk/catalyst-forge/blueprint/pkg/injector/mocks"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/testutils"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
	wmocks "github.com/input-output-hk/catalyst-forge/tools/pkg/walker/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fieldTest struct {
	fieldPath  string
	fieldType  string
	fieldValue any
}

// MockFileSeeker is a mock implementation of the FileSeeker interface.
type MockFileSeeker struct {
	*strings.Reader
}

func (MockFileSeeker) Stat() (fs.FileInfo, error) {
	return MockFileInfo{}, nil
}

func (MockFileSeeker) Close() error {
	return nil
}

// MockFileInfo is a mock implementation of the fs.FileInfo interface.
type MockFileInfo struct {
	fs.FileInfo
}

func (MockFileInfo) Name() string {
	return "Earthfile"
}

// NewMockFileSeeker creates a new MockFileSeeker with the given content.
func NewMockFileSeeker(s string) MockFileSeeker {
	return MockFileSeeker{strings.NewReader(s)}
}

func TestBlueprintLoaderLoad(t *testing.T) {
	tests := []struct {
		name      string
		root      string
		files     map[string]string
		want      []fieldTest
		expectErr bool
	}{
		{
			name:  "no files",
			root:  "/tmp/dir1/dir2",
			files: map[string]string{},
			want: []fieldTest{
				{
					fieldPath:  "version",
					fieldType:  "string",
					fieldValue: "1.0.0", // TODO: This may change
				},
			},
			expectErr: false,
		},
		{
			name: "single file",
			root: "/tmp/dir1/dir2",
			files: map[string]string{
				"/tmp/dir1/dir2/blueprint.cue": `
				version: "1.0"
				project: {
					name: "test"
					ci: {
						targets: {
							test: {
								privileged: true
							}
						}
					}
				}
				`,
				"/tmp/dir1/.git": "",
			},
			want: []fieldTest{
				{
					fieldPath:  "project.ci.targets.test.privileged",
					fieldType:  "bool",
					fieldValue: true,
				},
			},
			expectErr: false,
		},
		{
			name: "multiple files",
			root: "/tmp/dir1/dir2",
			files: map[string]string{
				"/tmp/dir1/dir2/blueprint.cue": `
				version: "1.0"
				project: {
					name: "test"
					ci: {
						targets: {
							test: {
								privileged: true
							}
						}
					}
				}
				`,
				"/tmp/dir1/blueprint.cue": `
				version: "1.1"
				project: ci: {
					targets: {
						test: {
							retries: 3
						}
					}
				}
				`,
				"/tmp/dir1/.git": "",
			},
			want: []fieldTest{
				{
					fieldPath:  "version",
					fieldType:  "string",
					fieldValue: "1.1.0",
				},
				{
					fieldPath:  "project.ci.targets.test.privileged",
					fieldType:  "bool",
					fieldValue: true,
				},
				{
					fieldPath:  "project.ci.targets.test.retries",
					fieldType:  "int",
					fieldValue: int64(3),
				},
			},
			expectErr: false,
		},
		{
			name: "multiple files, no git root",
			root: "/tmp/dir1/dir2",
			files: map[string]string{
				"/tmp/dir1/dir2/blueprint.cue": `
				version: "1.0"
				project: {
					name: "test"
					ci: {
						targets: {
							test: {
								privileged: true
							}
						}
					}
				}
				`,
				"/tmp/dir1/blueprint.cue": `
				version: "1.0"
				project: ci: {
					targets: {
						test: {
							retries: 3
						}
					}
			    }
				`,
			},
			want: []fieldTest{
				{
					fieldPath:  "project.ci.targets.test.privileged",
					fieldType:  "bool",
					fieldValue: true,
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := &wmocks.ReverseWalkerMock{
				WalkFunc: func(startPath string, endPath string, callback walker.WalkerCallback) error {
					// True when there is no git root, so we simulate only searching for blueprint files in the root path.
					if startPath == endPath && len(tt.files) > 0 {
						err := callback(filepath.Join(tt.root, "blueprint.cue"), walker.FileTypeFile, func() (walker.FileSeeker, error) {
							return NewMockFileSeeker(tt.files[filepath.Join(tt.root, "blueprint.cue")]), nil
						})

						if err != nil {
							return err
						}

						return nil
					} else if startPath == endPath && len(tt.files) == 0 {
						return nil
					}

					for path, content := range tt.files {
						var err error
						if content == "" {
							err = callback(path, walker.FileTypeDir, func() (walker.FileSeeker, error) {
								return nil, nil
							})
						} else {
							err = callback(path, walker.FileTypeFile, func() (walker.FileSeeker, error) {
								return NewMockFileSeeker(content), nil
							})
						}

						if errors.Is(err, io.EOF) {
							return nil
						} else if err != nil {
							return err
						}
					}

					return nil
				},
			}

			loader := DefaultBlueprintLoader{
				injector: injector.NewInjector(
					slog.New(slog.NewTextHandler(io.Discard, nil)),
					&imocks.EnvGetterMock{
						GetFunc: func(name string) (string, bool) {
							return "", false
						},
					},
				),
				logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
				rootPath: tt.root,
				walker:   walker,
			}

			bp, err := loader.Load()
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			for _, test := range tt.want {
				value := bp.Get(test.fieldPath)
				assert.Nil(t, value.Err(), "failed to lookup field %s: %v", test.fieldPath, value.Err())

				switch test.fieldType {
				case "bool":
					b, err := value.Bool()
					require.NoError(t, err, "failed to get bool value: %v", err)
					assert.Equal(t, b, test.fieldValue.(bool))
				case "int":
					i, err := value.Int64()
					require.NoError(t, err, "failed to get int value: %v", err)
					assert.Equal(t, i, test.fieldValue.(int64))
				case "string":
					s, err := value.String()
					require.NoError(t, err, "failed to get string value: %v", err)
					assert.Equal(t, s, test.fieldValue.(string))
				}
			}
		})
	}
}

func TestBlueprintLoader_findBlueprints(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		walkErr   error
		want      map[string][]byte
		expectErr bool
	}{
		{
			name: "simple",
			files: map[string]string{
				"/tmp/test1/test2/blueprint.cue": "test1",
				"/tmp/test1/foo.bar":             "foobar",
				"/tmp/test1/blueprint.cue":       "test2",
				"/tmp/blueprint.cue":             "test3",
			},
			want: map[string][]byte{
				"/tmp/test1/test2/blueprint.cue": []byte("test1"),
				"/tmp/test1/blueprint.cue":       []byte("test2"),
				"/tmp/blueprint.cue":             []byte("test3"),
			},
			expectErr: false,
		},
		{
			name: "no files",
			files: map[string]string{
				"/tmp/test1/foo.bar": "foobar",
			},
			want:      map[string][]byte{},
			expectErr: false,
		},
		{
			name: "error",
			files: map[string]string{
				"/tmp/test1/foo.bar": "foobar",
			},
			walkErr:   errors.New("error"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := &wmocks.ReverseWalkerMock{
				WalkFunc: func(startPath string, endPath string, callback walker.WalkerCallback) error {
					for path, content := range tt.files {
						err := callback(path, walker.FileTypeFile, func() (walker.FileSeeker, error) {
							return NewMockFileSeeker(content), nil
						})

						if err != nil {
							return err
						}
					}
					return tt.walkErr
				},
			}

			loader := DefaultBlueprintLoader{
				walker: walker,
			}
			got, err := loader.findBlueprints("/tmp", "/tmp")
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			for k, v := range got {
				require.Contains(t, tt.want, k)
				assert.Equal(t, tt.want[k], v)
			}
		})
	}
}

func TestBlueprintLoader_findGitRoot(t *testing.T) {
	tests := []struct {
		name      string
		start     string
		dirs      []string
		want      string
		expectErr bool
	}{
		{
			name:  "simple",
			start: "/tmp/test1/test2/test3",
			dirs: []string{
				"/tmp/test1/test2",
				"/tmp/test1",
				"/tmp/.git",
				"/",
			},
			want:      "/tmp",
			expectErr: false,
		},
		{
			name:  "no git root",
			start: "/tmp/test1/test2/test3",
			dirs: []string{
				"/tmp/test1/test2",
				"/tmp/test1",
				"/",
			},
			want:      "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lastPath string
			walker := &wmocks.ReverseWalkerMock{
				WalkFunc: func(startPath string, endPath string, callback walker.WalkerCallback) error {
					for _, dir := range tt.dirs {
						err := callback(dir, walker.FileTypeDir, func() (walker.FileSeeker, error) {
							return nil, nil
						})

						if errors.Is(err, io.EOF) {
							lastPath = dir
							return nil
						} else if err != nil {
							return err
						}
					}
					return nil
				},
			}

			loader := DefaultBlueprintLoader{
				walker: walker,
			}
			got, err := loader.findGitRoot(tt.start)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, lastPath, filepath.Join(tt.want, ".git"))
		})
	}
}
