package providers

// import (
// 	"os"
// 	"testing"

// 	"github.com/google/go-github/v66/github"
// 	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
// 	"github.com/spf13/afero"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// )

// func TestGithubProviderGetEventPayload(t *testing.T) {
// 	payload, err := os.ReadFile("testdata/event.json")
// 	require.NoError(t, err)

// 	tests := []struct {
// 		name     string
// 		env      map[string]string
// 		files    map[string]string
// 		validate func(*testing.T, any, error)
// 	}{
// 		{
// 			name: "full",
// 			env: map[string]string{
// 				"GITHUB_EVENT_PATH": "/event.json",
// 				"GITHUB_EVENT_NAME": "pull_request",
// 			},
// 			files: map[string]string{
// 				"/event.json": string(payload),
// 			},
// 			validate: func(t *testing.T, payload any, err error) {
// 				require.NoError(t, err)
// 				_, ok := payload.(*github.PullRequestEvent)
// 				require.True(t, ok)
// 			},
// 		},
// 		{
// 			name: "missing path",
// 			env: map[string]string{
// 				"GITHUB_EVENT_NAME": "pull_request",
// 			},
// 			validate: func(t *testing.T, payload any, err error) {
// 				require.ErrorIs(t, err, ErrNoEventFound)
// 			},
// 		},
// 		{
// 			name: "missing name",
// 			env: map[string]string{
// 				"GITHUB_EVENT_PATH": "/event.json",
// 			},
// 			files: map[string]string{
// 				"/event.json": string(payload),
// 			},
// 			validate: func(t *testing.T, payload any, err error) {
// 				require.ErrorIs(t, err, ErrNoEventFound)
// 			},
// 		},
// 		{
// 			name: "invalid payload",
// 			env: map[string]string{
// 				"GITHUB_EVENT_PATH": "/event.json",
// 				"GITHUB_EVENT_NAME": "pull_request",
// 			},
// 			files: map[string]string{
// 				"/event.json": "invalid",
// 			},
// 			validate: func(t *testing.T, payload any, err error) {
// 				require.Error(t, err)
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			for k, v := range tt.env {
// 				require.NoError(t, os.Setenv(k, v))
// 				defer os.Unsetenv(k)
// 			}

// 			fs := afero.NewMemMapFs()
// 			testutils.SetupFS(t, fs, tt.files)

// 			provider := GithubProvider{
// 				fs:     fs,
// 				logger: testutils.NewNoopLogger(),
// 			}

// 			payload, err := provider.GetEventPayload()
// 			tt.validate(t, payload, err)
// 		})
// 	}
// }

// func TestGithubProviderGetEventType(t *testing.T) {
// 	provider := GithubProvider{}

// 	require.NoError(t, os.Setenv("GITHUB_EVENT_NAME", "push"))
// 	assert.Equal(t, "push", provider.GetEventType())
// }

// func TestGithubProviderHasEvent(t *testing.T) {
// 	tests := []struct {
// 		name   string
// 		env    map[string]string
// 		expect bool
// 	}{
// 		{
// 			name: "has event",
// 			env: map[string]string{
// 				"GITHUB_EVENT_PATH": "/path/to/event",
// 				"GITHUB_EVENT_NAME": "push",
// 			},
// 			expect: true,
// 		},
// 		{
// 			name: "missing path",
// 			env: map[string]string{
// 				"GITHUB_EVENT_NAME": "push",
// 			},
// 			expect: false,
// 		},
// 		{
// 			name: "missing name",
// 			env: map[string]string{
// 				"GITHUB_EVENT_PATH": "/path/to/event",
// 			},
// 			expect: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			for k, v := range tt.env {
// 				require.NoError(t, os.Setenv(k, v))
// 				defer os.Unsetenv(k)
// 			}

// 			provider := GithubProvider{}
// 			assert.Equal(t, tt.expect, provider.HasEvent())
// 		})
// 	}
// }
