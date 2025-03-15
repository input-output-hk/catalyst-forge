package blueprint

import (
	"io"
	"io/fs"
	"log/slog"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
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
	tests := []struct {
		name      string
		project   string
		gitRoot   string
		files     map[string]string
		cond      func(*testing.T, cue.Value)
		expectErr bool
	}{
		{
			name:    "no files",
			project: "/tmp/dir1/dir2",
			gitRoot: "/tmp/dir1/dir2",
			files:   map[string]string{},
			cond: func(t *testing.T, v cue.Value) {
				assert.NoError(t, v.Err())
				assert.NotEmpty(t, v.LookupPath(cue.ParsePath("version")))
			},
			expectErr: false,
		},
		{
			name:    "single file",
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
			name:    "multiple files",
			project: "/tmp/dir1/dir2",
			gitRoot: "/tmp/dir1",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			testutils.SetupFS(t, fs, tt.files)

			loader := DefaultBlueprintLoader{
				ctx:    cuecontext.New(),
				fs:     fs,
				logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			}

			bp, err := loader.Load(tt.project, tt.gitRoot)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			tt.cond(t, bp.Value())
		})
	}
}
