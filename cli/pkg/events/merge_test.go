package events

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	s "github.com/input-output-hk/catalyst-forge/lib/schema"
	sg "github.com/input-output-hk/catalyst-forge/lib/schema/global"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeEventFiring(t *testing.T) {
	tests := []struct {
		name          string
		branch        string
		defaultBranch string
		configBranch  string
		expected      bool
	}{
		{
			name:          "firing on default",
			branch:        "main",
			defaultBranch: "main",
			expected:      true,
		},
		{
			name:          "firing on config",
			branch:        "test",
			defaultBranch: "",
			configBranch:  "test",
			expected:      true,
		},
		{
			name:          "not firing",
			branch:        "main",
			defaultBranch: "develop",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := testutils.NewTestRepo(t)

			err := repo.WriteFile("file.txt", []byte("content"))
			require.NoError(t, err)

			_, err = repo.Commit("Initial commit")
			require.NoError(t, err)

			err = repo.NewBranch(tt.branch)
			require.NoError(t, err)

			project := project.Project{
				Blueprint: s.Blueprint{
					Global: &sg.Global{
						Repo: &sg.Repo{
							DefaultBranch: tt.defaultBranch,
						},
					},
				},
				Repo: &repo,
			}

			event := MergeEvent{
				logger: testutils.NewNoopLogger(),
			}

			ctx := cuecontext.New()
			var config cue.Value
			if tt.configBranch != "" {
				config = ctx.CompileString(fmt.Sprintf(`{branch: "%s"}`, tt.configBranch))
			}

			firing, err := event.Firing(&project, config)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, firing)
		})
	}
}
