package generator

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratorGenerate(t *testing.T) {
	ctx := cuecontext.New()
	tests := []struct {
		name     string
		module   schema.DeploymentModule
		yaml     string
		err      bool
		validate func(t *testing.T, result GeneratorResult, err error)
	}{
		{
			name: "full",
			module: schema.DeploymentModule{
				Name:      "test",
				Namespace: "default",
				Values:    ctx.CompileString(`foo: "bar"`),
				Version:   "1.0.0",
			},
			yaml: "test",
			err:  false,
			validate: func(t *testing.T, result GeneratorResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, "test", string(result.Manifests))

				m := `{
	name:      "test"
	namespace: "default"
	values: {
		foo: "bar"
	}
	version: "1.0.0"
}`
				assert.Equal(t, m, string(result.Module))
			},
		},
		{
			name: "manifest error",
			module: schema.DeploymentModule{
				Name:      "test",
				Namespace: "default",
				Values:    ctx.CompileString(`foo: "bar"`),
				Version:   "1.0.0",
			},
			yaml: "test",
			err:  true,
			validate: func(t *testing.T, result GeneratorResult, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "module error",
			module: schema.DeploymentModule{
				Name:      "test",
				Namespace: "default",
				Values:    fmt.Errorf("error"),
				Version:   "1.0.0",
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
				GenerateFunc: func(mod schema.DeploymentModule, instance, registry string) ([]byte, error) {
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

			result, err := gen.Generate(tt.module, "", "")
			tt.validate(t, result, err)
		})
	}
}
