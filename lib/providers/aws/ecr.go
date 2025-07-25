package aws

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go/aws"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks -out mocks/ecr.go . AWSECRClient

// AWSECRClient is an interface for an AWS ECR client.
type AWSECRClient interface {
	CreateRepository(ctx context.Context, params *ecr.CreateRepositoryInput, optFns ...func(*ecr.Options)) (*ecr.CreateRepositoryOutput, error)
	DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error)
}

// ECRClient is a client for interacting with AWS ECR.
type ECRClient struct {
	client AWSECRClient
	logger *slog.Logger
}

// CreateECRRepository creates a new ECR repository.
// By default, the repository is immutable and has image scanning enabled.
// The repository is also tagged with metadata about the given project.
func (c *ECRClient) CreateECRRepository(name string, gitRepoName, gitRepoPath string) error {
	input := &ecr.CreateRepositoryInput{
		RepositoryName:     aws.String(name),
		ImageTagMutability: types.ImageTagMutabilityImmutable,
		ImageScanningConfiguration: &types.ImageScanningConfiguration{
			ScanOnPush: true,
		},
		EncryptionConfiguration: &types.EncryptionConfiguration{
			EncryptionType: types.EncryptionTypeAes256,
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("BuiltWith"),
				Value: aws.String("Catalyst Forge"),
			},
			{
				Key:   aws.String("Repo"),
				Value: aws.String(gitRepoName),
			},
			{
				Key:   aws.String("RepoPath"),
				Value: aws.String(gitRepoPath),
			},
		},
	}

	_, err := c.client.CreateRepository(context.TODO(), input)
	if err != nil {
		return err
	}

	return nil
}

// ECRRepoExists checks whether an ECR repository exists.
func (c *ECRClient) ECRRepoExists(name string) (bool, error) {
	_, err := c.client.DescribeRepositories(context.Background(), &ecr.DescribeRepositoriesInput{
		RepositoryNames: []string{name},
	})
	if err != nil {
		if strings.Contains(err.Error(), "RepositoryNotFoundException") || strings.Contains(err.Error(), "not found") {
			return false, nil
		} else {
			return false, err
		}
	}

	return true, nil
}

// NewECRClient returns a new ECR client.
func NewECRClient(logger *slog.Logger) (ECRClient, error) {
	cfg, err := NewConfig()
	if err != nil {
		return ECRClient{}, err
	}

	c := ecr.NewFromConfig(cfg)

	return ECRClient{
		client: c,
		logger: logger,
	}, nil
}

// NewCustomECRClient returns a new ECR client with a custom client.
func NewCustomECRClient(client AWSECRClient, logger *slog.Logger) ECRClient {
	return ECRClient{
		client: client,
		logger: logger,
	}
}

// ExtractECRRepoName extracts the repository name from an ECR URI.
func ExtractECRRepoName(ecrURI string) (string, error) {
	if strings.Contains(ecrURI, "://") {
		parts := strings.SplitN(ecrURI, "://", 2)
		ecrURI = parts[1]
	}

	parts := strings.SplitN(ecrURI, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid ECR URI format: no repository path found")
	}

	repoPath := parts[1]
	if strings.Contains(repoPath, "@sha256:") {
		repoPath = strings.Split(repoPath, "@sha256:")[0]
	}
	if strings.Contains(repoPath, ":") {
		repoPath = strings.Split(repoPath, ":")[0]
	}

	return repoPath, nil
}

// IsECRAddress returns whether the given address is an ECR address.
func IsECRRegistry(registryURL string) bool {
	// Match pattern: <account>.dkr.ecr.<region>.amazonaws.com
	// where account is 12 digits and region is a valid AWS region format
	ecrPattern := regexp.MustCompile(`^\d{12}\.dkr\.ecr\.[a-z0-9-]+\.amazonaws\.com`)

	cleanURL := registryURL
	if strings.Contains(cleanURL, "://") {
		parts := strings.SplitN(cleanURL, "://", 2)
		cleanURL = parts[1]
	}

	if strings.Contains(cleanURL, "/") {
		cleanURL = strings.SplitN(cleanURL, "/", 2)[0]
	}

	return ecrPattern.MatchString(cleanURL)
}
