package project

import (
	"testing"

	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestParseProjectTag(t *testing.T) {
	tests := []struct {
		name      string
		gitTag    string
		expected  ProjectTag
		expectErr bool
	}{
		{
			name:   "simple",
			gitTag: "project/1.0.0",
			expected: ProjectTag{
				Full:    "project/1.0.0",
				Project: "project",
				Version: "1.0.0",
			},
		},
		{
			name:      "invalid",
			gitTag:    "project",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseProjectTag(tt.gitTag)
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			assert.Equal(t, tt.expected, actual)
		})
	}
}
