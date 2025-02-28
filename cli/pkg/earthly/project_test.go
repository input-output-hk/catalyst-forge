package earthly

import (
	"log/slog"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/internal/testutils"
	emocks "github.com/input-output-hk/catalyst-forge/cli/pkg/executor/mocks"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/run"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	smocks "github.com/input-output-hk/catalyst-forge/lib/project/secrets/mocks"
	schema "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_generateOpts(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		src         string
		projectPath string
		ctx         run.RunContext
		err         bool
		validate    func(t *testing.T, ee EarthlyExecutor)
	}{
		{
			name:   "full",
			target: "target",
			src: `
			{
				global: ci: {
					providers: earthly: satellite: "sat"
					secrets: [
						{
							name: "bar"
							provider: "mock"
							path: "baz"
						}
					]
				}
				project: ci: targets: {
					nontarget: {
						args: {
							bar: "baz"
						}
					}
					target: {
						args: {
							foo: "bar"
						}
						platforms: ["linux/amd64"]
						privileged: true
						retries: 3
						secrets: [
							{
								name: "foo"
								provider: "mock"
								path: "bar"
							}
						]
					}
				}
			}`,
			ctx: run.RunContext{
				CI: true,
			},
			validate: func(t *testing.T, ee EarthlyExecutor) {
				assert.Contains(t, ee.targetArgs, "--foo")
				assert.Contains(t, ee.targetArgs, "bar")
				assert.NotContains(t, ee.targetArgs, "--bar")
				assert.NotContains(t, ee.targetArgs, "baz")
				assert.Contains(t, ee.opts.platforms, "linux/amd64")
				assert.Contains(t, ee.earthlyArgs, "--allow-privileged")
				assert.Equal(t, 3, ee.opts.retries)
				assert.Len(t, ee.secrets, 2)

				assert.Contains(t, ee.earthlyArgs, "--sat")
				assert.Contains(t, ee.earthlyArgs, "sat")
			},
		},
		{
			name:   "unified",
			target: "target",
			src: `
			{
				project: ci: targets: {
					".*": {
						args: {
							bar: "baz"
						}
					}
					"tar\\w+": {
						privileged: true
					}
					target: {
						args: {
							foo: "bar"
						}
					}
				}
			}`,
			ctx: run.RunContext{},
			validate: func(t *testing.T, ee EarthlyExecutor) {
				assert.Contains(t, ee.targetArgs, "--foo")
				assert.Contains(t, ee.targetArgs, "bar")
				assert.Contains(t, ee.targetArgs, "--bar")
				assert.Contains(t, ee.targetArgs, "baz")
				assert.Contains(t, ee.earthlyArgs, "--allow-privileged")
			},
		},
		{
			name:   "bad unified",
			target: "target",
			src: `
			{
				project: ci: targets: {
					".*": {
						platforms: ["linux/arm64"]
					}
					target: {
						platforms: ["linux/amd64"]
					}
				}
			}`,
			ctx:      run.RunContext{},
			err:      true,
			validate: func(t *testing.T, ee EarthlyExecutor) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cuecontext.New()
			rbp := ctx.CompileString(tt.src)
			require.NoError(t, rbp.Err())

			var bp schema.Blueprint
			require.NoError(t, rbp.Decode(&bp))

			executor := emocks.ExecutorMock{
				ExecuteFunc: func(command string, args ...string) ([]byte, error) {
					return nil, nil
				},
			}

			store := secrets.NewSecretStore(map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
				secrets.Provider("mock"): func(logger *slog.Logger) (secrets.SecretProvider, error) {
					return &smocks.SecretProviderMock{}, nil
				},
			})

			p := &DefaultProjectRunner{
				ctx:      tt.ctx,
				exectuor: &executor,
				logger:   testutils.NewNoopLogger(),
				project: &project.Project{
					Blueprint:    bp,
					RawBlueprint: blueprint.NewRawBlueprint(rbp),
				},
				store: store,
			}

			ee := EarthlyExecutor{}
			opts, err := p.generateOpts(tt.target)
			if !tt.err {
				require.NoError(t, err)
			}

			for _, opt := range opts {
				opt(&ee)
			}

			tt.validate(t, ee)
		})
	}
}
