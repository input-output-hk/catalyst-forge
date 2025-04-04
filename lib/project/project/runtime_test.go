package project

import (
	"fmt"
	"os"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	lc "github.com/input-output-hk/catalyst-forge/lib/tools/cue"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs/billy"
	"github.com/input-output-hk/catalyst-forge/lib/tools/git/repo"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeploymentRuntimeLoad(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name        string
		projectName string
		registry    string
		repo        string
		validate    func(*testing.T, map[string]cue.Value)
	}{
		{
			name:        "full",
			projectName: "test",
			registry:    "test-registry",
			repo:        "test-repo",
			validate: func(t *testing.T, data map[string]cue.Value) {
				assert.Contains(t, data, "CONTAINER_IMAGE")
				assert.Equal(t, "test-registry/test-repo/test", getString(t, data["CONTAINER_IMAGE"]))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := fmt.Sprintf(`
				project: name: "%s"
				global: {
					deployment: registries: containers: "%s"
					repo: name: "%s"
				}
			`, tt.name, tt.registry, tt.repo)

			v, err := lc.Compile(ctx, []byte(rv))
			require.NoError(t, err)

			p := &Project{
				ctx:          ctx,
				Name:         tt.projectName,
				RawBlueprint: blueprint.NewRawBlueprint(v),
			}

			runtime := &DeploymentRuntime{logger: testutils.NewNoopLogger()}
			data := runtime.Load(p)
			tt.validate(t, data)
		})
	}
}

func TestGitRuntimeLoad(t *testing.T) {
	ctx := cuecontext.New()
	prPayload, err := os.ReadFile("testdata/event_pr.json")
	require.NoError(t, err)

	pushPayload, err := os.ReadFile("testdata/event_push.json")
	require.NoError(t, err)

	tests := []struct {
		name     string
		tag      *ProjectTag
		env      map[string]string
		files    map[string]string
		validate func(*testing.T, repo.GitRepo, map[string]cue.Value)
	}{
		{
			name: "with tag",
			tag: &ProjectTag{
				Full:    "project/v1.0.0",
				Project: "project",
				Version: "v1.0.0",
			},
			validate: func(t *testing.T, repo repo.GitRepo, data map[string]cue.Value) {
				head, err := repo.Head()
				require.NoError(t, err)
				assert.Contains(t, data, "GIT_COMMIT_HASH")
				assert.Equal(t, head.Hash().String(), getString(t, data["GIT_COMMIT_HASH"]))

				assert.Contains(t, data, "GIT_TAG")
				assert.Equal(t, "project/v1.0.0", getString(t, data["GIT_TAG"]))

				assert.Contains(t, data, "GIT_TAG_VERSION")
				assert.Equal(t, "v1.0.0", getString(t, data["GIT_TAG_VERSION"]))
			},
		},
		{
			name: "with pr event",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "pull_request",
				"GITHUB_EVENT_PATH": "/event.json",
			},
			files: map[string]string{
				"/event.json": string(prPayload),
			},
			validate: func(t *testing.T, repo repo.GitRepo, data map[string]cue.Value) {
				require.NoError(t, err)
				assert.Contains(t, data, "GIT_COMMIT_HASH")
				assert.Equal(t, "0000000000000000000000000000000000000000", getString(t, data["GIT_COMMIT_HASH"]))
			},
		},
		{
			name: "with push event",
			env: map[string]string{
				"GITHUB_EVENT_NAME": "push",
				"GITHUB_EVENT_PATH": "/event.json",
			},
			files: map[string]string{
				"/event.json": string(pushPayload),
			},
			validate: func(t *testing.T, repo repo.GitRepo, data map[string]cue.Value) {
				require.NoError(t, err)
				assert.Contains(t, data, "GIT_COMMIT_HASH")
				assert.Equal(t, "0000000000000000000000000000000000000000", getString(t, data["GIT_COMMIT_HASH"]))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutils.NewNoopLogger()

			repo := testutils.NewTestRepo(t)
			err := repo.WriteFile("example.txt", []byte("example content"))
			require.NoError(t, err)

			_, err = repo.Commit("Initial commit")
			require.NoError(t, err)

			if len(tt.env) > 0 {
				for k, v := range tt.env {
					require.NoError(t, os.Setenv(k, v))
					defer os.Unsetenv(k)
				}
			}

			fs := billy.NewInMemoryFs()
			if len(tt.files) > 0 {
				testutils.SetupFS(t, fs, tt.files)
			}

			project := &Project{
				ctx:          ctx,
				RawBlueprint: blueprint.NewRawBlueprint(ctx.CompileString("{}")),
				Repo:         &repo,
				Tag:          tt.tag,
				logger:       logger,
			}

			runtime := NewCustomGitRuntime(fs, logger)
			data := runtime.Load(project)
			tt.validate(t, repo, data)
		})
	}
}

func getString(t *testing.T, v cue.Value) string {
	s, err := v.String()
	require.NoError(t, err)
	return s
}
