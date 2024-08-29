package providers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go/aws"
)

//go:generate go run github.com/matryer/moq@latest -out aws_mock_test.go . SecretsManagerClient

var (
	AWSSecretsManagerResourceExistsException *smtypes.ResourceExistsException
	AWSSecretsManagerInvalidRequestException *smtypes.InvalidRequestException
)

// SecretsManagerClient is an interface for the AWS Secrets Manager client.
type SecretsManagerClient interface {
	CreateSecret(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error)
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
	PutSecretValue(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error)
}

// AWSClient is a client for interacting with AWS Secrets Manager.
type AWSClient struct {
	logger *slog.Logger
	client SecretsManagerClient
}

// Get retrieves a secret from AWS Secrets Manager.
func (c *AWSClient) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.logger.Debug("Getting secret from AWS Secretsmanager", "key", key)

	resp, err := c.client.GetSecretValue(
		ctx,
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(key),
		},
	)
	if err != nil {
		return "", fmt.Errorf("unable to get secret: %w", err)
	}

	return *resp.SecretString, nil
}

// Set sets a secret in AWS Secrets Manager.
func (c *AWSClient) Set(key, value string) (string, error) {
	var (
		err       error
		versionId string
	)

	resp, err := c.createSecret(key, value)
	if err != nil {
		if errors.As(err, &AWSSecretsManagerResourceExistsException) {
			c.logger.Warn("Secret already exists. Creating new secret version.")
			resp, err := c.putSecretValue(key, value)
			if err != nil {
				return "", fmt.Errorf("unable to set secret: %w", err)
			}

			versionId = *resp.VersionId
		}
		if errors.As(err, &AWSSecretsManagerInvalidRequestException) {
			return "", fmt.Errorf("invalid request: %w", err)
		}

		if versionId == "" {
			return "", fmt.Errorf("unable to set secret: %w", err)
		}
	}

	if versionId == "" {
		versionId = *resp.VersionId
	}

	c.logger.Info("Successfully set secret using AWS Secretsmanager provider", "versionId", versionId)

	return versionId, nil
}

// createSecret creates a new secret in AWS Secrets Manager.
func (c *AWSClient) createSecret(key, value string) (*secretsmanager.CreateSecretOutput, error) {
	params := &secretsmanager.CreateSecretInput{
		Name:         aws.String(key),
		SecretString: aws.String(value),
		Tags: []smtypes.Tag{
			{
				Key:   aws.String("CreatedBy"),
				Value: aws.String("Forge"),
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.CreateSecret(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// putSecretValue creates a new version of a secret in AWS Secrets Manager.
func (c *AWSClient) putSecretValue(key, value string) (*secretsmanager.PutSecretValueOutput, error) {
	params := &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String(key),
		SecretString: aws.String(value),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.PutSecretValue(ctx, params)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// NewAWSClient creates a new AWSClient with the default configuration.
func NewDefaultAWSClient(logger *slog.Logger) (*AWSClient, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("eu-central-1"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	return &AWSClient{
		logger: logger,
		client: secretsmanager.NewFromConfig(cfg),
	}, nil
}

// NewAWSClient creates a new AWSClient.
func NewAWSClient(client SecretsManagerClient, logger *slog.Logger) *AWSClient {
	return &AWSClient{
		logger: logger,
		client: client,
	}
}
