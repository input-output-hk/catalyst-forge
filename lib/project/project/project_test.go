package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectTagMatches(t *testing.T) {
	tests := []struct {
		name        string
		info        *TagInfo
		rootPath    string
		projectPath string
		expect      bool
	}{
		{
			name: "mono tag",
			info: &TagInfo{
				Generated: "generated",
				Git:       "project/v1.0.0",
			},
			rootPath:    "/repo",
			projectPath: "/repo/project",
			expect:      true,
		},
		{
			name: "plain tag",
			info: &TagInfo{
				Generated: "generated",
				Git:       "v1.0.0",
			},
			rootPath:    "/repo",
			projectPath: "/repo/project",
			expect:      true,
		},
		{
			name: "non-matching tag",
			info: &TagInfo{
				Generated: "generated",
				Git:       "project1/v1.0.0",
			},
			rootPath:    "/repo",
			projectPath: "/repo/project",
			expect:      false,
		},
		{
			name: "no tag",
			info: &TagInfo{
				Generated: "generated",
				Git:       "",
			},
			rootPath:    "/repo",
			projectPath: "/repo/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := Project{
				Path:     tt.projectPath,
				RepoRoot: tt.rootPath,
				TagInfo:  tt.info,
			}

			matches, err := project.TagMatches()
			require.NoError(t, err)
			assert.Equal(t, tt.expect, matches)
		})
	}
}
