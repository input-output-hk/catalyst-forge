package events

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/cli/internal/testutils"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/stretchr/testify/assert"
)

func TestDefaultEventHandlerFiring(t *testing.T) {
	tests := []struct {
		name        string
		store       map[EventType]Event
		events      []string
		expected    bool
		expectErr   bool
		expectedErr error
	}{
		{
			name: "firing",
			store: map[EventType]Event{
				"merge": newEventMock(true, nil),
			},
			events:   []string{"merge"},
			expected: true,
		},
		{
			name: "multiple events",
			store: map[EventType]Event{
				"merge": newEventMock(true, nil),
				"tag":   newEventMock(false, nil),
			},
			events:   []string{"merge", "tag"},
			expected: true,
		},
		{
			name: "not firing",
			store: map[EventType]Event{
				"merge": newEventMock(false, nil),
			},
			events:   []string{"merge"},
			expected: false,
		},
		{
			name: "failing",
			store: map[EventType]Event{
				"merge": newEventMock(false, fmt.Errorf("error")),
			},
			events:   []string{"merge"},
			expected: false,
		},
		{
			name: "unknown event",
			store: map[EventType]Event{
				"merge": newEventMock(false, nil),
			},
			events:   []string{"tag"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := DefaultEventHandler{
				logger: testutils.NewNoopLogger(),
				store:  tt.store,
			}

			events := make(map[string]cue.Value)
			for _, event := range tt.events {
				events[event] = cue.Value{}
			}

			actual := handler.Firing(&project.Project{}, events)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

type mockEvent struct {
	firing bool
	err    error
}

func (m mockEvent) Firing(p *project.Project, config cue.Value) (bool, error) {
	return m.firing, m.err
}

func newEventMock(firing bool, err error) Event {
	return mockEvent{
		firing: firing,
		err:    err,
	}
}
