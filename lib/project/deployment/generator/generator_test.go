package generator

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/utils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratorGenerateBundle(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name     string
		bundle   schema.DeploymentModuleBundle
		yaml     string
		err      bool
		validate func(t *testing.T, result GeneratorResult, err error)
	}{
		{
			name: "full",
			bundle: schema.DeploymentModuleBundle{
				"test": schema.DeploymentModule{
					Instance:  "instance",
					Name:      utils.StringPtr("test"),
					Namespace: "default",
					Registry:  utils.StringPtr("registry"),
					Values:    ctx.CompileString(`foo: "bar"`),
					Version:   utils.StringPtr("1.0.0"),
				},
			},
			yaml: "test",
			err:  false,
			validate: func(t *testing.T, result GeneratorResult, err error) {
				require.NoError(t, err)

				m := `{
	test: {
		instance:  "instance"
		name:      "test"
		namespace: "default"
		registry:  "registry"
		values: {
			foo: "bar"
		}
		version: "1.0.0"
	}
}`
				assert.Equal(t, m, string(result.Module))
				assert.Equal(t, "test", string(result.Manifests["test"]))
			},
		},
		{
			name: "manifest error",
			bundle: schema.DeploymentModuleBundle{
				"test": schema.DeploymentModule{
					Instance:  "instance",
					Name:      utils.StringPtr("test"),
					Namespace: "default",
					Registry:  utils.StringPtr("registry"),
					Values:    ctx.CompileString(`foo: "bar"`),
					Version:   utils.StringPtr("1.0.0"),
				},
			},
			yaml: "test",
			err:  true,
			validate: func(t *testing.T, result GeneratorResult, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "module error",
			bundle: schema.DeploymentModuleBundle{
				"test": schema.DeploymentModule{
					Instance:  "instance",
					Name:      utils.StringPtr("test"),
					Namespace: "default",
					Registry:  utils.StringPtr("registry"),
					Values:    fmt.Errorf("error"),
					Version:   utils.StringPtr("1.0.0"),
				},
			},
			yaml: "test",
			err:  false,
			validate: func(t *testing.T, result GeneratorResult, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := &mocks.ManifestGeneratorMock{
				GenerateFunc: func(mod schema.DeploymentModule) ([]byte, error) {
					if tt.err {
						return nil, fmt.Errorf("error")
					}

					return []byte(tt.yaml), nil
				},
			}
			gen := Generator{
				mg:     mg,
				logger: testutils.NewNoopLogger(),
			}

			result, err := gen.GenerateBundle(tt.bundle)
			tt.validate(t, result, err)
		})
	}
}

func TestGeneratorGenerate(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name     string
		module   schema.DeploymentModule
		yaml     string
		err      bool
		validate func(t *testing.T, result []byte, err error)
	}{
		{
			name: "full",
			module: schema.DeploymentModule{
				Instance:  "instance",
				Name:      utils.StringPtr("test"),
				Namespace: "default",
				Registry:  utils.StringPtr("registry"),
				Values:    ctx.CompileString(`foo: "bar"`),
				Version:   utils.StringPtr("1.0.0"),
			},
			yaml: "test",
			err:  false,
			validate: func(t *testing.T, result []byte, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test", string(result))
			},
		},
		{
			name: "manifest error",
			module: schema.DeploymentModule{
				Name:      utils.StringPtr("test"),
				Namespace: "default",
				Values:    ctx.CompileString(`foo: "bar"`),
				Version:   utils.StringPtr("1.0.0"),
			},
			yaml: "test",
			err:  true,
			validate: func(t *testing.T, result []byte, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mg := &mocks.ManifestGeneratorMock{
				GenerateFunc: func(mod schema.DeploymentModule) ([]byte, error) {
					if tt.err {
						return nil, fmt.Errorf("error")
					}

					return []byte(tt.yaml), nil
				},
			}
			gen := Generator{
				mg:     mg,
				logger: testutils.NewNoopLogger(),
			}

			result, err := gen.Generate(tt.module)
			tt.validate(t, result, err)
		})
	}
}
