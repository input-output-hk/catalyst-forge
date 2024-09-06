package earthly

import (
	"fmt"
	"log/slog"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/input-output-hk/catalyst-forge/blueprint/pkg/utils"
	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
)

func TestEarthlyExecutorRun(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		earthlyExec EarthlyExecutor
		mockExec    executor.ExecutorMock
		expect      map[string]EarthlyExecutionResult
		expectCalls int
		expectErr   bool
	}{
		{
			name: "simple",
			earthlyExec: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
			),
			mockExec: executor.ExecutorMock{
				ExecuteFunc: func(command string, args []string) ([]byte, error) {
					return []byte(`foobarbaz
Image foo output as bar
Artifact foo output as bar`), nil
				},
			},
			expect: map[string]EarthlyExecutionResult{
				getNativePlatform(): {
					Images: map[string]string{
						"foo": "bar",
					},
					Artifacts: map[string]string{
						"foo": "bar",
					},
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
			mockExec: executor.ExecutorMock{
				ExecuteFunc: func(command string, args []string) ([]byte, error) {
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
			mockExec: executor.ExecutorMock{
				ExecuteFunc: func(command string, args []string) ([]byte, error) {
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
			got, err := tt.earthlyExec.Run()

			if tt.expectErr && err == nil {
				t.Errorf("expected error, got nil")
			} else if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(tt.mockExec.ExecuteCalls()) != tt.expectCalls {
				t.Errorf("expected %d calls to Execute, got %d", tt.expectCalls, len(tt.mockExec.ExecuteCalls()))
			}

			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("expected %v, got %v", tt.expect, got)
			}
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
			platform: getNativePlatform(),
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
			platform: getNativePlatform(),
			expect:   []string{"/test/dir+foo", "--arg1", "foo", "--arg2", "bar"},
		},
		{
			name: "with artifact",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithArtifact("test"),
			),
			platform: getNativePlatform(),
			expect:   []string{"--artifact", "/test/dir+foo/*", "test/"},
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
			name: "with ci",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithCI(),
			),
			platform: getNativePlatform(),
			expect:   []string{"--ci", "/test/dir+foo"},
		},
		{
			name: "with privileged",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithPrivileged(),
			),
			platform: getNativePlatform(),
			expect:   []string{"--allow-privileged", "/test/dir+foo"},
		},
		{
			name: "with satellite",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithSatellite("satellite"),
			),
			platform: getNativePlatform(),
			expect:   []string{"--sat", "satellite", "/test/dir+foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.e.buildArguments(tt.platform)
			if !slices.Equal(got, tt.expect) {
				t.Errorf("expected %v, got %v", tt.expect, got)
			}
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
		expectedErr error
	}{
		{
			name: "simple",
			provider: &secrets.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return `{"key": "value"}`, nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     utils.StringPtr("path"),
					Provider: utils.StringPtr("mock"),
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
			expectedErr: nil,
		},
		{
			name: "key does not exist",
			provider: &secrets.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return `{"key": "value"}`, nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     utils.StringPtr("path"),
					Provider: utils.StringPtr("mock"),
					Maps: map[string]string{
						"key1": "id1",
					},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: fmt.Errorf("secret key not found in secret values: key1"),
		},
		{
			name: "invalid JSON",
			provider: &secrets.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return `invalid`, nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     utils.StringPtr("path"),
					Provider: utils.StringPtr("mock"),
					Maps:     map[string]string{},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: fmt.Errorf("unable to unmarshal secret value: invalid character 'i' looking for beginning of value"),
		},
		{
			name: "secret provider does not exist",
			provider: &secrets.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return "", nil
				},
			},
			secrets: []schema.Secret{
				{
					Path:     utils.StringPtr("path"),
					Provider: utils.StringPtr("bad"),
					Maps:     map[string]string{},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: fmt.Errorf("unable to create new secret client: unknown secret provider: bad"),
		},
		{
			name: "secret provider error",
			provider: &secrets.SecretProviderMock{
				GetFunc: func(path string) (string, error) {
					return "", fmt.Errorf("error")
				},
			},
			secrets: []schema.Secret{
				{
					Path:     utils.StringPtr("path"),
					Provider: utils.StringPtr("mock"),
					Maps:     map[string]string{},
				},
			},
			expect:      nil,
			expectErr:   true,
			expectedErr: fmt.Errorf("unable to get secret path from provider: mock"),
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

			ret, err := testutils.CheckError(t, err, tt.expectErr, tt.expectedErr)
			if err != nil {
				t.Error(err)
				return
			} else if ret {
				return
			}

			if !slices.Equal(got, tt.expect) {
				t.Errorf("expected %v, got %v", tt.expect, got)
			}
		})
	}
}

func Test_parseOutput(t *testing.T) {
	tests := []struct {
		expect EarthlyExecutionResult
		name   string
		output string
	}{
		{
			name: "simple",
			output: `foobarbaz
Image foo output as bar
Artifact foo output as bar`,
			expect: EarthlyExecutionResult{
				Images: map[string]string{
					"foo": "bar",
				},
				Artifacts: map[string]string{
					"foo": "bar",
				},
			},
		},
		{
			name:   "no output",
			output: "",
			expect: EarthlyExecutionResult{
				Images:    map[string]string{},
				Artifacts: map[string]string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseResult(tt.output)
			if !maps.Equal(got.Images, tt.expect.Images) {
				t.Errorf("expected %v, got %v", tt.expect.Images, got.Images)
			}
			if !maps.Equal(got.Artifacts, tt.expect.Artifacts) {
				t.Errorf("expected %v, got %v", tt.expect.Artifacts, got.Artifacts)
			}
		})
	}
}
