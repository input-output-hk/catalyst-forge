package events

import (
	"testing"

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
		expected      bool
	}{
		{
			name:          "firing",
			branch:        "main",
			defaultBranch: "main",
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
						CI: schema.GlobalCI{
							DefaultBranch: &tt.defaultBranch,
						},
					},
				},
				Repo: repo.Repo,
			}

			event := MergeEvent{
				logger:  testutils.NewNoopLogger(),
				project: &project,
			}

			firing, err := event.Firing()
			require.NoError(t, err)
			assert.Equal(t, tt.expected, firing)
		})
	}
}
