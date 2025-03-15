package providers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	exmocks "github.com/input-output-hk/catalyst-forge/cli/pkg/executor/mocks"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	spr "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCueReleaserRelease(t *testing.T) {
	type testResults struct {
		calls    []string
		err      error
		registry string
		repoName string
	}

	newProject := func(
		registry string,
		prefix string,
		path string,
	) project.Project {
		return project.Project{
			Blueprint: sb.Blueprint{
				Global: &sg.Global{
					Ci: &sg.CI{
						Providers: &spr.Providers{
							Aws: &spr.AWS{
								Ecr: spr.AWSECR{
									AutoCreate: true,
								},
							},
							Cue: &spr.CUE{
								Registry:       registry,
								RegistryPrefix: prefix,
							},
						},
					},
					Repo: &sg.Repo{
						Name: "test",
					},
				},
				Project: &sp.Project{},
			},
			Path:     path,
			RepoRoot: "/",
		}
	}

	tests := []struct {
		name     string
		project  project.Project
		release  sp.Release
		config   CueReleaserConfig
		files    map[string]string
		firing   bool
		force    bool
		failOn   string
		validate func(t *testing.T, r testResults)
	}{
		{
			name: "full",
			project: newProject(
				"https://123456789012.dkr.ecr.us-west-2.amazonaws.com",
				"prefix",
				"/project",
			),
			release: sp.Release{},
			config: CueReleaserConfig{
				Version: "v1.0.0",
			},
			files: map[string]string{
				"/project/cue.mod/module.cue": `module: "site.com/test"`,
			},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Contains(t, r.calls, "mod publish v1.0.0")
				assert.Equal(t, "https://123456789012.dkr.ecr.us-west-2.amazonaws.com/prefix", r.registry)
				assert.Equal(t, "site.com/test", r.repoName)
			},
		},
		{
			name: "not firing",
			project: newProject(
				"https://123456789012.dkr.ecr.us-west-2.amazonaws.com",
				"prefix",
				"/project",
			),
			release: sp.Release{},
			config:  CueReleaserConfig{},
			files:   map[string]string{},
			firing:  false,
			force:   false,
			failOn:  "",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Empty(t, r.calls)
			},
		},
		{
			name:    "no registry",
			project: newProject("", "", "/project"),
			release: sp.Release{},
			config:  CueReleaserConfig{},
			files:   map[string]string{},
			firing:  true,
			force:   false,
			failOn:  "",
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
				assert.Equal(t, "must specify at least one CUE registry", r.err.Error())
			},
		},
		{
			name: "no module file",
			project: newProject(
				"https://123456789012.dkr.ecr.us-west-2.amazonaws.com",
				"prefix",
				"/project",
			),
			release: sp.Release{},
			config: CueReleaserConfig{
				Version: "v1.0.0",
			},
			files:  map[string]string{},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
				assert.Equal(t, "failed to load module: module file does not exist: /project/cue.mod/module.cue", r.err.Error())
			},
		},
		{
			name: "no module entry",
			project: newProject(
				"https://123456789012.dkr.ecr.us-west-2.amazonaws.com",
				"prefix",
				"/project",
			),
			release: sp.Release{},
			config: CueReleaserConfig{
				Version: "v1.0.0",
			},
			files: map[string]string{
				"/project/cue.mod/module.cue": `foo: "bar"`,
			},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, r testResults) {
				assert.Error(t, r.err)
				assert.Equal(t, "failed to load module: module file does not contain a module definition", r.err.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := billy.NewInMemoryFs()
			testutils.SetupFS(t, fs, tt.files)

			var repoName string
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

			var calls []string
			var registry string
			cue := CueReleaser{
				config:  tt.config,
				cue:     newWrappedCueExecuterMock(&calls, &registry, tt.failOn),
				ecr:     ecr,
				force:   tt.force,
				fs:      fs,
				handler: newReleaseEventHandlerMock(tt.firing),
				logger:  testutils.NewNoopLogger(),
				project: tt.project,
				release: tt.release,
			}

			err := cue.Release()

			tt.validate(t, testResults{
				calls:    calls,
				err:      err,
				registry: registry,
				repoName: repoName,
			})
		})
	}
}

func newWrappedCueExecuterMock(calls *[]string, registry *string, failOn string) *exmocks.WrappedExecuterMock {
	return &exmocks.WrappedExecuterMock{
		ExecuteFunc: func(args ...string) ([]byte, error) {
			call := strings.Join(args, " ")
			*calls = append(*calls, call)

			r := os.Getenv("CUE_REGISTRY")
			*registry = r

			if failOn != "" && strings.HasPrefix(call, failOn) {
				return nil, fmt.Errorf("failed to execute command")
			}
			return nil, nil
		},
	}
}
