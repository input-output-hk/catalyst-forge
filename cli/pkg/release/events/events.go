package events

import (
	"log/slog"

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
	Firing() (bool, error)
}

// ReleaseEventHandler handles release events.
type ReleaseEventHandler interface {
	// Firing returns true if any of the given release events are firing.
	Firing(events []string) bool
}

// DefaultReleaseEventHandler is the default release event handler.
type DefaultReleaseEventHandler struct {
	logger *slog.Logger
	store  map[EventType]ReleaseEvent
}

// Fires returns true if any of the given release events are firing.
func (r *DefaultReleaseEventHandler) Firing(events []string) bool {
	for _, event := range events {
		r.logger.Debug("checking release event", "event", event)
		releaseEvent, ok := r.store[EventType(event)]
		if !ok {
			r.logger.Error("unknown release event", "event", event)
			continue
		}

		firing, err := releaseEvent.Firing()
		if err != nil {
			r.logger.Error("failed to check if release event is firing", "error", err)
			continue
		}

		if firing {
			r.logger.Debug("release event is firing", "event", event)
			return true
		}
	}

	return false
}

// NewDefaultReleaseEventHandler returns a new default release event handler.
func NewDefaultReleaseEventHandler(project *project.Project, logger *slog.Logger) DefaultReleaseEventHandler {
	return DefaultReleaseEventHandler{
		logger: logger,
		store:  newEventStore(project, logger),
	}
}

// newEventStore returns a map of release events.
func newEventStore(project *project.Project, logger *slog.Logger) map[EventType]ReleaseEvent {
	return map[EventType]ReleaseEvent{
		MergeEventName: &MergeEvent{
			logger:  logger,
			project: project,
		},
		TagEventName: &TagEvent{
			logger:  logger,
			project: project,
		},
	}
}
