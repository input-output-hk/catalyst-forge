package providers

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
)

func TestAWSClientGet(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		mock        SecretsManagerClientMock
		expect      string
		expectErr   bool
		expectedErr error
		cond        func(*SecretsManagerClientMock) error
	}{
		{
			name: "simple",
			path: "path",
			mock: SecretsManagerClientMock{
				GetSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					return &secretsmanager.GetSecretValueOutput{
						SecretString: aws.String("secret"),
					}, nil
				},
			},
			expect:      "secret",
			expectErr:   false,
			expectedErr: nil,
			cond: func(m *SecretsManagerClientMock) error {
				if len(m.calls.GetSecretValue) != 1 {
					return fmt.Errorf("expected GetSecretValue to be called once, got %d", len(m.calls.GetSecretValue))
				}

				if *m.calls.GetSecretValue[0].Params.SecretId != "path" {
					return fmt.Errorf("expected GetSecretValue to be called with path, got %s", *m.calls.GetSecretValue[0].Params.SecretId)
				}

				return nil
			},
		},
		{
			name: "error",
			mock: SecretsManagerClientMock{
				GetSecretValueFunc: func(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
					return nil, fmt.Errorf("error")
				},
			},
			expect:      "",
			expectErr:   true,
			expectedErr: fmt.Errorf("unable to get secret: error"),
		},
	}

	for i := range tests {
		tt := &tests[i] // Required to avoid copying the generaetd RWMutex
		t.Run(tt.name, func(t *testing.T) {
			client := &AWSClient{
				client: &tt.mock,
				logger: testutils.NewNoopLogger(),
			}

			got, err := client.Get(tt.path)

			ret, err := testutils.CheckError(t, err, tt.expectErr, tt.expectedErr)
			if err != nil {
				t.Error(err)
				return
			} else if ret {
				return
			}

			if tt.cond != nil {
				if err := tt.cond(&tt.mock); err != nil {
					t.Error(err)
					return
				}
			}

			if got != tt.expect {
				t.Errorf("expected: %s, got: %s", tt.expect, got)
			}
		})
	}
}

func TestAWSClientSet(t *testing.T) {
	tests := []struct {
		name        string
		mock        SecretsManagerClientMock
		expect      string
		expectErr   bool
		expectedErr error
		cond        func(*SecretsManagerClientMock) error
	}{
		{
			name: "simple",
			mock: SecretsManagerClientMock{
				CreateSecretFunc: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
					return &secretsmanager.CreateSecretOutput{
						VersionId: aws.String("version"),
					}, nil
				},
			},
			expect:      "version",
			expectErr:   false,
			expectedErr: nil,
			cond: func(m *SecretsManagerClientMock) error {
				if len(m.calls.CreateSecret) != 1 {
					return fmt.Errorf("expected CreateSecret to be called once, got %d", len(m.calls.CreateSecret))
				}

				if *m.calls.CreateSecret[0].Params.Name != "path" {
					return fmt.Errorf("expected CreateSecret to be called with path, got %s", *m.calls.CreateSecret[0].Params.Name)
				}

				return nil
			},
		},
		{
			name: "secret already exists",
			mock: SecretsManagerClientMock{
				CreateSecretFunc: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
					return nil, AWSSecretsManagerResourceExistsException
				},
				PutSecretValueFunc: func(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
					return &secretsmanager.PutSecretValueOutput{
						VersionId: aws.String("version"),
					}, nil
				},
			},
			expect:      "version",
			expectErr:   false,
			expectedErr: nil,
			cond: func(m *SecretsManagerClientMock) error {
				if len(m.calls.CreateSecret) != 1 {
					return fmt.Errorf("expected CreateSecret to be called once, got %d", len(m.calls.CreateSecret))
				}

				if *m.calls.CreateSecret[0].Params.Name != "path" {
					return fmt.Errorf("expected CreateSecret to be called with path, got %s", *m.calls.CreateSecret[0].Params.Name)
				}

				if len(m.calls.PutSecretValue) != 1 {
					return fmt.Errorf("expected PutSecretValue to be called once, got %d", len(m.calls.PutSecretValue))
				}

				if *m.calls.PutSecretValue[0].Params.SecretId != "path" {
					return fmt.Errorf("expected PutSecretValue to be called with path, got %s", *m.calls.PutSecretValue[0].Params.SecretId)
				}

				return nil
			},
		},
		{
			name: "error creating secret",
			mock: SecretsManagerClientMock{
				CreateSecretFunc: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
					return nil, fmt.Errorf("error")
				},
			},
			expect:      "",
			expectErr:   true,
			expectedErr: fmt.Errorf("unable to set secret: error"),
		},
		{
			name: "error putting secret value",
			mock: SecretsManagerClientMock{
				CreateSecretFunc: func(ctx context.Context, params *secretsmanager.CreateSecretInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.CreateSecretOutput, error) {
					return nil, AWSSecretsManagerResourceExistsException
				},
				PutSecretValueFunc: func(ctx context.Context, params *secretsmanager.PutSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.PutSecretValueOutput, error) {
					return nil, fmt.Errorf("error")
				},
			},
			expect:      "",
			expectErr:   true,
			expectedErr: fmt.Errorf("unable to set secret: error"),
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			client := &AWSClient{
				client: &tt.mock,
				logger: testutils.NewNoopLogger(),
			}

			got, err := client.Set("path", "value")

			ret, err := testutils.CheckError(t, err, tt.expectErr, tt.expectedErr)
			if err != nil {
				t.Error(err)
				return
			} else if ret {
				return
			}

			if tt.cond != nil {
				if err := tt.cond(&tt.mock); err != nil {
					t.Error(err)
					return
				}
			}

			if got != tt.expect {
				t.Errorf("expected: %s, got: %s", tt.expect, got)
			}
		})
	}
}