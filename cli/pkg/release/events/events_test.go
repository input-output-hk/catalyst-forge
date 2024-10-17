package events

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/project/blueprint"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/project/schema"
	"github.com/stretchr/testify/assert"
)

func TestDefaultEventHandlerFiring(t *testing.T) {
	tests := []struct {
		name        string
		store       map[EventType]ReleaseEvent
		events      []string
		expected    bool
		expectErr   bool
		expectedErr error
	}{
		{
			name: "firing",
			store: map[EventType]ReleaseEvent{
				"merge": newReleaseEventMock(true, nil),
			},
			events:   []string{"merge"},
			expected: true,
		},
		{
			name: "multiple events",
			store: map[EventType]ReleaseEvent{
				"merge": newReleaseEventMock(true, nil),
				"tag":   newReleaseEventMock(false, nil),
			},
			events:   []string{"merge", "tag"},
			expected: true,
		},
		{
			name: "not firing",
			store: map[EventType]ReleaseEvent{
				"merge": newReleaseEventMock(false, nil),
			},
			events:   []string{"merge"},
			expected: false,
		},
		{
			name: "failing",
			store: map[EventType]ReleaseEvent{
				"merge": newReleaseEventMock(false, fmt.Errorf("error")),
			},
			events:   []string{"merge"},
			expected: false,
		},
		{
			name: "unknown event",
			store: map[EventType]ReleaseEvent{
				"merge": newReleaseEventMock(false, nil),
			},
			events:   []string{"tag"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := DefaultReleaseEventHandler{
				logger: testutils.NewNoopLogger(),
				store:  tt.store,
			}

			on := make(map[string]any)
			for _, event := range tt.events {
				on[event] = nil
			}
			p := project.Project{
				Blueprint: schema.Blueprint{
					Project: schema.Project{
						Release: map[string]schema.Release{
							"release": {
								On: on,
							},
						},
					},
				},
				RawBlueprint: blueprint.NewRawBlueprint(cue.Value{}),
			}
			actual := handler.Firing(&p, "release")
			assert.Equal(t, tt.expected, actual)
		})
	}
}

type mockReleaseEvent struct {
	firing bool
	err    error
}

func (m mockReleaseEvent) Firing(p *project.Project, config cue.Value) (bool, error) {
	return m.firing, m.err
}

func newReleaseEventMock(firing bool, err error) ReleaseEvent {
	return mockReleaseEvent{
		firing: firing,
		err:    err,
	}
}
