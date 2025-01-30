package events

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/input-output-hk/catalyst-forge/lib/project/secrets"
	"github.com/input-output-hk/catalyst-forge/lib/tools/testutils"
	"github.com/stretchr/testify/assert"
)

func TestTagEventFiring(t *testing.T) {
	tests := []struct {
		name        string
		tag         *project.ProjectTag
		projectPath string
		gitRoot     string
		expected    bool
		expectErr   bool
	}{
		{
			name: "firing",
			tag: &project.ProjectTag{
				Full:    "test/v1.0.0",
				Project: "test",
				Version: "v1.0.0",
			},
			expected:  true,
			expectErr: false,
		},
		{
			name:      "not firing",
			tag:       nil,
			expected:  false,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := project.NewProject(
				nil,
				nil,
				nil,
				"test",
				"",
				"",
				schema.Blueprint{},
				tt.tag,
				testutils.NewNoopLogger(),
				secrets.SecretStore{},
			)

			event := TagEvent{
				logger: testutils.NewNoopLogger(),
			}

			firing, err := event.Firing(&project, cue.Value{})
			if testutils.AssertError(t, err, tt.expectErr, "") {
				return
			}

			assert.Equal(t, tt.expected, firing)
		})
	}
}
