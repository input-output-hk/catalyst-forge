package events

import (
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
)

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/handler.go . ReleaseEventHandler

// EventType represents a release event type.
type EventType string

const (
	MergeEventName EventType = "merge"
	TagEventName   EventType = "tag"
)

// ReleaseEvent represents a release event.
type ReleaseEvent interface {
	// Firing returns true if the release event is firing.
	Firing(p *project.Project, config cue.Value) (bool, error)
}

// ReleaseEventHandler handles release events.
type ReleaseEventHandler interface {
	// Firing returns true if any of the events are firing for the given release.
	Firing(p *project.Project, releaseName string) bool
}

// DefaultReleaseEventHandler is the default release event handler.
type DefaultReleaseEventHandler struct {
	logger *slog.Logger
	store  map[EventType]ReleaseEvent
}

// Fires returns true if any of the given release events are firing.
func (r *DefaultReleaseEventHandler) Firing(p *project.Project, releaseName string) bool {
	_, ok := p.Blueprint.Project.Release[releaseName]
	if !ok {
		r.logger.Error("unknown release", "release", releaseName)
		return false
	}

	for event, config := range p.GetReleaseEvents(releaseName) {
		r.logger.Debug("checking event", "event", event)
		event, ok := r.store[EventType(event)]
		if !ok {
			r.logger.Error("unknown event", "event", event)
			continue
		}

		firing, err := event.Firing(p, config)
		if err != nil {
			r.logger.Error("failed to check if event is firing", "error", err)
			continue
		}

		if firing {
			r.logger.Debug("event is firing", "event", event)
			return true
		}
	}

	return false
}

// NewDefaultReleaseEventHandler returns a new default release event handler.
func NewDefaultReleaseEventHandler(logger *slog.Logger) DefaultReleaseEventHandler {
	return DefaultReleaseEventHandler{
		logger: logger,
		store: map[EventType]ReleaseEvent{
			MergeEventName: &MergeEvent{
				logger: logger,
			},
			TagEventName: &TagEvent{
				logger: logger,
			},
		},
	}
}
