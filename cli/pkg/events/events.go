package events

import (
	"log/slog"

	"cuelang.org/go/cue"
	"github.com/input-output-hk/catalyst-forge/lib/project/project"
	"github.com/input-output-hk/catalyst-forge/lib/providers/github"
)

//go:generate go run github.com/matryer/moq@latest -pkg mocks -out mocks/handler.go . EventHandler

// EventType represents a CI event type.
type EventType string

const (
	AlwaysEventName EventType = "always"
	MergeEventName  EventType = "merge"
	PREventName     EventType = "pr"
	TagEventName    EventType = "tag"
)

// Event represents a CI event.
type Event interface {
	// Firing returns true if the event is firing.
	Firing(p *project.Project, config cue.Value) (bool, error)
}

// EventHandler handles CI events.
type EventHandler interface {
	// Firing returns true if any of the events are firing for the given events.
	Firing(p *project.Project, events map[string]cue.Value) bool
}

// DefaultEventHandler is the default CI event handler.
type DefaultEventHandler struct {
	logger *slog.Logger
	store  map[EventType]Event
}

// Fires returns true if any of the given events are firing.
func (r *DefaultEventHandler) Firing(p *project.Project, events map[string]cue.Value) bool {
	for eventName, config := range events {
		r.logger.Debug("checking event", "event", eventName)
		event, ok := r.store[EventType(eventName)]
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
			r.logger.Debug("event is firing", "event", eventName)
			return true
		}
	}

	return false
}

// NewDefaultEventHandler returns a new default event handler.
func NewDefaultEventHandler(logger *slog.Logger) DefaultEventHandler {
	// This should never error
	gc, err := github.NewDefaultGithubClient("", "")
	if err != nil {
		panic(err)
	}

	return DefaultEventHandler{
		logger: logger,
		store: map[EventType]Event{
			AlwaysEventName: &AlwaysEvent{},
			MergeEventName: &MergeEvent{
				logger: logger,
			},
			PREventName: &PREvent{
				gc: gc,
			},
			TagEventName: &TagEvent{
				logger: logger,
			},
		},
	}
}
