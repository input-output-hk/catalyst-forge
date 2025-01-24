package deployment

import (
	"fmt"
	"testing"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/kcl"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/kcl/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestKCLRunnerGetMainValues(t *testing.T) {
	newProject := func(name string, modules map[string]schema.DeploymentModule) project.Project {
		return project.Project{
			Name: name,
			Blueprint: schema.Blueprint{
				Project: schema.Project{
					Deployment: schema.Deployment{
						Modules: modules,
					},
				},
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Environment: "test",
						Registries: schema.GlobalDeploymentRegistries{
							Modules: "test.com",
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name     string
		project  project.Project
		validate func(t *testing.T, values string, err error)
	}{
		{
			name: "full",
			project: newProject(
				"test",
				map[string]schema.DeploymentModule{
					"main": schema.DeploymentModule{
						Name:      "module",
						Namespace: "default",
						Values: map[string]string{
							"key": "value",
						},
						Version: "1.0.0",
					},
				},
			),
			validate: func(t *testing.T, values string, err error) {
				assert.NoError(t, err)
				assert.Equal(t, values, "key: value\n")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := KCLRunner{
				logger: testutils.NewNoopLogger(),
			}

			values, err := runner.GetMainValues(&tt.project, "main")
			tt.validate(t, values, err)
		})
	}
}

func TestKCLRunnerRunDeployment(t *testing.T) {
	type testResults struct {
		args   []kcl.KCLModuleArgs
		err    error
		result map[string]KCLRunResult
	}

	newProject := func(name, environment, registry string, modules map[string]schema.DeploymentModule) project.Project {
		return project.Project{
			Name: name,
			Blueprint: schema.Blueprint{
				Project: schema.Project{
					Deployment: schema.Deployment{
						Modules: modules,
					},
				},
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Environment: environment,
						Registries: schema.GlobalDeploymentRegistries{
							Modules: registry,
						},
					},
				},
			},
		}
	}

	tests := []struct {
		name     string
		project  project.Project
		output   string
		fail     bool
		validate func(t *testing.T, r testResults)
	}{
		{
			name: "full",
			project: newProject(
				"test",
				"dev",
				"test.com",
				map[string]schema.DeploymentModule{
					"main": {
						Name:      "module",
						Namespace: "default",
						Values: map[string]string{
							"key": "value",
						},
						Version: "1.0.0",
					},
					"support": {
						Name:      "module1",
						Namespace: "default",
						Values: map[string]string{
							"key1": "value1",
						},
						Version: "1.0.0",
					},
				},
			),
			output: "output",
			fail:   false,
			validate: func(t *testing.T, r testResults) {
				assert.NoError(t, r.err)
				assert.Equal(t, "output", r.result["main"].Manifests)
				assert.Equal(t, "key: value\n", r.result["main"].Values)
				assert.Equal(t, "output", r.result["support"].Manifests)
				assert.Equal(t, "key1: value1\n", r.result["support"].Values)

				assert.Contains(t, r.args, kcl.KCLModuleArgs{
					InstanceName: "test",
					Module:       "test.com/module",
					Namespace:    "default",
					Values:       "{\"key\":\"value\"}",
					Version:      "1.0.0",
				})

				assert.Contains(t, r.args, kcl.KCLModuleArgs{
					InstanceName: "test",
					Module:       "test.com/module1",
					Namespace:    "default",
					Values:       "{\"key1\":\"value1\"}",
					Version:      "1.0.0",
				})
			},
		},
		{
			name: "run failed",
			project: newProject(
				"test",
				"dev",
				"test.com",
				map[string]schema.DeploymentModule{
					"main": {
						Name:      "module",
						Namespace: "default",
						Values: map[string]string{
							"key": "value",
						},
						Version: "1.0.0",
					},
				},
			),
			output: "output",
			fail:   true,
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
				assert.ErrorContains(t, r.err, "error")
			},
		},
		{
			name: "no modules",
			project: newProject(
				"test",
				"dev",
				"test.com",
				nil,
			),
			output: "",
			fail:   false,
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
			},
		},
		{
			name: "no registry",
			project: newProject(
				"test",
				"dev",
				"",
				nil,
			),
			output: "",
			fail:   false,
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var kclArgs []kcl.KCLModuleArgs
			client := mocks.KCLClientMock{
				RunFunc: func(args kcl.KCLModuleArgs) (string, error) {
					kclArgs = append(kclArgs, args)

					if tt.fail {
						return "", fmt.Errorf("error")
					}

					return tt.output, nil
				},
				LogFunc: func() string {
					return "log"
				},
			}

			runner := KCLRunner{
				client: &client,
				logger: testutils.NewNoopLogger(),
			}

			result, err := runner.RunDeployment(&tt.project)
			tt.validate(t, testResults{
				args:   kclArgs,
				err:    err,
				result: result,
			})
		})
	}
}
