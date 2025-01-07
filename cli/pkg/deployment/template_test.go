package deployment

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor/mocks"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultBundleTemplaterRender(t *testing.T) {
	ctx := cuecontext.New()
	mkBundle := func(values cue.Value) Bundle {
		return Bundle{
			ApiVersion: "v1",
			Name:       "test",
			Instances: map[string]BundleInstance{
				"instance1": {
					Module: Module{
						Digest:  "test",
						Url:     "test",
						Version: "test",
					},
					Namespace: "test",
					Values:    values,
				},
			},
			ctx: ctx,
		}
	}
	tests := []struct {
		name     string
		bundle   Bundle
		stdout   string
		execFail bool
		validate func(*testing.T, string, []string, error)
	}{
		{
			name:     "full",
			bundle:   mkBundle(ctx.CompileString(`foo: "bar"`)),
			stdout:   "stdout",
			execFail: false,
			validate: func(t *testing.T, out string, calls []string, err error) {
				require.NoError(t, err)

				assert.Contains(t, calls, "bundle build --log-pretty=false --log-color=false -f tmp/bundle.cue")
				assert.Equal(t, "stdout", out)
			},
		},
		{
			name:     "fail encoding",
			bundle:   mkBundle(ctx.CompileString(`foo: doesnotexist`)),
			stdout:   "stdout",
			execFail: false,
			validate: func(t *testing.T, out string, calls []string, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:     "fail exec",
			bundle:   mkBundle(ctx.CompileString(`foo: "bar"`)),
			stdout:   "stdout",
			execFail: true,
			validate: func(t *testing.T, out string, calls []string, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			fs.MkdirAll("tmp", 0755)

			var calls []string
			e := mocks.WrappedExecuterMock{
				ExecuteFunc: func(args ...string) ([]byte, error) {
					calls = append(calls, strings.Join(args, " "))
					if tt.execFail {
						return nil, fmt.Errorf("exec failed")
					}

					return nil, nil
				},
			}

			templater := DefaultBundleTemplater{
				fs:      fs,
				logger:  testutils.NewNoopLogger(),
				timoni:  &e,
				stdout:  bytes.NewBuffer([]byte(tt.stdout)),
				workdir: "tmp",
			}

			out, err := templater.Render(tt.bundle)
			tt.validate(t, out, calls, err)
		})
	}
}
