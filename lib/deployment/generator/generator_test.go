package generator

import (
	"fmt"
	"log/slog"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/deployment"
	"github.com/input-output-hk/catalyst-forge/lib/deployment/mocks"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratorGenerateBundle(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name     string
		bundle   deployment.ModuleBundle
		env      cue.Value
		yaml     string
		err      bool
		validate func(t *testing.T, result GeneratorResult, err error)
	}{
		{
			name: "full",
			bundle: deployment.ModuleBundle{
				Bundle: sp.ModuleBundle{
					Env: "test",
					Modules: map[string]sp.Module{
						"test": sp.Module{
							Instance:  "instance",
							Name:      "test",
							Namespace: "default",
							Registry:  "registry",
							Type:      "kcl",
							Values:    ctx.CompileString(`foo: "bar"`),
							Version:   "1.0.0",
						},
					},
				},
			},
			env:  ctx.CompileString(`test: values: { bar: "baz" }`),
			yaml: "test",
			err:  false,
			validate: func(t *testing.T, result GeneratorResult, err error) {
				require.NoError(t, err)

				m := `{
	env: "test"
	modules: {
		test: {
			instance:  "instance"
			name:      "test"
			namespace: "default"
			registry:  "registry"
			type:      "kcl"
			values: {
				foo: "bar"
			}
			version: "1.0.0"
		}
	}
}`
				assert.Equal(t, m, string(result.Module))
				assert.Equal(t, "test", string(result.Manifests["test"]))
			},
		},
		{
			name: "manifest error",
			bundle: deployment.ModuleBundle{
				Bundle: sp.ModuleBundle{
					Env: "test",
					Modules: map[string]sp.Module{
						"test": sp.Module{
							Instance:  "instance",
							Name:      "test",
							Namespace: "default",
							Registry:  "registry",
							Type:      "kcl",
							Values:    ctx.CompileString(`foo: "bar"`),
							Version:   "1.0.0",
						},
					},
				},
			},
			yaml: "test",
			err:  true,
			validate: func(t *testing.T, result GeneratorResult, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := &mocks.ManifestGeneratorMock{
				GenerateFunc: func(mod sp.Module, raw cue.Value, env string) ([]byte, error) {
					if tt.err {
						return nil, fmt.Errorf("error")
					}

					return []byte(tt.yaml), nil
				},
			}

			store := deployment.NewManifestGeneratorStore(
				map[deployment.Provider]func(*slog.Logger) deployment.ManifestGenerator{
					deployment.ProviderKCL: func(logger *slog.Logger) deployment.ManifestGenerator {
						return mg
					},
				},
			)

			gen := Generator{
				logger: testutils.NewNoopLogger(),
				store:  store,
			}

			tt.bundle.Raw = getRawBundle(tt.bundle.Bundle)
			result, err := gen.GenerateBundle(tt.bundle, tt.env)
			tt.validate(t, result, err)
		})
	}
}

func TestGeneratorGenerate(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name     string
		module   sp.Module
		yaml     string
		env      string
		err      bool
		validate func(t *testing.T, result []byte, err error)
	}{
		{
			name: "full",
			module: sp.Module{
				Instance:  "instance",
				Name:      "test",
				Namespace: "default",
				Registry:  "registry",
				Type:      "kcl",
				Values:    ctx.CompileString(`foo: "bar"`),
				Version:   "1.0.0",
			},
			yaml: "test",
			env:  "test",
			err:  false,
			validate: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test", string(result))
			},
		},
		{
			name: "manifest error",
			module: sp.Module{
				Name:      "test",
				Namespace: "default",
				Type:      "kcl",
				Values:    ctx.CompileString(`foo: "bar"`),
				Version:   "1.0.0",
			},
			yaml: "test",
			env:  "test",
			err:  true,
			validate: func(t *testing.T, result []byte, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := &mocks.ManifestGeneratorMock{
				GenerateFunc: func(mod sp.Module, raw cue.Value, env string) ([]byte, error) {
					if tt.err {
						return nil, fmt.Errorf("error")
					}

					return []byte(tt.yaml), nil
				},
			}

			store := deployment.NewManifestGeneratorStore(
				map[deployment.Provider]func(*slog.Logger) deployment.ManifestGenerator{
					deployment.ProviderKCL: func(logger *slog.Logger) deployment.ManifestGenerator {
						return mg
					},
				},
			)

			gen := Generator{
				logger: testutils.NewNoopLogger(),
				store:  store,
			}

			result, err := gen.Generate(tt.module, getRawModule(tt.module), tt.env)
			tt.validate(t, result, err)
		})
	}
}

func getRawBundle(b sp.ModuleBundle) cue.Value {
	ctx := cuecontext.New()
	v := ctx.Encode(b)

	return v
}

func getRawModule(m sp.Module) cue.Value {
	ctx := cuecontext.New()
	v := ctx.Encode(m)

	return v
}
