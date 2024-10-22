package project

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitRuntimeLoad(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name     string
		tag      *ProjectTag
		validate func(*testing.T, *git.Repository, *plumbing.Hash, map[string]cue.Value)
	}{
		{
			name: "with tag",
			tag: &ProjectTag{
				Full:    "project/v1.0.0",
				Project: "project",
				Version: "v1.0.0",
			},
			validate: func(t *testing.T, repo *git.Repository, hash *plumbing.Hash, data map[string]cue.Value) {
				assert.Contains(t, data, "GIT_COMMIT_HASH")
				assert.Equal(t, hash.String(), getString(t, data["GIT_COMMIT_HASH"]))

				assert.Contains(t, data, "GIT_TAG")
				assert.Equal(t, "project/v1.0.0", getString(t, data["GIT_TAG"]))

				assert.Contains(t, data, "GIT_TAG_VERSION")
				assert.Equal(t, "v1.0.0", getString(t, data["GIT_TAG_VERSION"]))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := testutils.NewNoopLogger()

			repo := testutils.NewInMemRepo(t)
			repo.AddFile(t, "example.txt", "example content")
			commit := repo.Commit(t, "Initial commit")

			project := &Project{
				ctx:          ctx,
				RawBlueprint: blueprint.NewRawBlueprint(ctx.CompileString("{}")),
				Repo:         repo.Repo,
				Tag:          tt.tag,
				logger:       logger,
			}

			runtime := NewGitRuntime(logger)
			data := runtime.Load(project)
			tt.validate(t, repo.Repo, &commit, data)
		})
	}
}

func getString(t *testing.T, v cue.Value) string {
	s, err := v.String()
	require.NoError(t, err)
	return s
}
