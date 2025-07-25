package providers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws/mocks"
	sb "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/blueprint/global"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_createECRRepoIfNotExists(t *testing.T) {
	type testResults struct {
		err            error
		createParams   *ecr.CreateRepositoryInput
		describeParams *ecr.DescribeRepositoriesInput
	}

	tests := []struct {
		name     string
		registry string
		exists   bool
		validate func(t *testing.T, r testResults)
	}{
		{
			name:     "does not exist",
			registry: "test.com/myrepo",
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Equal(t, "myrepo", r.describeParams.RepositoryNames[0])
				assert.Equal(t, "myrepo", *r.createParams.RepositoryName)
			},
		},
		{
			name:     "exists",
			registry: "test.com/myrepo",
			exists:   true,
			validate: func(t *testing.T, r testResults) {
				require.NoError(t, r.err)
				assert.Equal(t, "myrepo", r.describeParams.RepositoryNames[0])
				assert.Nil(t, r.createParams)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var createParams *ecr.CreateRepositoryInput
			var describeParams *ecr.DescribeRepositoriesInput

			mock := mocks.AWSECRClientMock{
				CreateRepositoryFunc: func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
					createParams = params
					return &ecr.CreateRepositoryOutput{}, nil
				},
				DescribeRepositoriesFunc: func(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
					describeParams = params
					if !tt.exists {
						return nil, fmt.Errorf("RepositoryNotFoundException")
					} else {
						return &ecr.DescribeRepositoriesOutput{}, nil
					}
				},
			}
			client := aws.NewCustomECRClient(&mock, testutils.NewNoopLogger())

			project := project.Project{
				Blueprint: sb.Blueprint{
					Global: &sg.Global{
						Repo: &sg.Repo{
							Name: "test",
						},
					},
				},
				Path: "path",
			}

			err := createECRRepoIfNotExists(client, &project, tt.registry, testutils.NewNoopLogger())
			tt.validate(t, testResults{
				err:            err,
				createParams:   createParams,
				describeParams: describeParams,
			})
		})
	}
}
