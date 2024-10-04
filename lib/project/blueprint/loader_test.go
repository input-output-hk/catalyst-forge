package blueprint

import (
	"io"
	"io/fs"
	"log/slog"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/injector"
	imocks "github.com/input-output-hk/catalyst-forge/lib/project/injector/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	defaultInjector := func() injector.Injector {
		return injector.NewInjector(
			slog.New(slog.NewTextHandler(io.Discard, nil)),
			&imocks.EnvGetterMock{
				GetFunc: func(name string) (string, bool) {
					return "", false
				},
			},
		)
	}

	tests := []struct {
		name      string
		fs        afero.Fs
		injector  injector.Injector
		overrider InjectorOverrider
		project   string
		gitRoot   string
		files     map[string]string
		cond      func(*testing.T, cue.Value)
		expectErr bool
	}{
		{
			name:      "no files",
			fs:        afero.NewMemMapFs(),
			injector:  defaultInjector(),
			overrider: nil,
			project:   "/tmp/dir1/dir2",
			gitRoot:   "/tmp/dir1/dir2",
			files:     map[string]string{},
			cond: func(t *testing.T, v cue.Value) {
				assert.NoError(t, v.Err())
				assert.NotEmpty(t, v.LookupPath(cue.ParsePath("version")))
			},
			expectErr: false,
		},
		{
			name:      "single file",
			fs:        afero.NewMemMapFs(),
			injector:  defaultInjector(),
			overrider: nil,
			project:   "/tmp/dir1/dir2",
			gitRoot:   "/tmp/dir1/dir2",
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
			cond: func(t *testing.T, v cue.Value) {
				assert.NoError(t, v.Err())

				field, err := v.LookupPath(cue.ParsePath("project.ci.targets.test.privileged")).Bool()
				require.NoError(t, err)
				assert.Equal(t, true, field)
			},
			expectErr: false,
		},
		{
			name:      "multiple files",
			fs:        afero.NewMemMapFs(),
			injector:  defaultInjector(),
			overrider: nil,
			project:   "/tmp/dir1/dir2",
			gitRoot:   "/tmp/dir1",
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
			},
			cond: func(t *testing.T, v cue.Value) {
				assert.NoError(t, v.Err())

				field1, err := v.LookupPath(cue.ParsePath("project.ci.targets.test.privileged")).Bool()
				require.NoError(t, err)
				assert.Equal(t, true, field1)

				field2, err := v.LookupPath(cue.ParsePath("project.ci.targets.test.retries")).Int64()
				require.NoError(t, err)
				assert.Equal(t, int64(3), field2)
			},
			expectErr: false,
		},
		{
			name: "with injection",
			fs:   afero.NewMemMapFs(),
			injector: injector.NewInjector(
				slog.New(slog.NewTextHandler(io.Discard, nil)),
				&imocks.EnvGetterMock{
					GetFunc: func(name string) (string, bool) {
						if name == "RETRIES" {
							return "5", true
						}

						return "", false
					},
				},
			),
			overrider: nil,
			project:   "/tmp/dir1/dir2",
			gitRoot:   "/tmp/dir1/dir2",
			files: map[string]string{
				"/tmp/dir1/dir2/blueprint.cue": `
				version: "1.0"
				project: {
					name: "test"
					ci: {
						targets: {
							test: {
								retries: _ @env(name=RETRIES,type=int)
							}
						}
					}
				}
				`,
				"/tmp/dir1/.git": "",
			},
			cond: func(t *testing.T, v cue.Value) {
				assert.NoError(t, v.Err())

				field, err := v.LookupPath(cue.ParsePath("project.ci.targets.test.retries")).Int64()
				require.NoError(t, err)
				assert.Equal(t, int64(5), field)
			},
			expectErr: false,
		},
		{
			name:     "with injection overrides",
			fs:       afero.NewMemMapFs(),
			injector: defaultInjector(),
			overrider: func(bp cue.Value) map[string]string {
				return map[string]string{
					"RETRIES": "5",
				}
			},
			project: "/tmp/dir1/dir2",
			gitRoot: "/tmp/dir1/dir2",
			files: map[string]string{
				"/tmp/dir1/dir2/blueprint.cue": `
				version: "1.0"
				project: {
					name: "test"
					ci: {
						targets: {
							test: {
								retries: _ @env(name=RETRIES,type=int)
							}
						}
					}
				}
				`,
				"/tmp/dir1/.git": "",
			},
			cond: func(t *testing.T, v cue.Value) {
				assert.NoError(t, v.Err())

				field, err := v.LookupPath(cue.ParsePath("project.ci.targets.test.retries")).Int64()
				require.NoError(t, err)
				assert.Equal(t, int64(5), field)
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutils.SetupFS(t, tt.fs, tt.files)

			loader := DefaultBlueprintLoader{
				fs:        tt.fs,
				injector:  tt.injector,
				logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
				overrider: tt.overrider,
			}

			bp, err := loader.Load(tt.project, tt.gitRoot)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			tt.cond(t, bp.Value())
		})
	}
}
