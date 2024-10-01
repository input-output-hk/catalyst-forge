package deployment

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/project"
	"github.com/input-output-hk/catalyst-forge/lib/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/pointers"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestGenerateBundleEncode(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name        string
		blueprint   schema.Blueprint
		expected    string
		expectErr   bool
		expectedErr string
	}{
		{
			name: "simple",
			blueprint: schema.Blueprint{
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Registry: "test.registry.com",
					},
				},
				Project: schema.Project{
					Name: "test",
					Deployment: schema.Deployment{
						Environment: "test",
						Modules: &schema.DeploymentModules{
							Main: schema.Module{
								Container: pointers.String("test"),
								Namespace: "test",
								Values:    ctx.CompileString(`{foo: "bar"}`),
								Version:   "1.0.0",
							},
						},
					},
				},
			},
			expected: `{
	apiVersion: "v1alpha1"
	name:       "test"
	instances: {
		test: {
			module: {
				digest:  ""
				url:     "test.registry.com/test"
				version: "1.0.0"
			}
			namespace: "test"
			values: {
				foo: "bar"
			}
		}
	}
}`,
			expectErr: false,
		},
		{
			name: "support",
			blueprint: schema.Blueprint{
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Registry: "test.registry.com",
					},
				},
				Project: schema.Project{
					Name: "test",
					Deployment: schema.Deployment{
						Environment: "test",
						Modules: &schema.DeploymentModules{
							Main: schema.Module{
								Container: pointers.String("test"),
								Namespace: "test",
								Values:    ctx.CompileString(`{foo: "bar"}`),
								Version:   "1.0.0",
							},
							Support: map[string]schema.Module{
								"support": {
									Container: pointers.String("test"),
									Namespace: "test",
									Values:    ctx.CompileString(`{foo: "bar"}`),
									Version:   "1.0.0",
								},
							},
						},
					},
				},
			},
			expected: `{
	apiVersion: "v1alpha1"
	name:       "test"
	instances: {
		support: {
			module: {
				digest:  ""
				url:     "test.registry.com/test"
				version: "1.0.0"
			}
			namespace: "test"
			values: {
				foo: "bar"
			}
		}
		test: {
			module: {
				digest:  ""
				url:     "test.registry.com/test"
				version: "1.0.0"
			}
			namespace: "test"
			values: {
				foo: "bar"
			}
		}
	}
}`,
			expectErr: false,
		},
		{
			name: "no modules",
			blueprint: schema.Blueprint{
				Project: schema.Project{
					Name: "test",
				},
			},
			expected:    "",
			expectErr:   true,
			expectedErr: "no deployment modules found in project blueprint",
		},
		{
			name: "no registry",
			blueprint: schema.Blueprint{
				Project: schema.Project{
					Name: "test",
					Deployment: schema.Deployment{
						Environment: "test",
						Modules: &schema.DeploymentModules{
							Main: schema.Module{
								Container: pointers.String("test"),
								Namespace: "test",
								Values:    ctx.CompileString(`{foo: "bar"}`),
							},
						},
					},
				},
			},
			expected:    "",
			expectErr:   true,
			expectedErr: "no deployment registry found in project blueprint",
		},
		{
			name: "invalid url",
			blueprint: schema.Blueprint{
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Registry: "%^&",
					},
				},
				Project: schema.Project{
					Name: "test",
					Deployment: schema.Deployment{
						Environment: "test",
						Modules: &schema.DeploymentModules{
							Main: schema.Module{
								Container: pointers.String("test"),
								Namespace: "test",
								Values:    ctx.CompileString(""),
								Version:   "1.0.0",
							},
						},
					},
				},
			},
			expected:    "",
			expectErr:   true,
			expectedErr: `failed to build main module instance: failed to generate module URL: parse "%^&": invalid URL escape "%^&"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := project.Project{
				Blueprint: tt.blueprint,
			}

			bundle, err := GenerateBundle(&project)
			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}

			src, err := bundle.Encode()
			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}

			assert.Equal(t, tt.expected, string(src))
		})
	}
}
