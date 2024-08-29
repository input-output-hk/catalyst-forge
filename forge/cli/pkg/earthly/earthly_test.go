package earthly

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"testing"

	"github.com/input-output-hk/catalyst-forge/blueprint/schema"
	"github.com/input-output-hk/catalyst-forge/forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/executor"
	"github.com/input-output-hk/catalyst-forge/forge/cli/pkg/secrets"
)

func TestEarthlyExecutorRun(t *testing.T) {
	tests := []struct {
		expect      EarthlyExecutionResult
		name        string
		output      string
		earthlyExec EarthlyExecutor
		mockExec    executor.ExecutorMock
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
			expect: EarthlyExecutionResult{
				Images: map[string]string{
					"foo": "bar",
				},
				Artifacts: map[string]string{
					"foo": "bar",
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
			expect: EarthlyExecutionResult{
				Images:    map[string]string{},
				Artifacts: map[string]string{},
			},
			expectErr:   true,
			expectCalls: 4,
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

			if !maps.Equal(got.Artifacts, tt.expect.Artifacts) {
				t.Errorf("expected %v, got %v", tt.expect.Artifacts, got.Artifacts)
			}

			if !maps.Equal(got.Images, tt.expect.Images) {
				t.Errorf("expected %v, got %v", tt.expect.Images, got.Images)
			}
		})
	}
}

func TestEarthlyExecutor_buildArguments(t *testing.T) {
	tests := []struct {
		name   string
		e      EarthlyExecutor
		expect []string
	}{
		{
			name: "simple",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
			),
			expect: []string{"/test/dir+foo"},
		},
		{
			name: "with target args",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithTargetArgs("--arg1", "foo", "--arg2", "bar"),
			),
			expect: []string{"/test/dir+foo", "--arg1", "foo", "--arg2", "bar"},
		},
		{
			name: "with artifact",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithArtifact(),
			),
			expect: []string{"--artifact", "/test/dir+foo/*"},
		},
		{
			name: "with privileged",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithPrivileged(),
			),
			expect: []string{"--allow-privileged", "/test/dir+foo"},
		},
		{
			name: "with satellite",
			e: NewEarthlyExecutor("/test/dir", "foo", nil, secrets.SecretStore{},
				testutils.NewNoopLogger(),
				WithSatellite("satellite"),
			),
			expect: []string{"--sat", "satellite", "/test/dir+foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.e.buildArguments()
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
					Path:     "path",
					Provider: "mock",
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
					Path:     "path",
					Provider: "mock",
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
					Path:     "path",
					Provider: "bad",
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
					Path:     "path",
					Provider: "mock",
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

// var data string = "{\"secret_key\":\"secret_value\"}"

// func setup(t *testing.T) (*slog.Logger, string) {
// 	t.Helper()

// 	handler := log.New(os.Stderr)
// 	handler.SetLevel(log.DebugLevel)
// 	logger := slog.New(handler)

// 	f, err := os.CreateTemp("", "tmpfile-")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer f.Close()

// 	b := []byte(data)
// 	if _, err := f.Write(b); err != nil {
// 		log.Fatal(err)
// 	}

// 	return logger, f.Name()
// }

// func TestCreateMapFromSecrets(t *testing.T) {
// 	logger, tmpFile := setup(t)

// 	t.Cleanup(func() {
// 		os.Remove(tmpFile)
// 	})

// 	prefix := "path/to/earthfile"

// 	c := &cue.Config{
// 		Secrets: map[string]*cue.Secret{
// 			"my_secret": {
// 				Provider: "local",
// 				Path:     tmpFile,
// 				Maps: map[string]string{
// 					"secret_key": "secret_id_for_earthly",
// 				},
// 			},
// 		},
// 	}

// 	var secrets []*cue.Secret

// 	for _, v := range c.Secrets {
// 		secrets = append(secrets, v)
// 	}

// 	got, err := newSecrets(logger, secrets, prefix)

// 	want := []*Secret{
// 		{
// 			Value: "secret_value",
// 		},
// 	}

// 	assert.Nil(t, err)
// 	assert.NotNil(t, got)
// 	assert.Equal(t, want[0].Value, got[0].Value)
// }
