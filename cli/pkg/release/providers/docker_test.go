package providers

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	exmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/executor/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws/mocks"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerReleaserRelease(t *testing.T) {

	newProject := func(
		container string,
		registries []string,
		platforms []string,
	) project.Project {
		return project.Project{
			Blueprint: sb.Blueprint{
				Global: &sg.Global{
					Ci: &sg.CI{
						Registries: registries,
					},
					Repo: &sg.Repo{
						Name: "owner/repo",
					},
				},
				Project: &sp.Project{
					Container: container,
					Ci: &sp.CI{
						Targets: map[string]sp.Target{
							"test": {
								Platforms: platforms,
							},
						},
					},
				},
			},
		}
	}

	newRelease := func() sp.Release {
		return sp.Release{
			Target: "test",
		}
	}

	tests := []struct {
		name       string
		project    project.Project
		release    sp.Release
		config     DockerReleaserConfig
		firing     bool
		force      bool
		runFail    bool
		execFailOn string
		validate   func(t *testing.T, calls []string, repoName string, err error)
	}{
		{
			name: "full",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{},
			),
			release: newRelease(),
			config: DockerReleaserConfig{
				Tag: "test",
			},
			firing:  true,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, repoName string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s test.com/repo/test:test", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/repo/test:test")
			},
		},
		{
			name: "ecr",
			project: newProject(
				"test",
				[]string{"123456789012.dkr.ecr.us-west-2.amazonaws.com"},
				[]string{},
			),
			release: newRelease(),
			config: DockerReleaserConfig{
				Tag: "test",
			},
			firing:  true,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, repoName string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s 123456789012.dkr.ecr.us-west-2.amazonaws.com/repo/test:test", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push 123456789012.dkr.ecr.us-west-2.amazonaws.com/repo/test:test")
				assert.Equal(t, "repo/test", repoName)
			},
		},
		{
			name: "multiple platforms",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{"linux", "windows"},
			),
			release: newRelease(),
			config: DockerReleaserConfig{
				Tag: "test",
			},
			firing:  true,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, repoName string, err error) {
				require.NoError(t, err)

				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s_linux", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s_windows", CONTAINER_NAME, TAG_NAME))

				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s_linux test.com/repo/test:test_linux", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/repo/test:test_linux")

				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s_windows test.com/repo/test:test_windows", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/repo/test:test_windows")
			},
		},
		{
			name: "no image tag",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{"linux", "windows"},
			),
			release: sp.Release{},
			config:  DockerReleaserConfig{},
			firing:  true,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, repoName string, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "no image tag specified")
			},
		},
		{
			name:    "run fails",
			project: project.Project{},
			release: sp.Release{},
			firing:  true,
			force:   false,
			runFail: true,
			validate: func(t *testing.T, calls []string, repoName string, err error) {
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
			),
			release:    newRelease(),
			firing:     true,
			force:      false,
			runFail:    false,
			execFailOn: "inspect",
			validate: func(t *testing.T, calls []string, repoName string, err error) {
				require.Error(t, err)
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
				assert.NotContains(t, calls, "push test.com/repo/test:test")
			},
		},
		{
			name: "not firing",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{},
			),
			release: newRelease(),
			firing:  false,
			force:   false,
			runFail: false,
			validate: func(t *testing.T, calls []string, repoName string, err error) {
				require.NoError(t, err)
				assert.NotContains(t, calls, "push test.com/repo/test:test")
			},
		},
		{
			name: "forced",
			project: newProject(
				"test",
				[]string{"test.com"},
				[]string{},
			),
			release: newRelease(),
			config: DockerReleaserConfig{
				Tag: "test",
			},
			firing:  false,
			force:   true,
			runFail: false,
			validate: func(t *testing.T, calls []string, repoName string, err error) {
				require.NoError(t, err)
				assert.Contains(t, calls, fmt.Sprintf("inspect %s:%s", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, fmt.Sprintf("tag %s:%s test.com/repo/test:test", CONTAINER_NAME, TAG_NAME))
				assert.Contains(t, calls, "push test.com/repo/test:test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repoName string
			var calls []string

			mock := mocks.AWSECRClientMock{
				CreateRepositoryFunc: func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
					repoName = *params.RepositoryName
					return &ecr.CreateRepositoryOutput{}, nil
				},
				DescribeRepositoriesFunc: func(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
					return nil, fmt.Errorf("RepositoryNotFoundException")
				},
			}
			ecr := aws.NewCustomECRClient(&mock, testutils.NewNoopLogger())

			releaser := DockerReleaser{
				config:  tt.config,
				docker:  newWrappedExecuterMock(&calls, tt.execFailOn),
				ecr:     ecr,
				force:   tt.force,
				handler: newReleaseEventHandlerMock(tt.firing),
				logger:  testutils.NewNoopLogger(),
				project: tt.project,
				release: tt.release,
				runner:  newProjectRunnerMock(tt.runFail),
			}

			err := releaser.Release()
			tt.validate(t, calls, repoName, err)
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
