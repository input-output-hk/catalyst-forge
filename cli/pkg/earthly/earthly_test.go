package earthly

import (
	"fmt"
	"log/slog"
	"testing"

	emocks "github.com/input-output-hk/catalyst-forge/cli/pkg/executor/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	smocks "github.com/input-output-hk/catalyst-forge/lib/project/secrets/mocks"
	"github.com/input-output-hk/catalyst-forge/lib/project/utils"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestEarthlyExecutorRun(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		earthlyExec EarthlyExecutor
		mockExec    emocks.ExecutorMock
		expect      map[string]EarthlyExecutionResult
		expectCalls int
		expectErr   bool
	}{
		{
			name: "simple",
			earthlyExec: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
			),
			mockExec: emocks.ExecutorMock{
				ExecuteFunc: func(command string, args ...string) ([]byte, error) {
					return []byte(`foobarbaz
Image foo output as bar
Artifact foo output as bar`), nil
				},
			},
			expectErr:   false,
			expectCalls: 1,
		},
		{
			name: "with retries",
			earthlyExec: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithRetries(3),
			),
			mockExec: emocks.ExecutorMock{
				ExecuteFunc: func(command string, args ...string) ([]byte, error) {
					return []byte{}, fmt.Errorf("error")
				},
			},
			expect:      nil,
			expectErr:   true,
			expectCalls: 4,
		},
		{
			name: "with platforms",
			earthlyExec: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithPlatforms("foo", "bar"),
			),
			mockExec: emocks.ExecutorMock{
				ExecuteFunc: func(command string, args ...string) ([]byte, error) {
					return []byte(`foobarbaz
Image foo output as bar
Artifact foo output as bar`), nil
				},
			},
			expect: map[string]EarthlyExecutionResult{
				"foo": {
					Images: map[string]string{
						"foo": "bar",
					},
					Artifacts: map[string]string{
						"foo": "bar",
					},
				},
				"bar": {
					Images: map[string]string{
						"foo": "bar",
					},
					Artifacts: map[string]string{
						"foo": "bar",
					},
				},
			},
			expectErr:   false,
			expectCalls: 2,
		},
	}

	for i := range tests {
		tt := &tests[i] // Required to avoid copying the generaetd RWMutex
		t.Run(tt.name, func(t *testing.T) {
			tt.earthlyExec.executor = &tt.mockExec
			err := tt.earthlyExec.Run()

			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			assert.Equal(t, len(tt.mockExec.ExecuteCalls()), tt.expectCalls)
		})
	}
}

func TestEarthlyExecutor_buildArguments(t *testing.T) {
	tests := []struct {
		name     string
		e        EarthlyExecutor
		platform string
		expect   []string
	}{
		{
			name: "simple",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
			),
			platform: GetBuildPlatform(),
			expect:   []string{"/test/dir+foo"},
		},
		{
			name: "with platform",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
			),
			platform: "foo/bar",
			expect:   []string{"--platform", "foo/bar", "/test/dir+foo"},
		},
		{
			name: "with target args",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithTargetArgs("--arg1", "foo", "--arg2", "bar"),
			),
			platform: GetBuildPlatform(),
			expect:   []string{"/test/dir+foo", "--arg1", "foo", "--arg2", "bar"},
		},
		{
			name: "with artifact",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithArtifact("test"),
			),
			platform: "linux/amd65",
			expect:   []string{"--platform", "linux/amd65", "--artifact", "/test/dir+foo/*", "test/linux/amd65/"},
		},
		{
			name: "with artifact and platforms",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithPlatforms("foo", "bar"),
				WithArtifact("test"),
			),
			platform: "foo",
			expect:   []string{"--platform", "foo", "--artifact", "/test/dir+foo/*", "test/foo/"},
		},
		{
			name: "with privileged",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithPrivileged(),
			),
			platform: GetBuildPlatform(),
			expect:   []string{"--allow-privileged", "/test/dir+foo"},
		},
		{
			name: "with satellite",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithSatellite("satellite"),
			),
			platform: GetBuildPlatform(),
			expect:   []string{"--sat", "satellite", "/test/dir+foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.e.buildArguments(tt.platform)
			assert.Equal(t, tt.expect, got)
		})
	}
}

func TestEarthlyExecutor_buildSecrets(t *testing.T) {
	tests := []struct {
		name        string
		provider    secrets.SecretProvider
		secrets     []schema.Secret
		expect      []EarthlySecret
		expectErr   bool
		expectedErr string
	}{
		{
			name: "simple",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return `{"key": "value"}`, nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     "path",
					Provider: "mock",
					Maps: map[string]string{
						"key": "id",
					},
				},
			},
			expect: []EarthlySecret{
				{
					Id:    "id",
					Value: "value",
				},
			},
			expectErr:   false,
			expectedErr: "",
		},
		{
			name: "no JSON",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return "secret", nil
				},
			},
			secrets: []schema.Secret{
				{
					Name:     utils.StringPtr("name"),
					Path:     "path",
					Provider: "mock",
					Maps:     map[string]string{},
				},
			},
			expect: []EarthlySecret{
				{
					Id:    "name",
					Value: "secret",
				},
			},
			expectErr:   false,
			expectedErr: "",
		},
		{
			name: "optional secret",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return "", fmt.Errorf("not found")
				},
			},
			secrets: []schema.Secret{
				{
					Name:     utils.StringPtr("name"),
					Optional: utils.BoolPtr(true),
					Path:     "path",
					Provider: "mock",
					Maps:     map[string]string{},
				},
			},
			expect:    nil,
			expectErr: false,
		},
		{
			name: "name and maps defined",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return "", nil
				},
			},
			secrets: []schema.Secret{
				{
					Name:     utils.StringPtr("name"),
					Path:     "path",
					Provider: "mock",
					Maps: map[string]string{
						"key": "id",
					},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: "secret contains both name and maps: name",
		},
		{
			name: "key does not exist",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return `{"key": "value"}`, nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     "path",
					Provider: "mock",
					Maps: map[string]string{
						"key1": "id1",
					},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: "secret key not found in secret values: key1",
		},
		{
			name: "invalid JSON",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return `invalid`, nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     "path",
					Provider: "mock",
					Maps: map[string]string{
						"key1": "id1",
					},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: "failed to unmarshal secret values from provider mock: invalid character 'i' looking for beginning of value",
		},
		{
			name: "secret provider does not exist",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return "", nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     "path",
					Provider: "bad",
					Maps:     map[string]string{},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: "unable to create new secret client: unknown secret provider: bad",
		},
		{
			name: "secret provider error",
			provider: &smocks.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return "", fmt.Errorf("error")
				},
			},
			secrets: []schema.Secret{
				{
					Path:     "path",
					Provider: "mock",
					Maps:     map[string]string{},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: "unable to get secret path from provider: mock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := secrets.NewSecretStore(map[secrets.Provider]func(*slog.Logger) (secrets.SecretProvider, error){
				secrets.Provider("mock"): func(logger *slog.Logger) (secrets.SecretProvider, error) {
					return tt.provider, nil
				},
			})

			executor := NewEarthlyExecutor("", "", nil, store, testutils.NewNoopLogger())
			executor.secrets = tt.secrets
			got, err := executor.buildSecrets()

			if testutils.AssertError(t, err, tt.expectErr, tt.expectedErr) {
				return
			}

			assert.Equal(t, tt.expect, got)
		})
	}
}
