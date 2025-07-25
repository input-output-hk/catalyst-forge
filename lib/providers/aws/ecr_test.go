package aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/input-output-hk/catalyst-forge/lib/providers/aws/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestECRClient_CreateECRRepository(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		path     string
		fail     bool
		validate func(t *testing.T, params *ecr.CreateRepositoryInput, err error)
	}{
		{
			name: "simple",
			repo: "myapp",
			path: "/path/to/myapp",
			validate: func(t *testing.T, params *ecr.CreateRepositoryInput, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "myapp", *params.RepositoryName)
				assert.Equal(t, types.ImageTagMutabilityImmutable, params.ImageTagMutability)
				assert.Equal(t, true, params.ImageScanningConfiguration.ScanOnPush)
				assert.Equal(t, types.EncryptionTypeAes256, params.EncryptionConfiguration.EncryptionType)
				assert.ElementsMatch(t, []types.Tag{
					{Key: aws.String("BuiltWith"), Value: aws.String("Catalyst Forge")},
					{Key: aws.String("Repo"), Value: aws.String("myapp")},
					{Key: aws.String("RepoPath"), Value: aws.String("/path/to/myapp")},
				}, params.Tags)
			},
		},
		{
			name: "failed",
			repo: "myapp",
			path: "/path/to/myapp",
			fail: true,
			validate: func(t *testing.T, params *ecr.CreateRepositoryInput, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pp *ecr.CreateRepositoryInput
			mock := mocks.AWSECRClientMock{
				CreateRepositoryFunc: func(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error) {
					pp = params

					if tt.fail {
						return nil, fmt.Errorf("failed to create repository")
					} else {
						return &ecr.CreateRepositoryOutput{}, nil
					}
				},
			}

			client := &ECRClient{
				client: &mock,
				logger: testutils.NewNoopLogger(),
			}

			err := client.CreateECRRepository(tt.repo, tt.repo, tt.path)
			tt.validate(t, pp, err)
		})
	}
}

func TestECRClient_ECRRepoExists(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		exists   bool
		fail     bool
		validate func(t *testing.T, result bool, err error)
	}{
		{
			name:   "simple",
			repo:   "myapp",
			exists: true,
			validate: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.True(t, result)
			},
		},
		{
			name:   "not found",
			repo:   "myapp",
			exists: false,
			validate: func(t *testing.T, result bool, err error) {
				assert.NoError(t, err)
				assert.False(t, result)
			},
		},
		{
			name: "failed",
			repo: "myapp",
			fail: true,
			validate: func(t *testing.T, result bool, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mocks.AWSECRClientMock{
				DescribeRepositoriesFunc: func(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
					if tt.fail {
						return nil, fmt.Errorf("failed to describe repositories")
					} else if tt.exists {
						return &ecr.DescribeRepositoriesOutput{}, nil
					} else {
						return nil, fmt.Errorf("RepositoryNotFoundException")
					}
				},
			}

			client := &ECRClient{
				client: &mock,
				logger: testutils.NewNoopLogger(),
			}

			result, err := client.ECRRepoExists(tt.repo)
			tt.validate(t, result, err)
		})
	}
}

func TestExtractECRRepoName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple repository name",
			input: "123456789012.dkr.ecr.us-west-2.amazonaws.com/myapp",
			want:  "myapp",
		},
		{
			name:  "repository name with path",
			input: "123456789012.dkr.ecr.us-west-2.amazonaws.com/team/myapp",
			want:  "team/myapp",
		},
		{
			name:  "repository with tag",
			input: "123456789012.dkr.ecr.us-west-2.amazonaws.com/myapp:latest",
			want:  "myapp",
		},
		{
			name:  "repository with path and tag",
			input: "123456789012.dkr.ecr.us-west-2.amazonaws.com/team/myapp:v1.2.3",
			want:  "team/myapp",
		},
		{
			name:  "repository with digest",
			input: "123456789012.dkr.ecr.us-west-2.amazonaws.com/myapp@sha256:abc123",
			want:  "myapp",
		},
		{
			name:  "repository with HTTPS protocol",
			input: "https://123456789012.dkr.ecr.us-west-2.amazonaws.com/myapp",
			want:  "myapp",
		},
		{
			name:  "repository with multiple path segments",
			input: "123456789012.dkr.ecr.us-west-2.amazonaws.com/org/team/myapp",
			want:  "org/team/myapp",
		},
		{
			name:    "empty URI",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid URI format",
			input:   "123456789012.dkr.ecr.us-west-2.amazonaws.com",
			wantErr: true,
		},
		{
			name:  "repository with tag and digest",
			input: "123456789012.dkr.ecr.us-west-2.amazonaws.com/myapp:latest@sha256:abc123",
			want:  "myapp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractECRRepoName(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsECRRegistry(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123456789012.dkr.ecr.us-west-2.amazonaws.com", true},
		{"123456789012.dkr.ecr.us-west-2.amazonaws.com/myrepo", true},
		{"https://123456789012.dkr.ecr.us-west-2.amazonaws.com", true},
		{"ghcr.io/org/repo", false},
		{"docker.io/library/ubuntu", false},
		{"quay.io/organization/repo", false},
		{"123456.dkr.ecr.region.amazonaws.com", false},
		{"abcdef123456.dkr.ecr.us-west-2.amazonaws.com", false},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			actual := IsECRRegistry(test.input)
			assert.Equal(t, test.expected, actual)
		})
	}
}
