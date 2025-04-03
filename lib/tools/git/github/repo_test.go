package github

import (
	"os"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubRepoGetBranch(t *testing.T) {
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

			gh := DefaultGithubRepo{}
			tt.validate(t, gh.GetBranch())
		})
	}
}

func TestGithubRepoGetCommit(t *testing.T) {
	prPayload, err := os.ReadFile("testdata/event_pr.json")
	require.NoError(t, err)

	pushPayload, err := os.ReadFile("testdata/event_push.json")
	require.NoError(t, err)

	tests := []struct {
		name     string
		env      map[string]string
		files    map[string]string
		validate func(*testing.T, string, error)
	}{
		{
			name: "pull request",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "pull_request",
				"GITHUB_EVENT_PATH": "/event.json",
			},
			files: map[string]string{
				"/event.json": string(prPayload),
			},
			validate: func(t *testing.T, commit string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "0000000000000000000000000000000000000000", commit)
			},
		},
		{
			name: "push",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_EVENT_PATH": "/event.json",
			},
			files: map[string]string{
				"/event.json": string(pushPayload),
			},
			validate: func(t *testing.T, commit string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "0000000000000000000000000000000000000000", commit)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutils.NewNoopLogger()

			fs := billy.NewInMemoryFs()
			testutils.SetupFS(t, fs, tt.files)

			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
				defer os.Unsetenv(k)
			}

			gh := DefaultGithubRepo{
				fs:     fs,
				logger: logger,
			}
			commit, err := gh.GetCommit()
			tt.validate(t, commit, err)
		})
	}
}

func TestGithubRepoGetTag(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, string, bool)
	}{
		{
			name: "tag",
			env: map[string]string{
				"GITHUB_REF": "refs/tags/v1.0.0",
			},
			validate: func(t *testing.T, tag string, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, "v1.0.0", tag)
			},
		},
		{
			name: "no tag",
			env: map[string]string{
				"GITHUB_REF": "refs/heads/feature/branch",
			},
			validate: func(t *testing.T, tag string, ok bool) {
				assert.False(t, ok)
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

			gh := DefaultGithubRepo{}
			tag, ok := gh.GetTag()
			tt.validate(t, tag, ok)
		})
	}
}

func TestInGithubActions(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		validate func(*testing.T, bool)
	}{
		{
			name: "has event",
			env: map[string]string{
				"GITHUB_EVENT_PATH": "/path/to/event",
				"GITHUB_EVENT_NAME": "push",
			},
			validate: func(t *testing.T, exists bool) {
				assert.True(t, exists)
			},
		},
		{
			name: "missing path",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "push",
			},
			validate: func(t *testing.T, exists bool) {
				assert.False(t, exists)
			},
		},
		{
			name: "missing name",
			env: map[string]string{
				"GITHUB_EVENT_PATH": "/path/to/event",
			},
			validate: func(t *testing.T, exists bool) {
				assert.False(t, exists)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				require.NoError(t, os.Setenv(k, v))
				defer os.Unsetenv(k)
			}

			tt.validate(t, InGithubActions())
		})
	}
}
