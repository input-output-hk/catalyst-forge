package loader

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/injector"
	imocks "github.com/input-output-hk/catalyst-forge/blueprint/pkg/injector/mocks"
	"github.com/input-output-hk/catalyst-forge/tools/pkg/walker"
	wmocks "github.com/input-output-hk/catalyst-forge/tools/pkg/walker/mocks"
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
		name    string
		root    string
		files   map[string]string
		want    []fieldTest
		wantErr bool
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

			loader := BlueprintLoader{
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

			err := loader.Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
				return
			}

			for _, test := range tt.want {
				value := loader.blueprint.Value().LookupPath(cue.ParsePath(test.fieldPath))
				if value.Err() != nil {
					t.Fatalf("failed to lookup field %s: %v", test.fieldPath, value.Err())
				}

				switch test.fieldType {
				case "bool":
					b, err := value.Bool()
					if err != nil {
						t.Fatalf("failed to get bool value: %v", err)
					}
					if b != test.fieldValue.(bool) {
						t.Errorf("for %v - got %v, want %v", test.fieldPath, b, test.fieldValue)
					}
				case "int":
					i, err := value.Int64()
					if err != nil {
						t.Fatalf("failed to get int value: %v", err)
					}
					if i != test.fieldValue.(int64) {
						t.Errorf("for %v - got %v, want %v", test.fieldPath, i, test.fieldValue)
					}
				case "string":
					s, err := value.String()
					if err != nil {
						t.Fatalf("failed to get string value: %v", err)
					}
					if s != test.fieldValue.(string) {
						t.Errorf("for %v - got %v, want %v", test.fieldPath, s, test.fieldValue)
					}
				}
			}
		})
	}
}

func TestBlueprintLoader_findBlueprints(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		walkErr error
		want    map[string][]byte
		wantErr bool
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
		},
		{
			name: "no files",
			files: map[string]string{
				"/tmp/test1/foo.bar": "foobar",
			},
			want:    map[string][]byte{},
			wantErr: false,
		},
		{
			name: "error",
			files: map[string]string{
				"/tmp/test1/foo.bar": "foobar",
			},
			walkErr: errors.New("error"),
			wantErr: true,
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

			loader := BlueprintLoader{
				walker: walker,
			}
			got, err := loader.findBlueprints("/tmp", "/tmp")
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
				return
			}

			for k, v := range got {
				if _, ok := tt.want[k]; !ok {
					t.Errorf("got unexpected key %v", k)
				}

				if !slices.Equal(v, tt.want[k]) {
					t.Errorf("got %s, want %s", string(v), string(tt.want[k]))
				}
			}
		})
	}
}

func TestBlueprintLoader_findGitRoot(t *testing.T) {
	tests := []struct {
		name    string
		start   string
		dirs    []string
		want    string
		wantErr bool
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
			want:    "/tmp",
			wantErr: false,
		},
		{
			name:  "no git root",
			start: "/tmp/test1/test2/test3",
			dirs: []string{
				"/tmp/test1/test2",
				"/tmp/test1",
				"/",
			},
			want:    "",
			wantErr: true,
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

			loader := BlueprintLoader{
				walker: walker,
			}
			got, err := loader.findGitRoot(tt.start)
			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
			if err == nil && lastPath != filepath.Join(tt.want, ".git") {
				t.Errorf("got last path %v, want %v", lastPath, tt.want)
			}
		})
	}
}
