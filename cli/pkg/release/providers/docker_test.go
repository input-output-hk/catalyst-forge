package providers

import (
	"fmt"
	"strings"
	"testing"

	exmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/executor/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerReleaserRelease(t *testing.T) {

	newProject := func(
		container string,
		registries []string,
		platforms []string,
		tag *project.ProjectTag,
	) project.Project {
		return project.Project{
			Blueprint: schema.Blueprint{
				Global: schema.Global{
					CI: schema.GlobalCI{
						Registries: registries,
					},
				},
				Project: schema.Project{
					Container: container,
					CI: schema.ProjectCI{
						Targets: map[string]schema.Target{
							"test": {
								Platforms: platforms,
							},
						},
					},
				},
			},
			Tag: tag,
		}
	}

	newRelease := func() schema.Release {
		return schema.Release{
			Target: "test",
		}
	}

	tests := []struct {
		name       string
		project    project.Project
		release    schema.Release
		config     DockerReleaserConfig
		firing     bool
		force      bool
		runFail    bool
		execFailOn string
		validate   func(t *testing.T, calls []string, err error)
	}{
		{
			name: "full",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{},
				nil,
			),
			release: newRelease(),
			config: DockerReleaserConfig{
				Tag: "test",
			},
			firing:  true,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s test.com/test:test", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/test:test")
			},
		},
		// {
		// 	name: "with git tag",
		// 	project: newProject(
		// 		"test",
		// 		[]string{"test.com"},
		// 		[]string{},
		// 		&project.ProjectTag{
		// 			Version: "v1.0.0",
		// 		},
		// 	),
		// 	release: newRelease(),
		// 	config: DockerReleaserConfig{
		// 		Tag: "test",
		// 	},
		// 	firing:  true,
		// 	force:   false,
		// 	runFail: false,
		// 	validate: func(t *testing.T, calls []string, err error) {
		// 		require.NoError(t, err)
		// 		assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
		// 		assert.Contains(t, calls, fmt.Sprintf("tag %s:%s test.com/test:v1.0.0", CONTAINER_NAME, TAG_NAME))
		// 		assert.Contains(t, calls, "push test.com/test:v1.0.0")
		// 	},
		// },
		{
			name: "multiple platforms",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{"linux", "windows"},
				nil,
			),
			release: newRelease(),
			config: DockerReleaserConfig{
				Tag: "test",
			},
			firing:  true,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)

				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s_linux", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s_windows", CONTAINER_NAME, TAG_NAME))

				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s_linux test.com/test:test_linux", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/test:test_linux")

				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s_windows test.com/test:test_windows", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/test:test_windows")
			},
		},
		{
			name: "no image tag",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{"linux", "windows"},
				nil,
			),
			release: schema.Release{},
			config:  DockerReleaserConfig{},
			firing:  true,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "no image tag specified")
			},
		},
		{
			name:    "run fails",
			project: project.Project{},
			release: schema.Release{},
			firing:  true,
			force:   false,
			runFail: true,
			validate: func(t *testing.T, calls []string, err error) {
				require.Error(t, err)
				assert.NotContains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
			},
		},
		{
			name: "image does not exist",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{},
				nil,
			),
			release:    newRelease(),
			firing:     true,
			force:      false,
			runFail:    false,
			execFailOn: "inspect",
			validate: func(t *testing.T, calls []string, err error) {
				require.Error(t, err)
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
				assert.NotContains(t, calls, "push test.com/test:test")
			},
		},
		{
			name: "not firing",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{},
				nil,
			),
			release: newRelease(),
			firing:  false,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.NotContains(t, calls, "push test.com/test:test")
			},
		},
		{
			name: "forced",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{},
				nil,
			),
			release: newRelease(),
			config: DockerReleaserConfig{
				Tag: "test",
			},
			firing:  false,
			force:   true,
			runFail: false,
			validate: func(t *testing.T, calls []string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s test.com/test:test", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/test:test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			releaser := DockerReleaser{
				config:  tt.config,
				docker:  newWrappedExecuterMock(&calls, tt.execFailOn),
				force:   tt.force,
				handler: newReleaseEventHandlerMock(tt.firing),
				logger:  testutils.NewNoopLogger(),
				project: tt.project,
				release: tt.release,
				runner:  newProjectRunnerMock(tt.runFail),
			}

			err := releaser.Release()
			tt.validate(t, calls, err)
		})
	}
}

func newWrappedExecuterMock(calls *[]string, failOn string) *exmocks.WrappedExecuterMock {
	return &exmocks.WrappedExecuterMock{
		ExecuteFunc: func(args ...string) ([]byte, error) {
			call := strings.Join(args, " ")
			*calls = append(*calls, call)

			if failOn != "" && strings.HasPrefix(call, failOn) {
				return nil, fmt.Errorf("failed to execute command")
			}
			return nil, nil
		},
	}
}
