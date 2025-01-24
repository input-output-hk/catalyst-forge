package kcl

import (
	"fmt"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/kcl/client"
	"github.com/input-output-hk/catalyst-forge/lib/project/deployment/providers/kcl/client/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKCLManifestGeneratorGenerate(t *testing.T) {
	type testResult struct {
		conf      client.KCLModuleConfig
		container string
		err       error
		out       []byte
	}

	tests := []struct {
		name     string
		module   schema.DeploymentModule
		instance string
		registry string
		out      string
		err      bool
		validate func(t *testing.T, result testResult)
	}{
		{
			name: "full",
			module: schema.DeploymentModule{
				Name:      "module",
				Namespace: "default",
				Values:    "test",
				Version:   "1.0.0",
			},
			instance: "instance",
			registry: "registry",
			out:      "output",
			err:      false,
			validate: func(t *testing.T, result testResult) {
				require.NoError(t, result.err)
				assert.Equal(t, client.KCLModuleConfig{
					InstanceName: "instance",
					Namespace:    "default",
					Values:       "test",
				}, result.conf)
				assert.Equal(t, []byte("output"), result.out)
				assert.Equal(t, "oci://registry/module?tag=1.0.0", result.container)
			},
		},
		{
			name: "error",
			module: schema.DeploymentModule{
				Name:      "module",
				Namespace: "default",
				Values:    "test",
				Version:   "1.0.0",
			},
			instance: "instance",
			registry: "registry",
			out:      "output",
			err:      true,
			validate: func(t *testing.T, result testResult) {
				assert.Error(t, result.err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cont string
			var c client.KCLModuleConfig
			m := &mocks.KCLClientMock{
				RunFunc: func(container string, conf client.KCLModuleConfig) (string, error) {
					cont = container
					c = conf

					if tt.err {
						return "", fmt.Errorf("error")
					}

					return tt.out, nil
				},
			}

			g := &KCLManifestGenerator{
				client: m,
				logger: testutils.NewNoopLogger(),
			}

			out, err := g.Generate(tt.module, tt.instance, tt.registry)
			tt.validate(t, testResult{
				conf:      c,
				container: cont,
				err:       err,
				out:       out,
			})
		})
	}
}
