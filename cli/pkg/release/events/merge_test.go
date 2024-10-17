package events

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
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
			repo := testutils.NewInMemRepo(t)
			repo.AddFile(t, "file.txt", "content")
			repo.Commit(t, "Initial commit")

			repo.NewBranch(t, tt.branch)

			project := project.Project{
				Blueprint: schema.Blueprint{
					Global: schema.Global{
						Repo: schema.GlobalRepo{
							DefaultBranch: tt.defaultBranch,
						},
					},
				},
				Repo: repo.Repo,
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
