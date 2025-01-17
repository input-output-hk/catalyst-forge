package deployment

import (
	"fmt"
	"strings"
	"testing"

	"github.com/input-output-hk/catalyst-forge/cli/pkg/executor/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestKCLRunnerGetMainValues(t *testing.T) {
	newProject := func(name string, modules *schema.DeploymentModules) project.Project {
		return project.Project{
			Name: name,
			Blueprint: schema.Blueprint{
				Project: schema.Project{
					Deployment: schema.Deployment{
						Environment: "test",
						Modules:     modules,
					},
				},
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Registry: "test",
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
				&schema.DeploymentModules{
					Main: schema.Module{
						Module:    "module",
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
				kcl:    newWrappedExecuterMock("", nil, false),
				logger: testutils.NewNoopLogger(),
			}

			values, err := runner.GetMainValues(&tt.project)
			tt.validate(t, values, err)
		})
	}
}

func TestKCLRunnerRunDeployment(t *testing.T) {
	type testResults struct {
		calls  []string
		err    error
		output string
	}

	newProject := func(name, environment, registry string, modules *schema.DeploymentModules) project.Project {
		return project.Project{
			Name: name,
			Blueprint: schema.Blueprint{
				Project: schema.Project{
					Deployment: schema.Deployment{
						Environment: environment,
						Modules:     modules,
					},
				},
				Global: schema.Global{
					Deployment: schema.GlobalDeployment{
						Registry: registry,
					},
				},
			},
		}
	}

	tests := []struct {
		name     string
		project  project.Project
		output   string
		execFail bool
		validate func(t *testing.T, r testResults)
	}{
		{
			name: "full",
			project: newProject(
				"test",
				"dev",
				"test.com",
				&schema.DeploymentModules{
					Main: schema.Module{
						Module:    "module",
						Namespace: "default",
						Values: map[string]string{
							"key": "value",
						},
						Version: "1.0.0",
					},
					Support: map[string]schema.Module{
						"support": {
							Module:    "module1",
							Namespace: "default",
							Values: map[string]string{
								"key1": "value1",
							},
							Version: "1.0.0",
						},
					},
				},
			),
			output:   "output",
			execFail: false,
			validate: func(t *testing.T, r testResults) {
				assert.NoError(t, r.err)
				assert.Equal(t, "output\n---\noutput\n", r.output)
				assert.Contains(t, r.calls, "run -q -D name= -D namespace=default -D values={\"key\":\"value\"} -D 1.0.0 oci://test.com/module")
				assert.Contains(t, r.calls, "run -q -D name= -D namespace=default -D values={\"key1\":\"value1\"} -D 1.0.0 oci://test.com/module1")
			},
		},
		{
			name: "run failed",
			project: newProject(
				"test",
				"dev",
				"test.com",
				&schema.DeploymentModules{
					Main: schema.Module{
						Module:    "module",
						Namespace: "default",
						Values: map[string]string{
							"key": "value",
						},
						Version: "1.0.0",
					},
				},
			),
			output:   "output",
			execFail: true,
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
				assert.ErrorContains(t, r.err, "failed to execute command")
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
			output:   "",
			execFail: false,
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
			output:   "",
			execFail: false,
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			runner := KCLRunner{
				kcl:    newWrappedExecuterMock(tt.output, &calls, tt.execFail),
				logger: testutils.NewNoopLogger(),
			}

			output, err := runner.RunDeployment(&tt.project)
			tt.validate(t, testResults{
				calls:  calls,
				err:    err,
				output: string(output),
			})
		})
	}
}

func newWrappedExecuterMock(output string, calls *[]string, fail bool) *mocks.WrappedExecuterMock {
	return &mocks.WrappedExecuterMock{
		ExecuteFunc: func(args ...string) ([]byte, error) {
			call := strings.Join(args, " ")
			*calls = append(*calls, call)

			if fail {
				return nil, fmt.Errorf("failed to execute command")
			}
			return []byte(output), nil
		},
	}
}
