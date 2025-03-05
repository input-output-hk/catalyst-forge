package providers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws"
	"github.com/input-output-hk/catalyst-forge/cli/pkg/providers/aws/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	spr "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global/providers"
	sp "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/project"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKCLReleaserRelease(t *testing.T) {
	type testResults struct {
		calls    []string
		err      error
		repoName string
	}

	newProject := func(
		name string,
		registries []string,
	) project.Project {
		return project.Project{
			Name: name,
			Blueprint: sb.Blueprint{
				Global: &sg.Global{
					Ci: &sg.CI{
						Providers: &spr.Providers{
							Kcl: &spr.KCL{
								Registries: registries,
							},
						},
					},
					Repo: &sg.Repo{
						Name: "repo",
					},
				},
				Project: &sp.Project{},
			},
		}
	}

	tests := []struct {
		name     string
		project  project.Project
		release  sp.Release
		config   KCLReleaserConfig
		firing   bool
		force    bool
		failOn   string
		validate func(t *testing.T, r testResults)
	}{
		{
			name:    "full",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: KCLReleaserConfig{
				Container: "name",
			},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Contains(t, r.calls, "mod push oci://test.com/repo/name")
			},
		},
		{
			name:    "ECR",
			project: newProject("test", []string{"123456789012.dkr.ecr.us-west-2.amazonaws.com"}),
			release: sp.Release{},
			config: KCLReleaserConfig{
				Container: "name",
			},
			firing: true,
			force:  false,
			failOn: "",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Contains(t, r.calls, "mod push oci://123456789012.dkr.ecr.us-west-2.amazonaws.com/repo/name")
				assert.Equal(t, "repo/name", r.repoName)
			},
		},
		{
			name:    "no container",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config:  KCLReleaserConfig{},
			firing:  true,
			force:   false,
			failOn:  "",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Contains(t, r.calls, "mod push oci://test.com/repo/test")
			},
		},
		{
			name:    "not firing",
			project: newProject("test", []string{"test.com"}),
			firing:  false,
			force:   false,
			failOn:  "",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Len(t, r.calls, 0)
			},
		},
		{
			name:    "forced",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: KCLReleaserConfig{
				Container: "test",
			},
			firing: false,
			force:  true,
			failOn: "",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Contains(t, r.calls, "mod push oci://test.com/repo/test")
			},
		},
		{
			name:    "push fails",
			project: newProject("test", []string{"test.com"}),
			release: sp.Release{},
			config: KCLReleaserConfig{
				Container: "test",
			},
			firing: true,
			force:  false,
			failOn: "mod push",
			validate: func(t *testing.T, r testResults) {
				require.Error(t, r.err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			kcl := KCLReleaser{
				config:  tt.config,
				ecr:     ecr,
				force:   tt.force,
				handler: newReleaseEventHandlerMock(tt.firing),
				kcl:     newWrappedExecuterMock(&calls, tt.failOn),
				logger:  testutils.NewNoopLogger(),
				project: tt.project,
				release: tt.release,
			}

			err := kcl.Release()

			tt.validate(t, testResults{
				calls:    calls,
				err:      err,
				repoName: repoName,
			})
		})
	}
}
