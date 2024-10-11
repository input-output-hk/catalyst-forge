package events

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestTagEventFiring(t *testing.T) {
	tests := []struct {
		name        string
		tagInfo     *project.TagInfo
		projectPath string
		gitRoot     string
		expected    bool
		expectErr   bool
	}{
		{
			name: "firing",
			tagInfo: &project.TagInfo{
				Git: "v1.0.0",
			},
			expected:  true,
			expectErr: false,
		},
		{
			name: "not firing",
			tagInfo: &project.TagInfo{
				Git: "foo/v1.0.0",
			},
			expected:  false,
			expectErr: false,
		},
		{
			name: "no git tag",
			tagInfo: &project.TagInfo{
				Git: "",
			},
			expected:  false,
			expectErr: false,
		},
		{
			name:      "no tag info",
			tagInfo:   nil,
			expected:  false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := project.NewProject(
				testutils.NewNoopLogger(),
				nil,
				nil,
				nil,
				"test",
				tt.projectPath,
				tt.gitRoot,
				schema.Blueprint{},
				tt.tagInfo,
			)

			event := TagEvent{
				logger:  testutils.NewNoopLogger(),
				project: &project,
			}

			firing, err := event.Firing()
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			assert.Equal(t, tt.expected, firing)
		})
	}
}
