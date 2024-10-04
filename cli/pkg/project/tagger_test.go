package project

import (
	"os"
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestTaggerGetGitTag(t *testing.T) {
	tests := []struct {
		name        string
		trim        bool
		ci          bool
		aliases     map[string]string
		projectPath string
		repoRoot    string
		tag         string
		expect      string
	}{
		{
			name:        "simple tag",
			trim:        false,
			ci:          false,
			aliases:     map[string]string{},
			projectPath: "",
			repoRoot:    "",
			tag:         "v1.0.0",
			expect:      "v1.0.0",
		},
		{
			name:        "simple tag, ci",
			trim:        false,
			ci:          true,
			aliases:     map[string]string{},
			projectPath: "",
			repoRoot:    "",
			tag:         "v1.0.0",
			expect:      "v1.0.0",
		},
		{
			name:        "mono repo tag, matching",
			trim:        false,
			ci:          false,
			aliases:     map[string]string{},
			projectPath: "/test",
			repoRoot:    "/",
			tag:         "test/v1.0.0",
			expect:      "test/v1.0.0",
		},
		{
			name:        "mono repo tag, matching, trimmed",
			trim:        true,
			ci:          false,
			aliases:     map[string]string{},
			projectPath: "/test",
			repoRoot:    "/",
			tag:         "test/v1.0.0",
			expect:      "v1.0.0",
		},
		{
			name:        "mono repo tag, not matching",
			trim:        false,
			ci:          false,
			aliases:     map[string]string{},
			projectPath: "/test",
			repoRoot:    "/",
			tag:         "foo/v1.0.0",
			expect:      "",
		},
		{
			name: "mono repo tag, aliased",
			trim: false,
			ci:   false,
			aliases: map[string]string{
				"foo": "test/dir",
			},
			projectPath: "/test/dir",
			repoRoot:    "/",
			tag:         "foo/v1.0.0",
			expect:      "foo/v1.0.0",
		},
		{
			name: "mono repo tag, aliased, trimmed",
			trim: true,
			ci:   false,
			aliases: map[string]string{
				"foo": "test/dir",
			},
			projectPath: "/test/dir",
			repoRoot:    "/",
			tag:         "foo/v1.0.0",
			expect:      "v1.0.0",
		},
		{
			name: "mono repo tag, aliased, not matching",
			trim: false,
			ci:   false,
			aliases: map[string]string{
				"foo": "test/dir",
			},
			projectPath: "/test/dir",
			repoRoot:    "/",
			tag:         "bar/v1.0.0",
			expect:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := initRepo(t, tt.tag)

			if tt.ci {
				_ = os.Setenv("GITHUB_REF", "refs/tags/"+tt.tag)
			}

			bp := schema.Blueprint{
				Global: schema.Global{
					CI: schema.GlobalCI{
						Tagging: schema.Tagging{
							Aliases: tt.aliases,
						},
					},
				},
			}

			project := Project{
				Blueprint: bp,
				Earthfile: nil,
				Name:      "test",
				Path:      tt.projectPath,
				Repo:      repo.Repo,
				RepoRoot:  tt.repoRoot,
			}

			tagger := Tagger{
				ci:      tt.ci,
				logger:  testutils.NewNoopLogger(),
				project: &project,
				trim:    tt.trim,
			}

			got, err := tagger.GetGitTag()
			assert.NoError(t, err, "failed to get git tag")
			assert.Equal(t, tt.expect, got)
		})
	}
}

func initRepo(t *testing.T, tagName string) testutils.InMemRepo {
	repo := testutils.NewInMemRepo(t)
	repo.AddFile(t, "example.txt", "example content")
	commit := repo.Commit(t, "Initial commit")
	_ = repo.Tag(t, commit, tagName, "Initial tag")

	return repo
}
