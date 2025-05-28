package github

import (
	"os"
	"testing"

	"github.com/google/go-github/v66/github"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubEnvGetBranch(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, string)
	}{
		{
			name: "head ref",
			env: map[string]string{
				"GITHUB_HEAD_REF": "feature/branch",
			},
			validate: func(t *testing.T, branch string) {
				assert.Equal(t, "feature/branch", branch)
			},
		},
		{
			name: "ref",
			env: map[string]string{
				"GITHUB_REF": "refs/heads/feature/branch",
			},
			validate: func(t *testing.T, branch string) {
				assert.Equal(t, "feature/branch", branch)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
				defer os.Unsetenv(k)
			}

			gh := GithubEnv{}
			tt.validate(t, gh.GetBranch())
		})
	}
}

func TestGithubEnvGetEventPayload(t *testing.T) {
	payload, err := os.ReadFile("testdata/event.json")
	require.NoError(t, err)

	tests := []struct {
		name     string
		env      map[string]string
		files    map[string]string
		validate func(*testing.T, any, error)
	}{
		{
			name: "full",
			env: map[string]string{
				"GITHUB_EVENT_PATH": "/event.json",
				"GITHUB_EVENT_NAME": "pull_request",
			},
			files: map[string]string{
				"/event.json": string(payload),
			},
			validate: func(t *testing.T, payload any, err error) {
				require.NoError(t, err)
				_, ok := payload.(*github.PullRequestEvent)
				require.True(t, ok)
			},
		},
		{
			name: "missing path",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "pull_request",
			},
			validate: func(t *testing.T, payload any, err error) {
				require.ErrorIs(t, err, ErrNoEventFound)
			},
		},
		{
			name: "missing name",
			env: map[string]string{
				"GITHUB_EVENT_PATH": "/event.json",
			},
			files: map[string]string{
				"/event.json": string(payload),
			},
			validate: func(t *testing.T, payload any, err error) {
				require.ErrorIs(t, err, ErrNoEventFound)
			},
		},
		{
			name: "invalid payload",
			env: map[string]string{
				"GITHUB_EVENT_PATH": "/event.json",
				"GITHUB_EVENT_NAME": "pull_request",
			},
			files: map[string]string{
				"/event.json": "invalid",
			},
			validate: func(t *testing.T, payload any, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
				defer os.Unsetenv(k)
			}

			fs := billy.NewInMemoryFs()
			testutils.SetupFS(t, fs, tt.files)

			gh := GithubEnv{
				fs:     fs,
				logger: testutils.NewNoopLogger(),
			}

			payload, err := gh.GetEventPayload()
			tt.validate(t, payload, err)
		})
	}
}

func TestGithubEnvGetEventType(t *testing.T) {
	gh := GithubEnv{}

	require.NoError(t, os.Setenv("GITHUB_EVENT_NAME", "push"))
	assert.Equal(t, "push", gh.GetEventType())
}

func TestGithubEnvGetTag(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, string)
	}{
		{
			name: "tag",
			env: map[string]string{
				"GITHUB_REF": "refs/tags/v1.0.0",
			},
			validate: func(t *testing.T, tag string) {
				assert.Equal(t, "v1.0.0", tag)
			},
		},
		{
			name: "no tag",
			env: map[string]string{
				"GITHUB_REF": "refs/heads/feature/branch",
			},
			validate: func(t *testing.T, tag string) {
				assert.Empty(t, tag)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
				defer os.Unsetenv(k)
			}

			gh := GithubEnv{}
			tt.validate(t, gh.GetTag())
		})
	}
}

func TestGithubEnvHasEvent(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		expect bool
	}{
		{
			name: "has event",
			env: map[string]string{
				"GITHUB_EVENT_PATH": "/path/to/event",
				"GITHUB_EVENT_NAME": "push",
			},
			expect: true,
		},
		{
			name: "missing path",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "push",
			},
			expect: false,
		},
		{
			name: "missing name",
			env: map[string]string{
				"GITHUB_EVENT_PATH": "/path/to/event",
			},
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
				defer os.Unsetenv(k)
			}

			gh := GithubEnv{}
			assert.Equal(t, tt.expect, gh.HasEvent())
		})
	}
}
