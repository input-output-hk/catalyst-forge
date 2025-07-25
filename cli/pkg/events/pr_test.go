package events

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/providers/github"
	gm "github.com/input-output-hk/catalyst-forge/lib/providers/github/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPREventFiring(t *testing.T) {
	tests := []struct {
		name     string
		inPR     bool
		expected bool
	}{
		{
			name:     "firing on pr",
			inPR:     true,
			expected: true,
		},
		{
			name:     "not firing on pr",
			inPR:     false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := gm.GithubEnvMock{
				IsPRFunc: func() bool {
					return tt.inPR
				},
			}
			gc := gm.GithubClientMock{
				EnvFunc: func() github.GithubEnv {
					return &em
				},
			}

			event := PREvent{
				gc: &gc,
			}
			firing, err := event.Firing(nil, cue.Value{})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, firing)
		})
	}
}
