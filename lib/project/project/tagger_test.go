package project

import (
	"fmt"
	"os"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaggerGetTagInfo(t *testing.T) {
	tests := []struct {
		name     string
		ci       bool
		skipTag  bool
		strategy string
		tag      string
		validate func(*testing.T, testutils.InMemRepo, TagInfo, error)
	}{
		{
			name:     "full",
			ci:       false,
			skipTag:  false,
			strategy: "commit",
			tag:      "v1.0.0",
			validate: func(t *testing.T, repo testutils.InMemRepo, info TagInfo, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "v1.0.0", string(info.Git))

				head, err := repo.Repo.Head()
				require.NoError(t, err)
				assert.Equal(t, head.Hash().String(), string(info.Generated))
			},
		},
		{
			name:     "full ci",
			ci:       true,
			skipTag:  true,
			strategy: "commit",
			tag:      "v1.0.0",
			validate: func(t *testing.T, repo testutils.InMemRepo, info TagInfo, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "v1.0.0", string(info.Git))

				head, err := repo.Repo.Head()
				require.NoError(t, err)
				assert.Equal(t, head.Hash().String(), string(info.Generated))
			},
		},
		{
			name:     "no strategy",
			ci:       false,
			skipTag:  false,
			strategy: "",
			tag:      "v1.0.0",
			validate: func(t *testing.T, repo testutils.InMemRepo, info TagInfo, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "v1.0.0", string(info.Git))
				assert.Equal(t, "", string(info.Generated))
			},
		},
		{
			name:     "no tag",
			ci:       false,
			skipTag:  true,
			strategy: "commit",
			tag:      "",
			validate: func(t *testing.T, repo testutils.InMemRepo, info TagInfo, err error) {
				assert.NoError(t, err)
				assert.Equal(t, "", string(info.Git))

				head, err := repo.Repo.Head()
				require.NoError(t, err)
				assert.Equal(t, head.Hash().String(), string(info.Generated))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := initRepo(t, tt.tag, tt.skipTag)

			if tt.ci {
				_ = os.Setenv("GITHUB_ACTIONS", "1")
				_ = os.Setenv("GITHUB_REF", fmt.Sprintf("refs/tags/%s", tt.tag))

				defer func() {
					_ = os.Unsetenv("GITHUB_ACTIONS")
					_ = os.Unsetenv("GITHUB_REF")
				}()
			}

			var bp schema.Blueprint
			if tt.strategy != "" {
				bp = schema.Blueprint{
					Global: schema.Global{
						CI: schema.GlobalCI{
							Tagging: schema.Tagging{
								Strategy: schema.TagStrategy(tt.strategy),
							},
						},
					},
				}
			} else {
				bp = schema.Blueprint{}
			}

			project := Project{
				Blueprint: bp,
				Earthfile: nil,
				Name:      "test",
				Repo:      repo.Repo,
			}

			tagger := Tagger{
				logger:  testutils.NewNoopLogger(),
				project: &project,
			}

			got, err := tagger.GetTagInfo()
			tt.validate(t, repo, got, err)
		})
	}
}

func initRepo(t *testing.T, tagName string, skipTag bool) testutils.InMemRepo {
	repo := testutils.NewInMemRepo(t)
	repo.AddFile(t, "example.txt", "example content")
	commit := repo.Commit(t, "Initial commit")
	if !skipTag {
		_ = repo.Tag(t, commit, tagName, "Initial tag")
	}

	return repo
}
